package pandora

import (
	"context"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	eth1Types "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)


var (
	logThreshold = 8

	backOffPeriod = 15 * time.Second

	ConnectionError = errors.New("Client not connected")
)

// Client defines a subset of methods conformed to by Catalyst RPC clients for
// producing catalyst block and insert catalyst block.
type PandoraService interface {
	// PrepareExecutableBlock calls pandora client to get executable block
	PrepareExecutableBlock(ctx context.Context, extraData *ExtraData, blsSignature []byte) (*eth1Types.Block, error)
	// InsertExecutableBlock inserts the executable block into pandora chain
	InsertExecutableBlock(ctx context.Context, executableBlock *eth1Types.Block, signature []byte) (bool, error)
	// GetCoinbaseAddress returns coinbase address of pandora chain
	GetCoinbaseAddress() common.Address

}

type RPCClient interface {
	Call(result interface{}, method string, args ...interface{}) error
}

type Service struct {
	connected 			  bool
	isRunning             bool
	processingLock        sync.RWMutex
	ctx                   context.Context
	cancel                context.CancelFunc
	endpoint              string
	rpcClient             RPCClient
	pandoraClient    	  *PandoraClient
	coinbase 			  common.Address
	runError              error
}


func New(ctx context.Context, cliCtx *cli.Context, ecdsAddr common.Address, endpoint string) *Service {
	ctx, cancel := context.WithCancel(ctx)
	_ = cancel // govet fix for lost cancel. Cancel is handled in service.Stop()

	log.WithFields(logrus.Fields{
		"endpoint": endpoint}).Info("Initializing pandora client")

	return &Service{
		ctx:              	ctx,
		cancel:           	cancel,
		endpoint:      	  	endpoint,
		coinbase:  	  		ecdsAddr,
	}
}


func (s *Service) Start() {
	log.WithField("endpoint", s.endpoint).Info("Starting pandora client service")
	// Exit early if eth1 endpoint is not set.
	if s.endpoint == "" {
		return
	}

	go func() {
		s.isRunning = true
		s.waitForConnection()
		if s.ctx.Err() != nil {
			log.Info("Context closed, exiting pandora client goroutine")
			return
		}
	}()
}


func (s *Service) Stop() error {
	if s.cancel != nil {
		defer s.cancel()
	}
	s.closeClients()
	return nil
}


func (s *Service) Status() error {
	// Service don't start
	if !s.isRunning {
		return nil
	}
	// get error from run function
	if s.runError != nil {
		return s.runError
	}
	return nil
}

func (s *Service) closeClients() {
	rpcClient, ok := s.rpcClient.(*rpc.Client)
	if ok {
		rpcClient.Close()
	}

	if s.pandoraClient == nil {
		return
	}
	s.pandoraClient.Close()
}

func (s *Service) waitForConnection() {
	errConnect := s.connectToPandora()
	if errConnect == nil {
		s.connected = true
		s.runError = nil
		log.Info("Connected to pandora chain")
		return
	}

	if errConnect != nil {
		s.runError = errConnect
		log.WithError(errConnect).Error("Could not connect to pandora endpoint")
	}
	// Use a custom logger to only log errors
	// once in  a while.
	logCounter := 0
	errorLogger := func(err error, msg string) {
		if logCounter > logThreshold {
			log.WithError(err).Error(msg)
			logCounter = 0
		}
		logCounter++
	}

	ticker := time.NewTicker(backOffPeriod)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			log.Debugf("Trying to dial endpoint: %s", s.endpoint)
			errConnect := s.connectToPandora()
			if errConnect != nil {
				errorLogger(errConnect, "Could not connect to pandora endpoint")
				s.runError = errConnect
				s.connected = false
				continue
			}
			s.connected = true
			s.runError = nil
			log.WithFields(logrus.Fields{"endpoint": s.endpoint,}).Info("Connected to pandora chain")
			return
		case <-s.ctx.Done():
			log.Debug("Received cancelled context,closing existing orchestrator service")
			return
		}
	}
}

func (s *Service) connectToPandora() error {
	log.Info("Dialing to server")
	rpcClient, err := rpc.Dial(s.endpoint)
	if err != nil {
		return errors.Wrap(err, "could not dial node")
	}

	pandoraClient := NewClient(rpcClient)
	s.initializeConnection(pandoraClient, rpcClient)
	return nil
}


func (s *Service) initializeConnection(pandoraClient *PandoraClient, rpcClient *rpc.Client) {
	s.pandoraClient = pandoraClient
	s.rpcClient = rpcClient
}


func (s *Service) PrepareExecutableBlock(ctx context.Context, extraData *ExtraData,
	blsSignature []byte) (*eth1Types.Block, error) {

	if !s.connected {
		log.WithError(ConnectionError).Error("Failed to get orchestrator block")
		return nil, ConnectionError
	}

	requestParams := NewPrepareBlockRequest(extraData, blsSignature)
	response, err := s.pandoraClient.PrepareExecutableBlock(ctx, requestParams)
	if err != nil {
		log.WithError(err).Error("Pandora block preparation failed")
		return nil, err
	}
	log.WithField("executableBlock", response.ExecutableBlock).Info("Successfully prepared pandora block")
	return response.ExecutableBlock, nil
}


func (s *Service) InsertExecutableBlock(ctx context.Context, executableBlock *eth1Types.Block,
	signature []byte) (bool, error) {

	if !s.connected {
		log.WithError(ConnectionError).Error("Failed to get orchestrator block")
		return false, ConnectionError
	}

	requestParams := NewInsertBlockRequest(executableBlock, signature)
	response, err := s.pandoraClient.InsertExecutableBlock(ctx, requestParams)
	if err != nil || !response.Success {
		log.WithError(err).Error("Pandora block insertion failed")
		return false, err
	}

	log.WithField("sucess", response.Success).Info("Successfully prepared pandora block")
	return response.Success, nil
}


func (s *Service) GetCoinbaseAddress() common.Address {
	return s.coinbase
}