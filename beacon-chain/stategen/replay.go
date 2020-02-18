package stategen

import (
	"context"
	"errors"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	transition "github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
)

// ReplayBlocks replays the input blocks on the input state until the target slot is reached.
func (s *State) ReplayBlocks(ctx context.Context, state *state.BeaconState, signed []*ethpb.SignedBeaconBlock, targetSlot uint64) (*state.BeaconState, error) {
	var err error
	// The input block list is sorted in decreasing slots order.
	if len(signed) > 0 {
		for i := len(signed) - 1; i >= 0; i-- {
			state, err = transition.ExecuteStateTransitionNoVerifyAttSigs(ctx, state, signed[i])
			if err != nil {
				return nil, err
			}
		}
	}

	// If there is skip slots at the end.
	state, err = transition.ProcessSlots(ctx, state, targetSlot)
	if err != nil {
		return nil, err
	}

	return state, nil
}

// LoadBlocks loads the blocks between start slot and end slot by recursively fetching from end block root.
// The Blocks are returned in slot-descending order.
func (s *State) LoadBlocks(ctx context.Context, startSlot uint64, endSlot uint64, endBlockRoot [32]byte) ([]*ethpb.SignedBeaconBlock, error) {
	filter := filters.NewFilter().SetStartSlot(startSlot).SetEndSlot(endSlot)
	blocks, err := s.beaconDB.Blocks(ctx, filter)
	if err != nil {
		return nil, err
	}
	blockRoots, err := s.beaconDB.BlockRoots(ctx, filter)
	if err != nil {
		return nil, err
	}
	// The retrieved blocks and block roots have to be in the same length given same filter.
	if len(blocks) != len(blockRoots) {
		return nil, errors.New("length of blocks and roots don't match")
	}

	// The last retrieved block root has to match input end block root.
	// Covers the edge case if there's multiple blocks on the same end slot,
	// the end root may not be the last index in `blockRoots`.
	length := len(blocks)
	for length >= 3 && blocks[length-1].Block.Slot == blocks[length-2].Block.Slot && blockRoots[length-1] != endBlockRoot {
		length--
		if blockRoots[length-2] == endBlockRoot {
			length--
			break
		}
	}

	if blockRoots[length-1] != endBlockRoot {
		return nil, errors.New("end block roots don't match")
	}

	filteredBlocks := []*ethpb.SignedBeaconBlock{blocks[length-1]}
	// Starting from second to last index because the last block is already in the filtered block list.
	for i := length - 2; i >= 0; i-- {
		b := filteredBlocks[len(filteredBlocks)-1]
		if bytesutil.ToBytes32(b.Block.ParentRoot) != blockRoots[i] {
			continue
		}
		filteredBlocks = append(filteredBlocks, blocks[i])
	}

	return filteredBlocks, nil
}
