package stategen

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	transition "github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"go.opencensus.io/trace"
)

// ReplayBlocks replays the input blocks on the input state until the target slot is reached.
func (s *State) ReplayBlocks(ctx context.Context, state *state.BeaconState, signed []*ethpb.SignedBeaconBlock, targetSlot uint64) (*state.BeaconState, error) {
	var err error
	// The input block list is sorted in decreasing slots order.
	if len(signed) > 0 {
		for i := len(signed) - 1; i >= 0; i-- {
			if featureconfig.Get().EnableStateGenSigVerify {
				state, err = transition.ExecuteStateTransition(ctx, state, signed[i])
				if err != nil {
					return nil, err
				}
			} else {
				state, err = executeStateTransitionStateGen(ctx, state, signed[i])
				if err != nil {
					return nil, err
				}
			}
		}
	}

	// If there is skip slots at the end.
	if featureconfig.Get().EnableStateGenSigVerify {
		state, err = transition.ProcessSlots(ctx, state, targetSlot)
		if err != nil {
			return nil, err
		}
	} else {
		state, err = processSlotsStateGen(ctx, state, targetSlot)
		if err != nil {
			return nil, err
		}
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

// executeStateTransitionStateGen applies state transition on input historical state and block for state gen usages.
// There's no signature verification involved given state gen only works with stored block and state in DB.
// If the objects are already in stored in DB, one can omit redundant signature checks and ssz hashing calculations.
// WARNING: This method should not be used on an unverified new block.
func executeStateTransitionStateGen(
	ctx context.Context,
	state *stateTrie.BeaconState,
	signed *ethpb.SignedBeaconBlock,
) (*stateTrie.BeaconState, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if signed == nil || signed.Block == nil {
		return nil, errors.New("nil block")
	}

	ctx, span := trace.StartSpan(ctx, "beacon-chain.ChainService.ExecuteStateTransitionStateGen")
	defer span.End()
	var err error

	// Execute per slots transition.
	// Given this is for state gen, a node uses the version process slots without skip slots cache.
	state, err = processSlotsStateGen(ctx, state, signed.Block.Slot)
	if err != nil {
		return nil, errors.Wrap(err, "could not process slot")
	}

	// Execute per block transition.
	// Given this is for state gen, a node only cares about the post state without proposer
	// and randao signature verifications.
	state, err = transition.ProcessBlockForStateRoot(ctx, state, signed)
	if err != nil {
		return nil, errors.Wrap(err, "could not process block")
	}

	return state, nil
}

// processSlotsStateGen to process old slots for state gen usages.
// There's no skip slot cache involved given state gen only works with already stored block and state in DB.
// WARNING: This method should not be used for future slot.
func processSlotsStateGen(ctx context.Context, state *stateTrie.BeaconState, slot uint64) (*stateTrie.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "beacon-chain.ChainService.ProcessSlotsStateGen")
	defer span.End()
	if state == nil {
		return nil, errors.New("nil state")
	}

	if state.Slot() > slot {
		err := fmt.Errorf("expected state.slot %d < slot %d", state.Slot(), slot)
		return nil, err
	}

	if state.Slot() == slot {
		return state, nil
	}

	for state.Slot() < slot {
		state, err := transition.ProcessSlot(ctx, state)
		if err != nil {
			return nil, errors.Wrap(err, "could not process slot")
		}
		if transition.CanProcessEpoch(state) {
			state, err = transition.ProcessEpochPrecompute(ctx, state)
			if err != nil {
				return nil, errors.Wrap(err, "could not process epoch with optimizations")
			}
		}
		state.SetSlot(state.Slot() + 1)
	}

	return state, nil
}
