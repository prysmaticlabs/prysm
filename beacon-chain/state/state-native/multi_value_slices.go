package state_native

import (
	"runtime"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	multi_value_slice "github.com/prysmaticlabs/prysm/v4/container/multi-value-slice"
)

type MultiValueRandaoMixes = multi_value_slice.Slice[[32]byte, *BeaconState]

func NewMultiValueRandaoMixes(mixes [][]byte) *MultiValueRandaoMixes {
	items := make([][32]byte, fieldparams.RandaoMixesLength)
	for i, v := range mixes {
		items[i] = [32]byte(v)
	}
	return &MultiValueRandaoMixes{
		SharedItems:     items,
		IndividualItems: map[uint64]*multi_value_slice.MultiValue[[32]byte]{},
		AppendedItems:   []*multi_value_slice.MultiValue[[32]byte]{},
	}
}

type MultiValueBlockRoots = multi_value_slice.Slice[[32]byte, *BeaconState]

func NewMultiValueBlockRoots(roots [][]byte) *MultiValueBlockRoots {
	items := make([][32]byte, fieldparams.BlockRootsLength)
	for i, v := range roots {
		items[i] = [32]byte(v)
	}
	return &MultiValueBlockRoots{
		SharedItems:     items,
		IndividualItems: map[uint64]*multi_value_slice.MultiValue[[32]byte]{},
		AppendedItems:   []*multi_value_slice.MultiValue[[32]byte]{},
	}
}

type MultiValueStateRoots = multi_value_slice.Slice[[32]byte, *BeaconState]

func NewMultiValueStateRoots(roots [][]byte) *MultiValueStateRoots {
	items := make([][32]byte, fieldparams.StateRootsLength)
	for i, v := range roots {
		items[i] = [32]byte(v)
	}
	return &MultiValueStateRoots{
		SharedItems:     items,
		IndividualItems: map[uint64]*multi_value_slice.MultiValue[[32]byte]{},
		AppendedItems:   []*multi_value_slice.MultiValue[[32]byte]{},
	}
}

var (
	balancesCount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "balances_count",
	})
)

type MultiValueBalances = multi_value_slice.Slice[uint64, *BeaconState]

func NewMultiValueBalances(balances []uint64) *MultiValueBalances {
	items := make([]uint64, len(balances))
	copy(items, balances)
	b := &MultiValueBalances{
		SharedItems:     items,
		IndividualItems: map[uint64]*multi_value_slice.MultiValue[uint64]{},
		AppendedItems:   []*multi_value_slice.MultiValue[uint64]{},
	}

	balancesCount.Inc()
	runtime.SetFinalizer(b, balancesFinalizer)

	return b
}

func balancesFinalizer(b *MultiValueBalances) {
	balancesCount.Dec()
}
