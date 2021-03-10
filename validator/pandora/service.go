package pandora

import (
	"context"
	"github.com/prysmaticlabs/prysm/shared/logutil"
	"sync"
	"time"

	eth1Types "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)


var (
	logThreshold = 8
	dialInterval = 15 * time.Second

	ConnectionError = errors.New("Client not connected")
	errNotSynced = errors.New("pandora node is still syncing")
)

// Client defines a subset of methods conformed to by Catalyst RPC clients for
// producing catalyst block and insert catalyst block.
type PandoraService interface {
	// PrepareExecutableBlock calls pandora client to get executable block
	PrepareExecutableBlock(ctx context.Context, extraData *ExtraData, blsSignature []byte) (*eth1Types.Block, error)
	// InsertExecutableBlock inserts the executable block into pandora chain
	InsertExecutableBlock(ctx context.Context, executableBlock *eth1Types.Block, signature []byte) (bool, error)
}

type RPCClient interface {
	Call(result interface{}, method string, args ...interface{}) error
}

type DialRPCFn func(endpoint string) (*PandoraClient, *rpc.Client, error)

type Service struct {
	connected 			  bool
	isRunning             bool
	processingLock        sync.RWMutex
	ctx                   context.Context
	cancel                context.CancelFunc
	endpoint              string
	rpcClient             RPCClient
	pandoraClient    	  *PandoraClient
	runError              error

	dialPandoraFn 		  DialRPCFn
}

func NewService(ctx context.Context, endpoint string, dialPandoraFn DialRPCFn) *Service {

	ctx, cancel := context.WithCancel(ctx)
	_ = cancel // govet fix for lost cancel. Cancel is handled in service.Stop()

	log.WithFields(logrus.Fields{
		"endpoint": endpoint}).Info("Initializing pandora client")

	return &Service{
		ctx:              	ctx,
		cancel:           	cancel,
		endpoint:      	  	endpoint,
		dialPandoraFn: 		dialPandoraFn,
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
 	s.cancel()
	log.Info("Stopping service")
	s.closeClient()
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

func (s *Service) closeClient() error {
	if s.pandoraClient != nil {
		return s.pandoraClient.Close()
	}
	return nil
}

func (s *Service) waitForConnection() {
	pandoraClient, rpcClient, errConnect := s.dialPandoraFn(s.endpoint)
	s.pandoraClient = pandoraClient
	s.rpcClient = rpcClient

	if errConnect == nil {
		synced, errSynced := s.isPandoraNodeSynced()
		// Resume if eth1 node is synced.
		if synced {
			s.connected = true
			s.runError = nil
			log.WithFields(logrus.Fields{"endpoint": logutil.MaskCredentialsLogging(s.endpoint),
			}).Info("Connected to pandora chain")
			return
		}
		if errSynced != nil {
			s.runError = errSynced
			log.WithError(errSynced).Error("Could not check sync status of pandora chain")
		}
	}
	if errConnect != nil {
		s.runError = errConnect
		log.WithError(errConnect).Error("Could not connect to pandora chain")
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

	ticker := time.NewTicker(dialInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			log.Debugf("Trying to dial endpoint: %s", s.endpoint)
			pandoraClient, rpcClient, errConnect := s.dialPandoraFn(s.endpoint)
			s.pandoraClient = pandoraClient
			s.rpcClient = rpcClient

			if errConnect != nil {
				errorLogger(errConnect, "Could not connect to pandora chain")
				s.runError = errConnect
				continue
			}

			synced, errSynced := s.isPandoraNodeSynced()
			if errSynced != nil {
				errorLogger(errSynced, "Could not check sync status of pandora chain")
				s.runError = errSynced
				continue
			}
			if synced {
				s.connected = true
				s.runError = nil
				log.WithFields(logrus.Fields{
					"endpoint": logutil.MaskCredentialsLogging(s.endpoint)}).
					Info("Connected to pandora chain")
				return
			}
			s.runError = errNotSynced
			log.Debug("Pandora node is currently syncing")
		case <-s.ctx.Done():
			log.Debug("Received cancelled context, closing existing pandora client service")
			return
		}
	}
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

// checks if the pandora node is healthy and ready to serve before
// fetching data from  it.
func (s *Service) isPandoraNodeSynced() (bool, error) {
	syncProg, err := s.pandoraClient.SyncProgress(s.ctx)
	if err != nil {
		return false, err
	}
	return syncProg == nil, nil
}