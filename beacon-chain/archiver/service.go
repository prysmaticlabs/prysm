package archiver

import (
	"context"

	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "archiver")

// Service defining archiver functionality for persisting checkpointed
// beacon chain information to a database backend for historical purposes.
type Service struct {
	ctx             context.Context
	cancel          context.CancelFunc
	beaconDB        db.Database
	newHeadNotifier blockchain.NewHeadNotifier
	newHeadSlotChan chan uint64
}

// Config options for the archiver service.
type Config struct {
	BeaconDB        db.Database
	NewHeadNotifier blockchain.NewHeadNotifier
}

// NewArchiverService initializes the service from configuration options.
func NewArchiverService(ctx context.Context, cfg *Config) *Service {
	ctx, cancel := context.WithCancel(ctx)
	return &Service{
		ctx:             ctx,
		cancel:          cancel,
		beaconDB:        cfg.BeaconDB,
		newHeadNotifier: cfg.NewHeadNotifier,
		newHeadSlotChan: make(chan uint64, 1),
	}
}

// Start the archiver service event loop.
func (s *Service) Start() {
	log.Info("Starting service")
	go s.run()
}

// Stop the archiver service event loop.
func (s *Service) Stop() error {
	defer s.cancel()
	log.Info("Stopping service")
	return nil
}

// Status reports the healthy status of the archiver. Returning nil means service
// is correctly running without error.
func (s *Service) Status() error {
	return nil
}

func (s *Service) archiveParticipation(slot uint64) {
	// We do the same logic as the RPC server does when computing
	// participation metrics below.
}

func (s *Service) run() {
	sub := s.newHeadNotifier.HeadUpdatedFeed().Subscribe(s.newHeadSlotChan)
	defer sub.Unsubscribe()
	for {
		select {
		case slot := <-s.newHeadSlotChan:
			log.WithField("headSlot", slot).Info("New chain head event")
			if helpers.IsEpochStart(slot) {
				s.archiveParticipation(slot)
			}
		case <-s.ctx.Done():
			log.Info("Context closed, exiting goroutine")
			return
		case err := <-sub.Err():
			log.WithError(err).Error("Subscription to new chain head notifier failed")
		}
	}
}
