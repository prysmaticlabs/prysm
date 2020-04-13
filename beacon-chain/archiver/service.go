package archiver

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "archiver")

// Service defining archiver functionality for persisting checkpointed
// beacon chain information to a database backend for historical purposes.
type Service struct {
	ctx                  context.Context
	cancel               context.CancelFunc
	beaconDB             db.NoHeadAccessDatabase
	headFetcher          blockchain.HeadFetcher
	participationFetcher blockchain.ParticipationFetcher
	stateNotifier        statefeed.Notifier
	lastArchivedEpoch    uint64
}

// Config options for the archiver service.
type Config struct {
	BeaconDB             db.NoHeadAccessDatabase
	HeadFetcher          blockchain.HeadFetcher
	ParticipationFetcher blockchain.ParticipationFetcher
	StateNotifier        statefeed.Notifier
}

// NewArchiverService initializes the service from configuration options.
func NewArchiverService(ctx context.Context, cfg *Config) *Service {
	ctx, cancel := context.WithCancel(ctx)
	return &Service{
		ctx:                  ctx,
		cancel:               cancel,
		beaconDB:             cfg.BeaconDB,
		headFetcher:          cfg.HeadFetcher,
		participationFetcher: cfg.ParticipationFetcher,
		stateNotifier:        cfg.StateNotifier,
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
func (s *Service) archiveCommitteeInfo(ctx context.Context, headState *state.BeaconState, epoch uint64) error {
	proposerSeed, err := helpers.Seed(headState, epoch, params.BeaconConfig().DomainBeaconProposer)
	if err != nil {
		return errors.Wrap(err, "could not generate seed")
	}
	attesterSeed, err := helpers.Seed(headState, epoch, params.BeaconConfig().DomainBeaconAttester)
	if err != nil {
		return errors.Wrap(err, "could not generate seed")
	}

	info := &pb.ArchivedCommitteeInfo{
		ProposerSeed: proposerSeed[:],
		AttesterSeed: attesterSeed[:],
	}
	if err := s.beaconDB.SaveArchivedCommitteeInfo(ctx, epoch, info); err != nil {
		return errors.Wrap(err, "could not archive committee info")
	}
	return nil
}

// We archive active validator set changes that happened during the previous epoch.
func (s *Service) archiveActiveSetChanges(ctx context.Context, headState *state.BeaconState, epoch uint64) error {
	prevEpoch := epoch - 1
	vals := headState.Validators()
	activations := validators.ActivatedValidatorIndices(prevEpoch, vals)
	slashings := validators.SlashedValidatorIndices(prevEpoch, vals)
	activeValidatorCount, err := helpers.ActiveValidatorCount(headState, prevEpoch)
	if err != nil {
		return errors.Wrap(err, "could not get active validator count")
	}
	exited, err := validators.ExitedValidatorIndices(prevEpoch, vals, activeValidatorCount)
	if err != nil {
		return errors.Wrap(err, "could not determine exited validator indices")
	}
	activeSetChanges := &pb.ArchivedActiveSetChanges{
		Activated: activations,
		Exited:    exited,
		Slashed:   slashings,
	}
	if err := s.beaconDB.SaveArchivedActiveValidatorChanges(ctx, prevEpoch, activeSetChanges); err != nil {
		return errors.Wrap(err, "could not archive active validator set changes")
	}
	return nil
}

// We compute participation metrics by first retrieving the head state and
// matching validator attestations during the epoch.
func (s *Service) archiveParticipation(ctx context.Context, epoch uint64) error {
	p := s.participationFetcher.Participation(epoch)
	participation := &ethpb.ValidatorParticipation{}
	if p != nil {
		participation = &ethpb.ValidatorParticipation{
			EligibleEther:           p.PrevEpoch,
			VotedEther:              p.PrevEpochTargetAttesters,
			GlobalParticipationRate: float32(p.PrevEpochTargetAttesters) / float32(p.PrevEpoch),
		}
	}
	return s.beaconDB.SaveArchivedValidatorParticipation(ctx, epoch, participation)
}

// We archive validator balances and active indices.
func (s *Service) archiveBalances(ctx context.Context, balances []uint64, epoch uint64) error {
	if err := s.beaconDB.SaveArchivedBalances(ctx, epoch, balances); err != nil {
		return errors.Wrap(err, "could not archive balances")
	}
	return nil
}

func (s *Service) run(ctx context.Context) {
	stateChannel := make(chan *feed.Event, 1)
	stateSub := s.stateNotifier.StateFeed().Subscribe(stateChannel)
	defer stateSub.Unsubscribe()
	for {
		select {
		case event := <-stateChannel:
			if event.Type == statefeed.BlockProcessed {
				data, ok := event.Data.(*statefeed.BlockProcessedData)
				if !ok {
					log.Error("Event feed data is not type *statefeed.BlockProcessedData")
					continue
				}
				log.WithField("headRoot", fmt.Sprintf("%#x", data.BlockRoot)).Debug("Received block processed event")
				headState, err := s.headFetcher.HeadState(ctx)
				if err != nil {
					log.WithError(err).Error("Head state is not available")
					continue
				}
				slot := headState.Slot()
				currentEpoch := helpers.SlotToEpoch(slot)
				if !helpers.IsEpochEnd(slot) && currentEpoch <= s.lastArchivedEpoch {
					continue
				}
				epochToArchive := currentEpoch
				if !helpers.IsEpochEnd(slot) {
					epochToArchive--
				}
				if err := s.archiveCommitteeInfo(ctx, headState, epochToArchive); err != nil {
					log.WithError(err).Error("Could not archive committee info")
					continue
				}
				if err := s.archiveActiveSetChanges(ctx, headState, epochToArchive); err != nil {
					log.WithError(err).Error("Could not archive active validator set changes")
					continue
				}
				if err := s.archiveParticipation(ctx, epochToArchive); err != nil {
					log.WithError(err).Error("Could not archive validator participation")
					continue
				}
				if err := s.archiveBalances(ctx, headState.Balances(), epochToArchive); err != nil {
					log.WithError(err).Error("Could not archive validator balances and active indices")
					continue
				}
				log.WithField(
					"epoch",
					epochToArchive,
				).Debug("Successfully archived beacon chain data during epoch")
				s.lastArchivedEpoch = epochToArchive
			}
		case <-s.ctx.Done():
			log.Debug("Context closed, exiting goroutine")
			return
		case err := <-stateSub.Err():
			log.WithError(err).Error("Subscription to state feed notifier failed")
			return
		}
	}
}
