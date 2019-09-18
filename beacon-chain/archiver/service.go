package archiver

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/epoch"
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
	headFetcher     blockchain.HeadFetcher
	newHeadNotifier blockchain.NewHeadNotifier
	newHeadSlotChan chan uint64
}

// Config options for the archiver service.
type Config struct {
	BeaconDB        db.Database
	HeadFetcher     blockchain.HeadFetcher
	NewHeadNotifier blockchain.NewHeadNotifier
}

// NewArchiverService initializes the service from configuration options.
func NewArchiverService(ctx context.Context, cfg *Config) *Service {
	ctx, cancel := context.WithCancel(ctx)
	return &Service{
		ctx:             ctx,
		cancel:          cancel,
		beaconDB:        cfg.BeaconDB,
		headFetcher:     cfg.HeadFetcher,
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

// We compute participation metrics by first retrieving the head state and
// matching validator attestations during the epoch.
func (s *Service) archiveParticipation(slot uint64) error {
	headState := s.headFetcher.HeadState()
	participation, err := epoch.ComputeValidatorParticipation(headState, slot)
	if err != nil {
		return errors.Wrap(err, "could not compute participation")
	}
	return s.beaconDB.SaveArchivedValidatorParticipation(s.ctx, helpers.SlotToEpoch(slot), participation)
}

func (s *Service) run() {
	sub := s.newHeadNotifier.HeadUpdatedFeed().Subscribe(s.newHeadSlotChan)
	defer sub.Unsubscribe()
	for {
		select {
		case slot := <-s.newHeadSlotChan:
			log.WithField("headSlot", slot).Info("New chain head event")
			if !helpers.IsEpochStart(slot) {
				continue
			}
			if err := s.archiveParticipation(slot); err != nil {
				log.WithError(err).Error("Could not archive validator participation")
			}
		case <-s.ctx.Done():
			log.Info("Context closed, exiting goroutine")
			return
		case err := <-sub.Err():
			log.WithError(err).Error("Subscription to new chain head notifier failed")
		}
	}
}
