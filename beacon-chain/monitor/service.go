package monitor

import (
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
	balanceChange uint64
}

// ValidatorAggregatedPerformance keeps track of the accumulated performance of
// the validator since launch
type ValidatorAggregatedPerformance struct {
	totalAttestedCount  uint64
	totalRequestedCount uint64
	totalDistance       uint64
	totalCorrectSource  uint64
	totalCorrectTarget  uint64
	totalCorrectHead    uint64
	totalProposedCount  uint64
	totalAggregations   uint64
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
	config                *ValidatorMonitorConfig
	latestPerformance     map[types.ValidatorIndex]ValidatorLatestPerformance
	aggregatedPerformance map[types.ValidatorIndex]ValidatorAggregatedPerformance
}

// TrackedIndex returns if the given validator index corresponds to one of the
// validators we follow
func (s *Service) TrackedIndex(idx types.ValidatorIndex) bool {
	_, ok := s.config.TrackedValidators[idx]
	return ok
}
