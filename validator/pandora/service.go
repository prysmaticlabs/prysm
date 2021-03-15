package pandora

import (
	"context"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/prysmaticlabs/prysm/shared/logutil"
	"time"

	eth1Types "github.com/ethereum/go-ethereum/core/types"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var (
	logThreshold = 8
	dialInterval = 5 * time.Second

	ConnectionError = errors.New("Client not connected")
	errNotSynced    = errors.New("Pandora node is still syncing")
	errNoEndpoint   = errors.New("No endpoint defined")
)

type ExtraData struct {
	Slot            uint64
	Epoch           uint64
	ProposerIndex   uint64
}

// Client defines a subset of methods conformed to by Pandora RPC clients for
// producing catalyst block and insert pandora block.
type PandoraService interface {
	// GetWork gets the new block header and hash of pandora client
	GetWork(ctx context.Context) (*eth1Types.Header, *ExtraData, error)
	// SubmitWork submits the header hash and signature of pandora block header
	SubmitWork(ctx context.Context, blockNonce uint64, headerHash common.Hash, sig [32]byte) (bool, error)
}

type RPCClient interface {
	Call(result interface{}, method string, args ...interface{}) error
}

type DialRPCFn func(endpoint string) (*PandoraClient, error)

type Service struct {
	connected     bool
	isRunning     bool
	ctx           context.Context
	cancel        context.CancelFunc
	endpoint      string
	pandoraClient *PandoraClient
	runError      error
	dialPandoraFn DialRPCFn
}

func NewService(ctx context.Context, endpoint string, dialPandoraFn DialRPCFn) (*Service, error) {
	ctx, cancel := context.WithCancel(ctx)
	_ = cancel // govet fix for lost cancel. Cancel is handled in service.Stop()

	pandoraClient, err := dialPandoraFn(endpoint)
	if err != nil {
		log.WithError(err).Error("Pandora service initialization failed!")
		return nil, errors.Wrap(err, "Pandora service initialization failed!")
	}

	return &Service{
		ctx:           ctx,
		cancel:        cancel,
		endpoint:      endpoint,
		dialPandoraFn: dialPandoraFn,
		pandoraClient: pandoraClient,
	}, nil
}

func (s *Service) Start() {
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
	synced, errSynced := s.isPandoraNodeSynced()
	// Resume if eth1 node is synced.
	if synced {
		s.connected = true
		s.runError = nil
		log.WithFields(logrus.Fields{"endpoint": logutil.MaskCredentialsLogging(s.endpoint)}).Info("Connected to pandora chain")
		return
	}
	if errSynced != nil {
		s.runError = errSynced
		log.WithError(errSynced).Error("Could not check sync status of pandora chain")
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
		case <-s.ctx.Done():
			log.Debug("Received cancelled context, closing existing pandora client service")
			return
		}
	}
}

// GetPandoraBlock method calls pandora client's `eth_getWork` api and decode header and extra data fields
// This methods returns eth1Types.Header and ExtraData
func (s *Service) GetWork(ctx context.Context) (*eth1Types.Header, *ExtraData, error) {
	if !s.connected {
		log.WithError(ConnectionError).Error("Pandora chain is not connected")
		return nil, nil, ConnectionError
	}

	response, err := s.pandoraClient.GetWork(ctx)
	if err != nil {
		log.WithError(err).Error("Pandora block preparation failed")
		return nil, nil, err
	}
	header := response.Header
	var extraData ExtraData
	if err := rlp.DecodeBytes(header.Extra, &extraData); err != nil {
		return nil, nil, errors.Wrap(err, "Failed to decode extra data fields")
	}
	return header, &extraData, nil
}

// SubmitWork method calls pandora client's `eth_submitWork` api
// This method returns a boolean status
func (s *Service) SubmitWork(ctx context.Context, blockNonce uint64,
	headerHash common.Hash, sig [32]byte) (bool, error) {

	if !s.connected {
		log.WithError(ConnectionError).Error("Pandora chain is not connected")
		return false, ConnectionError
	}

	status, err := s.pandoraClient.SubmitWork(ctx, blockNonce, headerHash, sig)
	if err != nil || !status {
		log.WithError(err).Error("Work submission failed")
		return false, err
	}
	return status, nil
}

// checks if the pandora node is healthy and ready to serve before
// fetching data from  it.
func (s *Service) isPandoraNodeSynced() (bool, error) {
	syncProg, err := s.pandoraClient.SyncProgress(s.ctx)
	if err != nil {
		return false, err
	}
	return syncProg != nil, nil
}
