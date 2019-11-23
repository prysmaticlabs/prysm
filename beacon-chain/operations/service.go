// Package operations defines the life-cycle of beacon block operations.
package operations

import (
	"context"
	"sync"

	"github.com/karlseguin/ccache"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	dbpb "github.com/prysmaticlabs/prysm/proto/beacon/db"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/event"
	handler "github.com/prysmaticlabs/prysm/shared/messagehandler"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// OperationFeeds inteface defines the informational feeds from the operations
// service.
type OperationFeeds interface {
	IncomingProcessedBlockFeed() *event.Feed
	Pool
}

// Service represents a service that handles the internal
// logic of beacon block operations.
type Service struct {
	ctx                        context.Context
	cancel                     context.CancelFunc
	beaconDB                   db.Database
	incomingProcessedBlockFeed *event.Feed
	incomingProcessedBlock     chan *ethpb.BeaconBlock
	error                      error
	attestationPool            map[[32]byte]*dbpb.AttestationContainer
	recentAttestationBitlist   *recentAttestationMultiMap
	attestationPoolLock        sync.RWMutex
	attestationLockCache       *ccache.Cache
}

// Config options for the service.
type Config struct {
	BeaconDB db.Database
}

// NewService instantiates a new operation service instance that will
// be registered into a running beacon node.
func NewService(ctx context.Context, cfg *Config) *Service {
	ctx, cancel := context.WithCancel(ctx)
	return &Service{
		ctx:                        ctx,
		cancel:                     cancel,
		beaconDB:                   cfg.BeaconDB,
		incomingProcessedBlockFeed: new(event.Feed),
		incomingProcessedBlock:     make(chan *ethpb.BeaconBlock, params.BeaconConfig().DefaultBufferSize),
		attestationPool:            make(map[[32]byte]*dbpb.AttestationContainer),
		recentAttestationBitlist:   newRecentAttestationMultiMap(),
		attestationLockCache:       ccache.New(ccache.Configure()),
	}
}

// Start an beacon block operation pool service's main event loop.
func (s *Service) Start() {
	go s.removeOperations()
}

// Stop the beacon block operation pool service's main event loop
// and associated goroutines.
func (s *Service) Stop() error {
	defer s.cancel()
	return nil
}

// Status returns the current service error if there's any.
func (s *Service) Status() error {
	if s.error != nil {
		return s.error
	}
	return nil
}

// removeOperations removes the processed operations from operation pool and DB.
func (s *Service) removeOperations() {
	incomingBlockSub := s.incomingProcessedBlockFeed.Subscribe(s.incomingProcessedBlock)
	defer incomingBlockSub.Unsubscribe()

	for {
		ctx := context.TODO()
		select {
		case err := <-incomingBlockSub.Err():
			log.WithError(err).Error("Subscription to incoming block sub failed")
			return
		case <-s.ctx.Done():
			log.Debug("Context closed, exiting goroutine")
			return
		// Listen for processed block from the block chain service.
		case block := <-s.incomingProcessedBlock:
			handler.SafelyHandleMessage(ctx, s.handleProcessedBlock, block)
		}
	}
}
