package monitor

import (
	"sync"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
	"github.com/prysmaticlabs/prysm/time/slots"
)

// ValidatorLatestPerformance keeps track of the latest participation of the validator
type ValidatorLatestPerformance struct {
	attestedSlot  types.Slot
	inclusionSlot types.Slot
	timelySource  bool
	timelyTarget  bool
	timelyHead    bool
	balance       uint64
	balanceChange int64
}

// ValidatorAggregatedPerformance keeps track of the accumulated performance of
// the validator since launch
type ValidatorAggregatedPerformance struct {
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
// monitor service tracks, as well as the event feed notifier that the
// monitor needs to subscribe.
type ValidatorMonitorConfig struct {
	StateGen          stategen.StateManager
	TrackedValidators map[types.ValidatorIndex]interface{}
}

// Service is the main structure that tracks validators and reports logs and
// metrics of their performances throughout their lifetime.
type Service struct {
	config *ValidatorMonitorConfig

	// monitorLock Locks access to TrackedValidators, latestPerformance, aggregatedPerformance,
	// trackedSyncedCommitteeIndices and lastSyncedEpoch
	monitorLock sync.RWMutex

	latestPerformance           map[types.ValidatorIndex]ValidatorLatestPerformance
	aggregatedPerformance       map[types.ValidatorIndex]ValidatorAggregatedPerformance
	trackedSyncCommitteeIndices map[types.ValidatorIndex][]types.CommitteeIndex
	lastSyncedEpoch             types.Epoch
}

// TrackedIndex returns if the given validator index corresponds to one of the
// validators we follow.
// It assumes the caller holds a Lock on the monitorLock
func (s *Service) trackedIndex(idx types.ValidatorIndex) bool {
	_, ok := s.config.TrackedValidators[idx]
	return ok
}

// updateSyncCommitteeTrackedVals updates the sync committee assignments of our
// tracked validators. It gets called when we sync a block after the Sync Period changes.
func (s *Service) updateSyncCommitteeTrackedVals(state state.BeaconState) {
	s.monitorLock.Lock()
	defer s.monitorLock.Unlock()
	for idx := range s.config.TrackedValidators {
		syncIdx, err := helpers.CurrentPeriodSyncSubcommitteeIndices(state, idx)
		if err != nil {
			log.WithError(err).WithField("ValidatorIndex", idx).Error(
				"Sync committee assignments will not be reported")
			delete(s.trackedSyncCommitteeIndices, idx)
		} else if len(syncIdx) == 0 {
			delete(s.trackedSyncCommitteeIndices, idx)
		} else {
			s.trackedSyncCommitteeIndices[idx] = syncIdx
		}
	}
	s.lastSyncedEpoch = slots.ToEpoch(state.Slot())
}
