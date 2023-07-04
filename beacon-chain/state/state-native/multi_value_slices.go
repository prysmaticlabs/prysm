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
	mv := &MultiValueRandaoMixes{}
	mv.Init(items)
	return mv
}

type MultiValueBlockRoots = multi_value_slice.Slice[[32]byte, *BeaconState]

func NewMultiValueBlockRoots(roots [][]byte) *MultiValueBlockRoots {
	items := make([][32]byte, fieldparams.BlockRootsLength)
	for i, v := range roots {
		items[i] = [32]byte(bytesutil.PadTo(v, 32))
	}
	mv := &MultiValueBlockRoots{}
	mv.Init(items)
	return mv
}

type MultiValueStateRoots = multi_value_slice.Slice[[32]byte, *BeaconState]

func NewMultiValueStateRoots(roots [][]byte) *MultiValueStateRoots {
	items := make([][32]byte, fieldparams.StateRootsLength)
	for i, v := range roots {
		items[i] = [32]byte(bytesutil.PadTo(v, 32))
	}
	mv := &MultiValueStateRoots{}
	mv.Init(items)
	return mv
}

type MultiValueBalances = multi_value_slice.Slice[uint64, *BeaconState]

func NewMultiValueBalances(balances []uint64) *MultiValueBalances {
	items := make([]uint64, len(balances))
	copy(items, balances)
	mv := &MultiValueBalances{}
	mv.Init(items)
	return mv
}

type MultiValueInactivityScores = multi_value_slice.Slice[uint64, *BeaconState]

func NewMultiValueInactivityScores(scores []uint64) *MultiValueInactivityScores {
	items := make([]uint64, len(scores))
	copy(items, scores)
	mv := &MultiValueInactivityScores{}
	mv.Init(items)
	return mv
}

type MultiValueValidators = multi_value_slice.Slice[*ethpb.Validator, *BeaconState]

func NewMultiValueValidators(vals []*ethpb.Validator) *MultiValueValidators {
	mv := &MultiValueValidators{}
	mv.Init(vals)
	return mv
}
