package state_native

import (
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	multi_value_slice "github.com/prysmaticlabs/prysm/v4/container/multi-value-slice"
)

type MultiValueRandaoMixes = multi_value_slice.Slice[[32]byte, *BeaconState]

func NewMultiValueRandaoMixes(mixes [][]byte) *MultiValueRandaoMixes {
	items := make([]*multi_value_slice.MultiValue[[32]byte], fieldparams.RandaoMixesLength)
	for i, b := range mixes {
		items[i] = &multi_value_slice.MultiValue[[32]byte]{Shared: *(*[32]byte)(b), Individual: nil}
	}
	return &MultiValueRandaoMixes{
		Items: items,
	}
}

type MultiValueBlockRoots = multi_value_slice.Slice[[32]byte, *BeaconState]

func NewMultiValueBlockRoots(roots [][]byte) *MultiValueBlockRoots {
	items := make([]*multi_value_slice.MultiValue[[32]byte], fieldparams.BlockRootsLength)
	for i, b := range roots {
		items[i] = &multi_value_slice.MultiValue[[32]byte]{Shared: *(*[32]byte)(b), Individual: nil}
	}
	return &MultiValueBlockRoots{
		Items: items,
	}
}

type MultiValueStateRoots = multi_value_slice.Slice[[32]byte, *BeaconState]

func NewMultiValueStateRoots(roots [][]byte) *MultiValueStateRoots {
	items := make([]*multi_value_slice.MultiValue[[32]byte], fieldparams.StateRootsLength)
	for i, b := range roots {
		items[i] = &multi_value_slice.MultiValue[[32]byte]{Shared: *(*[32]byte)(b), Individual: nil}
	}
	return &MultiValueStateRoots{
		Items: items,
	}
}
