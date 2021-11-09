// Package validator-monitor defines a runtime service which receives
// notifications triggered by events related to performance of tracked
// validating keys. It then logs and emits metrics for a user to keep finely
// detailed performance measures.
package validator_monitor

import (
	"context"

	"github.com/pkg/errors"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed/operation"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
	"github.com/prysmaticlabs/prysm/time/slots"
	"github.com/sirupsen/logrus"
)

var (
	log = logrus.WithField("prefix", "validator-monitor")
	// TODO: The Prometheus gauge vectors and counters in this package deprecate the
	// corresponding gauge vectors and counters in the validator client.

	// inclusionSlotGauge used to track attestation inclusion distance
	inclusionSlotGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "validator-monitor",
			Name:      "inclusion_slot",
			Help:      "Attestations inclusion slot",
		},
		[]string{
			"validator_index",
		},
	)
	// timelyHeadCounter used to track attestation timely head flags
	timelyHeadCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "validator-monitor",
			Name:      "timely_head",
			Help:      "Attestation timely Head flag",
		},
		[]string{
			"validator_index",
		},
	)
	// timelyTargetCounter used to track attestation timely head flags
	timelyTargetCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "validator-monitor",
			Name:      "timely_head",
			Help:      "Attestation timely Target flag",
		},
		[]string{
			"validator_index",
		},
	)
	// timelySourceCounter used to track attestation timely head flags
	timelySourceCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "validator-monitor",
			Name:      "timely_head",
			Help:      "Attestation timely Source flag",
		},
		[]string{
			"validator_index",
		},
	)

	// proposedSlotsCounter used to track proposed blocks
	proposedSlotsCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "validator-monitor",
			Name:      "proposed_slots",
			Help:      "Number of proposed blocks included",
		},
		[]string{
			"validator_index",
		},
	)
	// aggregationCounter used to track aggregations
	aggregationCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "validator-monitor",
			Name:      "aggregations",
			Help:      "Number of aggregation duties performed",
		},
		[]string{
			"validator_index",
		},
	)
	// syncCommitteeContributionCounter used to track sync committee
	// contributions
	syncCommitteeContributionCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "validator-monitor",
			Name:      "sync_committee_contributions",
			Help:      "Number of Sync committee contributions performed",
		},
		[]string{
			"validator_index",
		},
	)
	// Error when event feed data is not statefeed.SyncedData
	errorNotSyncedData = errors.New("Event feed data is not of type *statefeed.SyncedData")

	// Error when the context is closed while waiting for sync
	errorContextClosedWhileWaiting = errors.New("Context closed while waiting for beacon to sync to latest Head")
)

// ValidatorLatestPerformance keeps track of the latest participation of the validator
type ValidatorLatestPerformance struct {
	attestedSlot  types.Slot
	inclusionSlot types.Slot
	timelySource  bool
	timelyTarget  bool
	timelyHead    bool
	timeStamp     uint64
	balance       uint64
	balanceChange uint64
}

// ValidatorAggregatedPerformance keeps track of the accumulated performance of
// the validator since launch
type ValidatorAggregatedPerformance struct {
	startEpoch                     types.Epoch
	startBalance                   uint64
	totalAttestedCount             uint64
	totalRequestedCount            uint64
	totalDistance                  uint64
	totalCorrectSource             uint64
	totalCorrectTarget             uint64
	totalCorrectHead               uint64
	totalProposedCount             uint64
	totalAggregations              uint64
	totalSyncComitteeContributions uint64
	totalSyncComitteeAggregations  uint64
}

// ValidatorMonitorConfig contains the list of validator indices that the
// validator-monitor service tracks, as well as the event feed notifier that the
// monitor needs to subscribe.
type ValidatorMonitorConfig struct {
	TrackedValidators   []types.ValidatorIndex
	StateNotifier       statefeed.Notifier
	AttestationNotifier operation.Notifier
	StateGen            stategen.StateManager
	HeadFetcher         blockchain.HeadFetcher
}

// Service is the main structure that tracks validators and reports logs and
// metrics of their performances throughout their lifetime.
type Service struct {
	config                      *ValidatorMonitorConfig
	ctx                         context.Context
	cancel                      context.CancelFunc
	latestPerformance           map[types.ValidatorIndex]ValidatorLatestPerformance
	aggregatedPerformance       map[types.ValidatorIndex]ValidatorAggregatedPerformance
	trackedSyncCommitteeIndices map[types.ValidatorIndex][]types.CommitteeIndex
}

