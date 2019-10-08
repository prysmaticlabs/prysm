package archiver

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/epoch"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
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
	go s.run(s.ctx)
}

// Stop the archiver service event loop.
func (s *Service) Stop() error {
	defer s.cancel()
	return nil
}

// Status reports the healthy status of the archiver. Returning nil means service
// is correctly running without error.
func (s *Service) Status() error {
	return nil
}

// We archive committee information pertaining to the head state's epoch.
func (s *Service) archiveCommitteeInfo(ctx context.Context, headState *pb.BeaconState) error {
	currentEpoch := helpers.SlotToEpoch(headState.Slot)
	committeeCount, err := helpers.CommitteeCount(headState, currentEpoch)
	if err != nil {
		return errors.Wrap(err, "could not get committee count")
	}
	seed, err := helpers.Seed(headState, currentEpoch)
	if err != nil {
		return errors.Wrap(err, "could not generate seed")
	}
	startShard, err := helpers.StartShard(headState, currentEpoch)
	if err != nil {
		return errors.Wrap(err, "could not get start shard")
	}
	proposerIndex, err := helpers.BeaconProposerIndex(headState)
	if err != nil {
		return errors.Wrap(err, "could not get beacon proposer index")
	}
	info := &ethpb.ArchivedCommitteeInfo{
		Seed:           seed[:],
		StartShard:     startShard,
		CommitteeCount: committeeCount,
		ProposerIndex:  proposerIndex,
	}
	if err := s.beaconDB.SaveArchivedCommitteeInfo(ctx, currentEpoch, info); err != nil {
		return errors.Wrap(err, "could not archive committee info")
	}
	return nil
}

// We archive active validator set changes that happened during the epoch.
func (s *Service) archiveActiveSetChanges(ctx context.Context, headState *pb.BeaconState) error {
	activations := validators.ActivatedValidatorIndices(headState)
	slashings := validators.SlashedValidatorIndices(headState)
	exited, err := validators.ExitedValidatorIndices(headState)
	if err != nil {
		return errors.Wrap(err, "could not determine exited validator indices")
	}
	activeSetChanges := &ethpb.ArchivedActiveSetChanges{
		Activated: activations,
		Exited:    exited,
		Slashed:   slashings,
	}
	if err := s.beaconDB.SaveArchivedActiveValidatorChanges(ctx, helpers.CurrentEpoch(headState), activeSetChanges); err != nil {
		return errors.Wrap(err, "could not archive active validator set changes")
	}
	return nil
}

// We compute participation metrics by first retrieving the head state and
// matching validator attestations during the epoch.
func (s *Service) archiveParticipation(ctx context.Context, headState *pb.BeaconState) error {
	participation, err := epoch.ComputeValidatorParticipation(headState)
	if err != nil {
		return errors.Wrap(err, "could not compute participation")
	}
	return s.beaconDB.SaveArchivedValidatorParticipation(ctx, helpers.SlotToEpoch(headState.Slot), participation)
}

// We archive validator balances and active indices.
func (s *Service) archiveBalances(ctx context.Context, headState *pb.BeaconState) error {
	balances := headState.Balances
	currentEpoch := helpers.CurrentEpoch(headState)
	if err := s.beaconDB.SaveArchivedBalances(ctx, currentEpoch, balances); err != nil {
		return errors.Wrap(err, "could not archive balances")
	}
	return nil
}

func (s *Service) run(ctx context.Context) {
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
			if err := s.archiveCommitteeInfo(ctx, headState); err != nil {
				log.WithError(err).Error("Could not archive committee info")
				continue
			}
			if err := s.archiveActiveSetChanges(ctx, headState); err != nil {
				log.WithError(err).Error("Could not archive active validator set changes")
				continue
			}
			if err := s.archiveParticipation(ctx, headState); err != nil {
				log.WithError(err).Error("Could not archive validator participation")
				continue
			}
			if err := s.archiveBalances(ctx, headState); err != nil {
				log.WithError(err).Error("Could not archive validator balances and active indices")
				continue
			}
			log.WithField(
				"epoch",
				helpers.CurrentEpoch(headState),
			).Debug("Successfully archived beacon chain data during epoch")
		case <-s.ctx.Done():
			log.Debug("Context closed, exiting goroutine")
			return
		case err := <-sub.Err():
			log.WithError(err).Error("Subscription to new chain head notifier failed")
			return
		}
	}
}
