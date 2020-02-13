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

// This replays the input blocks on the input state until the target slot is reached.
func (s *Service) replayBlocks(ctx context.Context, state *state.BeaconState, signed []*ethpb.SignedBeaconBlock, targetSlot uint64) (*state.BeaconState, error) {
	var err error
	// The input block list is sorted in decreasing slots order.
	for i := len(signed) - 1; i >= 0; i-- {
		// If there is skip slot.
		for state.Slot() < signed[i].Block.Slot {
			state, err = transition.ProcessSlot(ctx, state)
			if err != nil {
				return nil, err
			}
		}
		state, err = transition.ProcessBlock(ctx, state, signed[i])
		if err != nil {
			return nil, err
		}
	}

	// If there is skip slots at the end.
	for state.Slot() < targetSlot {
		state, err = transition.ProcessSlot(ctx, state)
		if err != nil {
			return nil, err
		}
	}

	return state, nil
}

// This loads the blocks between start slot and end slot by recursively fetching from end block root.
// The Blocks are returned in slot-descending order.
func (s *Service) loadBlocks(ctx context.Context, startSlot uint64, endSlot uint64, endBlockRoot [32]byte) ([]*ethpb.SignedBeaconBlock, error) {
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
	length := len(blocks)
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
