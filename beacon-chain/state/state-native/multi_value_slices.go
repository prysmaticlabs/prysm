package state_native

import (
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	multi_value_slice "github.com/prysmaticlabs/prysm/v4/container/multi-value-slice"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
)

type MultiValueRandaoMixes = multi_value_slice.Slice[[32]byte, *BeaconState]

func NewMultiValueRandaoMixes(mixes [][]byte) *MultiValueRandaoMixes {
	items := make([][32]byte, fieldparams.RandaoMixesLength)
	for i, v := range mixes {
		items[i] = [32]byte(bytesutil.PadTo(v, 32))
	}
	return &MultiValueRandaoMixes{
		SharedItems:     items,
		IndividualItems: map[multi_value_slice.Id]*multi_value_slice.MultiValue[[32]byte]{},
		AppendedItems:   []*multi_value_slice.MultiValue[[32]byte]{},
	}
}

type MultiValueBlockRoots = multi_value_slice.Slice[[32]byte, *BeaconState]

func NewMultiValueBlockRoots(roots [][]byte) *MultiValueBlockRoots {
	items := make([][32]byte, fieldparams.BlockRootsLength)
	for i, v := range roots {
		items[i] = [32]byte(bytesutil.PadTo(v, 32))
	}
	return &MultiValueBlockRoots{
		SharedItems:     items,
		IndividualItems: map[multi_value_slice.Id]*multi_value_slice.MultiValue[[32]byte]{},
		AppendedItems:   []*multi_value_slice.MultiValue[[32]byte]{},
	}
}

type MultiValueStateRoots = multi_value_slice.Slice[[32]byte, *BeaconState]

func NewMultiValueStateRoots(roots [][]byte) *MultiValueStateRoots {
	items := make([][32]byte, fieldparams.StateRootsLength)
	for i, v := range roots {
		items[i] = [32]byte(bytesutil.PadTo(v, 32))
	}
	return &MultiValueStateRoots{
		SharedItems:     items,
		IndividualItems: map[multi_value_slice.Id]*multi_value_slice.MultiValue[[32]byte]{},
		AppendedItems:   []*multi_value_slice.MultiValue[[32]byte]{},
	}
}

type MultiValueBalances = multi_value_slice.Slice[uint64, *BeaconState]

func NewMultiValueBalances(balances []uint64) *MultiValueBalances {
	items := make([]uint64, len(balances))
	copy(items, balances)
	return &MultiValueBalances{
		SharedItems:     items,
		IndividualItems: map[multi_value_slice.Id]*multi_value_slice.MultiValue[uint64]{},
		AppendedItems:   []*multi_value_slice.MultiValue[uint64]{},
	}
}

type MultiValueInactivityScores = multi_value_slice.Slice[uint64, *BeaconState]

func NewMultiValueInactivityScores(balances []uint64) *MultiValueInactivityScores {
	items := make([]uint64, len(balances))
	copy(items, balances)
	return &MultiValueInactivityScores{
		SharedItems:     items,
		IndividualItems: map[multi_value_slice.Id]*multi_value_slice.MultiValue[uint64]{},
		AppendedItems:   []*multi_value_slice.MultiValue[uint64]{},
	}
}

type MultiValueValidators = multi_value_slice.Slice[*ethpb.Validator, *BeaconState]

func NewMultiValueValidators(vals []*ethpb.Validator) *MultiValueValidators {
	return &MultiValueValidators{
		SharedItems:     vals,
		IndividualItems: map[multi_value_slice.Id]*multi_value_slice.MultiValue[*ethpb.Validator]{},
		AppendedItems:   []*multi_value_slice.MultiValue[*ethpb.Validator]{},
	}
}
