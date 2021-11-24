package monitor

import (
	"sync"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
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
	StateGen stategen.StateManager
}

// Service is the main structure that tracks validators and reports logs and
// metrics of their performances throughout their lifetime.
type Service struct {
	config *ValidatorMonitorConfig

	// Locks access to TrackedValidators, latestPerformance, aggregatedPerformance,
	// trackedSyncedCommitteeIndices and lastSyncedEpoch
	sync.RWMutex

	trackedValidators           map[types.ValidatorIndex]interface{}
	latestPerformance           map[types.ValidatorIndex]ValidatorLatestPerformance
	aggregatedPerformance       map[types.ValidatorIndex]ValidatorAggregatedPerformance
	trackedSyncCommitteeIndices map[types.ValidatorIndex][]types.CommitteeIndex
	lastSyncedEpoch             types.Epoch
}

// TrackedIndex returns if the given validator index corresponds to one of the
// validators we follow.
// It assumes the caller holds the service lock.
func (s *Service) trackedIndex(idx types.ValidatorIndex) bool {
	_, ok := s.trackedValidators[idx]
	return ok
}