// NewService sets up a new validator monitor instance when given a list of
// validator indices to track
func NewService(ctx context.Context, config *ValidatorMonitorConfig) (*Service, error) {
	ctx, cancel := context.WithCancel(ctx)
	r := &Service{
		config:                      config,
		ctx:                         ctx,
		cancel:                      cancel,
		latestPerformance:           make(map[types.ValidatorIndex]ValidatorLatestPerformance),
		aggregatedPerformance:       make(map[types.ValidatorIndex]ValidatorAggregatedPerformance),
		trackedSyncCommitteeIndices: make(map[types.ValidatorIndex][]types.CommitteeIndex),
	}
	log.WithFields(logrus.Fields{
		"ValidatorIndices": config.TrackedValidators,
	}).Info("Started service")

	if err := r.waitForSync(); err != nil {
		return nil, err
	}

	state, err := config.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not get head state")
	}
	epoch := slots.ToEpoch(state.Slot())

	for _, idx := range config.TrackedValidators {
		balance, err := state.BalanceAtIndex(idx)
		if err != nil {
			log.WithError(err).WithField("ValidatorIndex", idx).Error("Aggregated report will be wrong")
		}
		r.aggregatedPerformance[idx] = ValidatorAggregatedPerformance{
			startEpoch:   epoch,
			startBalance: balance,
		}
		r.latestPerformance[idx] = ValidatorLatestPerformance{
			balance: balance,
		}
		syncIdx, err := helpers.CurrentPeriodSyncSubcommitteeIndices(state, idx)
		if err != nil {
			log.WithError(err).WithField("ValidatorIndex", idx).Error(
				"Validator sync committee assignments will report wrong")
			r.trackedSyncCommitteeIndices[idx] = nil
		} else {
			r.trackedSyncCommitteeIndices[idx] = syncIdx
		}
	}
	go r.monitorRoutine()
	return r, nil
}

// waitForSync waits until the beacon node is synced to the latest head
func (s *Service) waitForSync() error {
	stateChannel := make(chan *feed.Event, 1)
	stateSub := s.config.StateNotifier.StateFeed().Subscribe(stateChannel)
	defer stateSub.Unsubscribe()
	for {
		select {
		case event := <-stateChannel:
			switch event.Type {
			case statefeed.Synced:
				_, ok := event.Data.(*statefeed.SyncedData)
				if !ok {
					return errorNotSyncedData
				}
				return nil
			}
		case <-s.ctx.Done():
			log.Debug("Context closed, exiting goroutine")
			return errorContextClosedWhileWaiting
		case err := <-stateSub.Err():
			log.WithError(err).Error("Could not subscribe to state notifier")
			return err
		}
	}
}

// monitorRoutine is the main dispatcher, it registers event channels for the
// state feed and the operation feed. It then calls the appropriate function
// when we get messages after syncing a block or processing attestations/sync
// committee contributions.
func (s *Service) monitorRoutine() {
	stateChannel := make(chan *feed.Event, 1)
	stateSub := s.config.StateNotifier.StateFeed().Subscribe(stateChannel)
	defer stateSub.Unsubscribe()

	opChannel := make(chan *feed.Event, 1)
	opSub := s.config.AttestationNotifier.OperationFeed().Subscribe(opChannel)
	defer opSub.Unsubscribe()

	for {
		select {
		case event := <-stateChannel:
			switch event.Type {
			case statefeed.BlockProcessed:
				data, ok := event.Data.(*statefeed.BlockProcessedData)
				if !ok {
					log.Error("Event feed data is not of type *statefeed.BlockProcessedData")
				} else if data.Verified {
					// We only process blocks that have been verified
					s.processBlock(data.SignedBlock)
				}
			}
		case event := <-opChannel:
			switch event.Type {
			case operation.UnaggregatedAttReceived:
				data, ok := event.Data.(*operation.UnAggregatedAttReceivedData)
				if !ok {
					log.Error("Event feed data is not of type *operation.UnAggregatedAttReceivedData")
				} else {
					s.processUnaggregatedAttestation(data.Attestation)
				}
			case operation.AggregatedAttReceived:
				data, ok := event.Data.(*operation.AggregatedAttReceivedData)
				if !ok {
					log.Error("Event feed data is not of type *operation.AggregatedAttReceivedData")
				} else {
					s.processAggregatedAttestation(data.Attestation)
				}
			case operation.ExitReceived:
				data, ok := event.Data.(*operation.ExitReceivedData)
				if !ok {
					log.Error("Event feed data is not of type *operation.ExitReceivedData")
				} else {
					s.processExit(data.Exit)
				}
			case operation.SyncCommitteeContributionReceived:
				data, ok := event.Data.(*operation.SyncCommitteeContributionReceivedData)
				if !ok {
					log.Error("Event feed data is not of type *operation.SyncCommitteeContributionReceivedData")
				} else {
					s.processSyncCommitteeContribution(data.Contribution)
				}
			}
		case <-s.ctx.Done():
			log.Debug("Context closed, exiting goroutine")
			return
		case err := <-stateSub.Err():
			log.WithError(err).Error("Could not subscribe to state notifier")
			return
		}
	}

}

// TrackedIndex returns if the given validator index corresponds to one of the
// validators we follow
func (s *Service) TrackedIndex(idx types.ValidatorIndex) bool {
	for _, tracked := range s.config.TrackedValidators {
		if tracked == idx {
			return true
		}
	}
	return false
}

// updateSyncCommitteeTrackedVals updates the sync committee assignments of our
// tracked validators. It gets called when we sync a block after the Sync Period changes.
func (s *Service) updateSyncCommitteeTrackedVals(state state.BeaconState) {
	for _, idx := range s.config.TrackedValidators {
		syncIdx, err := helpers.CurrentPeriodSyncSubcommitteeIndices(state, idx)
		if err != nil {
			log.WithError(err).WithField("ValidatorIndex", idx).Error(
				"Validator sync committee assignments will report wrong")
			s.trackedSyncCommitteeIndices[idx] = nil
		} else {
			s.trackedSyncCommitteeIndices[idx] = syncIdx
		}
	}
}
