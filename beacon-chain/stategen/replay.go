package stategen

import (
	"context"
	"errors"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	transition "github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"go.opencensus.io/trace"
)

// ReplayBlocks replays the input blocks on the input state until the target slot is reached.
func (s *State) ReplayBlocks(ctx context.Context, state *state.BeaconState, signed []*ethpb.SignedBeaconBlock, targetSlot uint64) (*state.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "stateGen.ReplayBlocks")
	defer span.End()

	var err error
	// The input block list is sorted in decreasing slots order.
	if len(signed) > 0 {
		for i := len(signed) - 1; i >= 0; i-- {
			if state.Slot() == targetSlot {
				break
			}
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
	ctx, span := trace.StartSpan(ctx, "stateGen.LoadBlocks")
	defer span.End()

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
	if length == 0 {
		return []*ethpb.SignedBeaconBlock{}, nil
	}

	if blockRoots[length-1] != endBlockRoot {
		return nil, errors.New("end block roots don't match")
	}

	filteredBlocks := []*ethpb.SignedBeaconBlock{blocks[length-1]}
	// Starting from second to last index because the last block is already in the filtered block list.
	if length >= 2 {
		for i := length - 2; i >= 0; i-- {
			b := filteredBlocks[len(filteredBlocks)-1]
			if bytesutil.ToBytes32(b.Block.ParentRoot) != blockRoots[i] {
				continue
			}
			filteredBlocks = append(filteredBlocks, blocks[i])
		}
	}

	return filteredBlocks, nil
}

// ComputeStateUpToSlot returns a processed state up to input target slot.
func (s *State) ComputeStateUpToSlot(ctx context.Context, targetSlot uint64) (*state.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "stateGen.ComputeStateUpToSlot")
	defer span.End()

	lastBlockRoot, lastBlockSlot, err := s.getLastValidBlock(ctx, targetSlot)
	if err != nil {
		return nil, err
	}
	lastBlockRootForState, err := s.getLastValidState(ctx, targetSlot)
	if err != nil {
		return nil, err
	}
	lastState, err := s.beaconDB.State(ctx, lastBlockRootForState)
	if err != nil {
		return nil, err
	}

	if lastState.Slot() == lastBlockSlot {
		return lastState, nil
	}

	blks, err := s.LoadBlocks(ctx, lastState.Slot()+1, lastBlockSlot, lastBlockRoot)
	if err != nil {
		return nil, err
	}
	lastState, err = s.ReplayBlocks(ctx, lastState, blks, targetSlot)
	if err != nil {
		return nil, err
	}

	return lastState, nil
}

// This finds the last valid block in DB from searching backwards starting at input slot,
// it returns the slot and the root of the block.
func (s *State) getLastValidBlock(ctx context.Context, slot uint64) ([32]byte, uint64, error) {
	ctx, span := trace.StartSpan(ctx, "stateGen.getLastValidBlock")
	defer span.End()

	filter := filters.NewFilter().SetStartSlot(0).SetEndSlot(slot)
	// We know the epoch boundary root will be the last index using the filter.
	rs, err := s.beaconDB.BlockRoots(ctx, filter)
	if err != nil {
		return [32]byte{}, 0, err
	}
	lastRoot := rs[len(rs)-1]

	b, err := s.beaconDB.Block(ctx, lastRoot)
	if err != nil {
		return [32]byte{}, 0, err
	}
	if b == nil || b.Block == nil {
		return [32]byte{}, 0, errors.New("last valid block can't be nil")
	}

	return lastRoot, b.Block.Slot, nil
}

// This finds the last valid state in DB from searching backwards starting at input slot,
// it returns the root of the block which used to produce the state.
func (s *State) getLastValidState(ctx context.Context, slot uint64) ([32]byte, error) {
	ctx, span := trace.StartSpan(ctx, "stateGen.getLastValidState")
	defer span.End()

	filter := filters.NewFilter().SetStartSlot(0).SetEndSlot(slot)
	// We know the epoch boundary root will be the last index using the filter.
	rs, err := s.beaconDB.BlockRoots(ctx, filter)
	if err != nil {
		return [32]byte{}, err
	}

	for i := len(rs) - 1; i >= 0; i-- {
		r := rs[i]
		if s.beaconDB.HasState(ctx, r) {
			return r, nil
		}
	}

	return [32]byte{}, errors.New("no valid state found")
}
