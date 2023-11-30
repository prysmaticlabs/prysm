package state_native

import (
	"runtime"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	multi_value_slice "github.com/prysmaticlabs/prysm/v4/container/multi-value-slice"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
)

var (
	multiValueRandaoMixesCountGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "multi_value_randao_mixes_count",
	})
	multiValueBlockRootsCountGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "multi_value_block_roots_count",
	})
	multiValueStateRootsCountGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "multi_value_state_roots_count",
	})
	multiValueBalancesCountGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "multi_value_balances_count",
	})
	multiValueValidatorsCountGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "multi_value_validators_count",
	})
	multiValueInactivityScoresCountGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "multi_value_inactivity_scores_count",
	})
)

// MultiValueRandaoMixes is a multi-value slice of randao mixes.
type MultiValueRandaoMixes = multi_value_slice.Slice[[32]byte]

// NewMultiValueRandaoMixes creates a new slice whose shared items will be populated with copies of input values.
func NewMultiValueRandaoMixes(mixes [][]byte) *MultiValueRandaoMixes {
	items := make([][32]byte, fieldparams.RandaoMixesLength)
	for i, v := range mixes {
		items[i] = [32]byte(bytesutil.PadTo(v, 32))
	}
	mv := &MultiValueRandaoMixes{}
	mv.Init(items)
	multiValueRandaoMixesCountGauge.Inc()
	runtime.SetFinalizer(mv, randaoMixesFinalizer)
	return mv
}

// MultiValueBlockRoots is a multi-value slice of block roots.
type MultiValueBlockRoots = multi_value_slice.Slice[[32]byte]

// NewMultiValueBlockRoots creates a new slice whose shared items will be populated with copies of input values.
func NewMultiValueBlockRoots(roots [][]byte) *MultiValueBlockRoots {
	items := make([][32]byte, fieldparams.BlockRootsLength)
	for i, v := range roots {
		items[i] = [32]byte(bytesutil.PadTo(v, 32))
	}
	mv := &MultiValueBlockRoots{}
	mv.Init(items)
	multiValueBlockRootsCountGauge.Inc()
	runtime.SetFinalizer(mv, blockRootsFinalizer)
	return mv
}

// MultiValueStateRoots is a multi-value slice of state roots.
type MultiValueStateRoots = multi_value_slice.Slice[[32]byte]

// NewMultiValueStateRoots creates a new slice whose shared items will be populated with copies of input values.
func NewMultiValueStateRoots(roots [][]byte) *MultiValueStateRoots {
	items := make([][32]byte, fieldparams.StateRootsLength)
	for i, v := range roots {
		items[i] = [32]byte(bytesutil.PadTo(v, 32))
	}
	mv := &MultiValueStateRoots{}
	mv.Init(items)
	multiValueStateRootsCountGauge.Inc()
	runtime.SetFinalizer(mv, stateRootsFinalizer)
	return mv
}

// MultiValueBalances is a multi-value slice of balances.
type MultiValueBalances = multi_value_slice.Slice[uint64]

// NewMultiValueBalances creates a new slice whose shared items will be populated with copies of input values.
func NewMultiValueBalances(balances []uint64) *MultiValueBalances {
	items := make([]uint64, len(balances))
	copy(items, balances)
	mv := &MultiValueBalances{}
	mv.Init(items)
	multiValueBalancesCountGauge.Inc()
	runtime.SetFinalizer(mv, balancesFinalizer)
	return mv
}

// MultiValueInactivityScores is a multi-value slice of inactivity scores.
type MultiValueInactivityScores = multi_value_slice.Slice[uint64]

// NewMultiValueInactivityScores creates a new slice whose shared items will be populated with copies of input values.
func NewMultiValueInactivityScores(scores []uint64) *MultiValueInactivityScores {
	items := make([]uint64, len(scores))
	copy(items, scores)
	mv := &MultiValueInactivityScores{}
	mv.Init(items)
	multiValueInactivityScoresCountGauge.Inc()
	runtime.SetFinalizer(mv, inactivityScoresFinalizer)
	return mv
}

// MultiValueValidators is a multi-value slice of validator references.
type MultiValueValidators = multi_value_slice.Slice[*ethpb.Validator]

// NewMultiValueValidators creates a new slice whose shared items will be populated with input values.
func NewMultiValueValidators(vals []*ethpb.Validator) *MultiValueValidators {
	mv := &MultiValueValidators{}
	mv.Init(vals)
	multiValueValidatorsCountGauge.Inc()
	runtime.SetFinalizer(mv, validatorsFinalizer)
	return mv
}

func randaoMixesFinalizer(m *MultiValueRandaoMixes) {
	multiValueRandaoMixesCountGauge.Dec()
}

func blockRootsFinalizer(m *MultiValueBlockRoots) {
	multiValueBlockRootsCountGauge.Dec()
}

func stateRootsFinalizer(m *MultiValueStateRoots) {
	multiValueStateRootsCountGauge.Dec()
}

func balancesFinalizer(m *MultiValueBalances) {
	multiValueBalancesCountGauge.Dec()
}

func validatorsFinalizer(m *MultiValueValidators) {
	multiValueValidatorsCountGauge.Dec()
}

func inactivityScoresFinalizer(m *MultiValueInactivityScores) {
	multiValueInactivityScoresCountGauge.Dec()
}
