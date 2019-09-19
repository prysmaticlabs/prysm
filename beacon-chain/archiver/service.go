package archiver

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/epoch"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
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
	newHeadRootChan chan [32]byte
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
		newHeadRootChan: make(chan [32]byte, 1),
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
func (s *Service) archiveParticipation(headState *pb.BeaconState) error {
	participation, err := epoch.ComputeValidatorParticipation(headState)
	if err != nil {
		return errors.Wrap(err, "could not compute participation")
	}
	return s.beaconDB.SaveArchivedValidatorParticipation(s.ctx, helpers.SlotToEpoch(headState.Slot), participation)
}

func (s *Service) run() {
	sub := s.newHeadNotifier.HeadUpdatedFeed().Subscribe(s.newHeadRootChan)
	defer sub.Unsubscribe()
	for {
		select {
		case r := <-s.newHeadRootChan:
			log.WithField("headRoot", fmt.Sprintf("%#x", r)).Debug("New chain head event")
			headState := s.headFetcher.HeadState()
			if !helpers.IsEpochEnd(headState.Slot) {
				continue
			}
			if err := s.archiveParticipation(headState); err != nil {
				log.WithError(err).Error("Could not archive validator participation")
			}
			log.WithField(
				"epoch",
				helpers.SlotToEpoch(headState.Slot),
			).Debug("Successfully archived validator participation during epoch")
		case <-s.ctx.Done():
			log.Info("Context closed, exiting goroutine")
			return
		case err := <-sub.Err():
			log.WithError(err).Error("Subscription to new chain head notifier failed")
		}
	}
}
