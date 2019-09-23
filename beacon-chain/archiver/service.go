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
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
	"github.com/prysmaticlabs/prysm/shared/params"
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
	go s.run(s.ctx)
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
	info := &ethpb.ArchivedCommitteeInfo{
		Seed:           seed[:],
		StartShard:     startShard,
		CommitteeCount: committeeCount,
	}
	if err := s.beaconDB.SaveArchivedCommitteeInfo(ctx, currentEpoch, info); err != nil {
		return errors.Wrap(err, "could not archive committee info")
	}
	return nil
}

// We archive active validator set changes that happened during the epoch.
func (s *Service) archiveActiveSetChanges(ctx context.Context, headState *pb.BeaconState) error {
	currentEpoch := helpers.CurrentEpoch(headState)
	activations := make([]uint64, 0)
	slashings := make([]uint64, 0)
	exited := make([]uint64, 0)
	exitEpochs := make([]uint64, 0)
	delayedActivationEpoch := helpers.DelayedActivationExitEpoch(currentEpoch)
	for i := 0; i < len(headState.Validators); i++ {
		val := headState.Validators[i]
		if val.ActivationEpoch == delayedActivationEpoch {
			activations = append(activations, uint64(i))
		}
		maxWithdrawableEpoch := mathutil.Max(val.WithdrawableEpoch, currentEpoch+params.BeaconConfig().EpochsPerSlashingsVector)
		if val.WithdrawableEpoch == maxWithdrawableEpoch && val.Slashed {
			slashings = append(slashings, uint64(i))
		}
		if val.ExitEpoch != params.BeaconConfig().FarFutureEpoch {
			exitEpochs = append(exitEpochs, val.ExitEpoch)
		}
	}
	exitQueueEpoch := uint64(0)
	for _, i := range exitEpochs {
		if exitQueueEpoch < i {
			exitQueueEpoch = i
		}
	}

	// We use the exit queue churn to determine if we have passed a churn limit.
	exitQueueChurn := 0
	for _, val := range headState.Validators {
		if val.ExitEpoch == exitQueueEpoch {
			exitQueueChurn++
		}
	}
	churn, err := helpers.ValidatorChurnLimit(headState)
	if err != nil {
		return errors.Wrap(err, "could not get churn limit")
	}

	if uint64(exitQueueChurn) >= churn {
		exitQueueEpoch++
	}
	withdrawableEpoch := exitQueueEpoch + params.BeaconConfig().MinValidatorWithdrawabilityDelay
	for i, val := range headState.Validators {
		if val.ExitEpoch == exitQueueEpoch && val.WithdrawableEpoch == withdrawableEpoch {
			exited = append(exited, uint64(i))
		}
	}

	activeSetChanges := &ethpb.ArchivedActiveSetChanges{
		Activated: activations,
		Exited:    exited,
		Slashed:   slashings,
	}
	if err := s.beaconDB.SaveArchivedActiveValidatorChanges(ctx, currentEpoch, activeSetChanges); err != nil {
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
func (s *Service) archiveBalancesAndIndices(ctx context.Context, headState *pb.BeaconState) error {
	balances := headState.Balances
	currentEpoch := helpers.CurrentEpoch(headState)
	activeIndices, err := helpers.ActiveValidatorIndices(headState, currentEpoch)
	if err != nil {
		return errors.Wrap(err, "could not determine active indices")
	}
	if err := s.beaconDB.SaveArchivedBalances(ctx, currentEpoch, balances); err != nil {
		return errors.Wrap(err, "could not archive balances")
	}
	if err := s.beaconDB.SaveArchivedActiveIndices(ctx, currentEpoch, activeIndices); err != nil {
		return errors.Wrap(err, "could not archive active indices")
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
			if err := s.archiveBalancesAndIndices(ctx, headState); err != nil {
				log.WithError(err).Error("Could not archive validator balances and active indices")
				continue
			}
			log.WithField(
				"epoch",
				helpers.CurrentEpoch(headState),
			).Debug("Successfully archived committee info during epoch")
			log.WithField(
				"epoch",
				helpers.CurrentEpoch(headState),
			).Debug("Successfully archived active validator set changes during epoch")
			log.WithField(
				"epoch",
				helpers.CurrentEpoch(headState),
			).Debug("Successfully archived validator balances and active indices during epoch")
			log.WithField(
				"epoch",
				helpers.CurrentEpoch(headState),
			).Debug("Successfully archived validator participation during epoch")
		case <-s.ctx.Done():
			log.Info("Context closed, exiting goroutine")
			return
		case err := <-sub.Err():
			log.WithError(err).Error("Subscription to new chain head notifier failed")
		}
	}
}
