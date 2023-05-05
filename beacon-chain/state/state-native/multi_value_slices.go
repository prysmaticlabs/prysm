package state_native

import (
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	multi_value_slice "github.com/prysmaticlabs/prysm/v4/container/multi-value-slice"
)

type MultiValueRandaoMixes = multi_value_slice.Slice[[32]byte, *BeaconState]

func NewMultiValueRandaoMixes(mixes [][]byte) *MultiValueRandaoMixes {
	items := make([]*multi_value_slice.MultiValue[[32]byte], fieldparams.RandaoMixesLength)
	for i, v := range mixes {
		items[i] = &multi_value_slice.MultiValue[[32]byte]{Shared: *(*[32]byte)(v), Individual: nil}
	}
	return &MultiValueRandaoMixes{
		Items: items,
	}
}

type MultiValueBlockRoots = multi_value_slice.Slice[[32]byte, *BeaconState]

func NewMultiValueBlockRoots(roots [][]byte) *MultiValueBlockRoots {
	items := make([]*multi_value_slice.MultiValue[[32]byte], fieldparams.BlockRootsLength)
	for i, v := range roots {
		items[i] = &multi_value_slice.MultiValue[[32]byte]{Shared: *(*[32]byte)(v), Individual: nil}
	}
	return &MultiValueBlockRoots{
		Items: items,
	}
}

type MultiValueStateRoots = multi_value_slice.Slice[[32]byte, *BeaconState]

func NewMultiValueStateRoots(roots [][]byte) *MultiValueStateRoots {
	items := make([]*multi_value_slice.MultiValue[[32]byte], fieldparams.StateRootsLength)
	for i, v := range roots {
		items[i] = &multi_value_slice.MultiValue[[32]byte]{Shared: *(*[32]byte)(v), Individual: nil}
	}
	return &MultiValueStateRoots{
		Items: items,
	}
}

type MultiValueBalances = multi_value_slice.Slice[uint64, *BeaconState]

func NewMultiValueBalances(balances []uint64) *MultiValueBalances {
	items := make([]*multi_value_slice.MultiValue[uint64], len(balances))
	for i, v := range balances {
		items[i] = &multi_value_slice.MultiValue[uint64]{Shared: v, Individual: nil}
	}
	return &MultiValueBalances{
		Items: items,
	}
}

type MultiValueInactivityScores = multi_value_slice.Slice[uint64, *BeaconState]

func NewMultiValueInactivityScores(scores []uint64) *MultiValueInactivityScores {
	items := make([]*multi_value_slice.MultiValue[uint64], len(scores))
	for i, v := range scores {
		items[i] = &multi_value_slice.MultiValue[uint64]{Shared: v, Individual: nil}
	}
	return &MultiValueInactivityScores{
		Items: items,
	}
}
