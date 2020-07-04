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
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
	"go.opencensus.io/trace"
)

// ComputeStateUpToSlot returns a processed state up to input target slot.
// If the last processed block is at slot 32, given input target slot at 40, this
// returns processed state up to slot 40 via empty slots.
// If there's duplicated blocks in a single slot, the canonical block will be returned.
func (s *State) ComputeStateUpToSlot(ctx context.Context, targetSlot uint64) (*state.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "stateGen.ComputeStateUpToSlot")
	defer span.End()

	// Return genesis state if target slot is 0.
	if targetSlot == 0 {
		return s.beaconDB.GenesisState(ctx)
	}

	lastBlockRoot, lastBlockSlot, err := s.lastSavedBlock(ctx, targetSlot)
	if err != nil {
		return nil, errors.Wrap(err, "could not get last saved block")
	}

	lastState, err := s.lastSavedState(ctx, targetSlot)
	if err != nil {
		return nil, errors.Wrap(err, "could not get last valid state")
	}
	if lastState == nil {
		return nil, errUnknownState
	}
	// Short circuit if no block was saved, replay using slots only.
	if lastBlockSlot == 0 {
		return s.ReplayBlocks(ctx, lastState, []*ethpb.SignedBeaconBlock{}, targetSlot)
	}

	// Return if the last valid state's slot is higher than the target slot.
	if lastState.Slot() >= targetSlot {
		return lastState, nil
	}

	blks, err := s.LoadBlocks(ctx, lastState.Slot()+1, lastBlockSlot, lastBlockRoot)
	if err != nil {
		return nil, errors.Wrap(err, "could not load blocks")
	}
	lastState, err = s.ReplayBlocks(ctx, lastState, blks, targetSlot)
	if err != nil {
		return nil, errors.Wrap(err, "could not replay blocks")
	}

	return lastState, nil
}

// ReplayBlocks replays the input blocks on the input state until the target slot is reached.
func (s *State) ReplayBlocks(ctx context.Context, state *state.BeaconState, signed []*ethpb.SignedBeaconBlock, targetSlot uint64) (*state.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "stateGen.ReplayBlocks")
	defer span.End()

	var err error
	// The input block list is sorted in decreasing slots order.
	if len(signed) > 0 {
		for i := len(signed) - 1; i >= 0; i-- {
			if state.Slot() >= targetSlot {
				break
			}

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
	if targetSlot > state.Slot() {
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
	// Return early if there's no block given the input.
	length := len(blocks)
	if length == 0 {
		return nil, nil
	}

	// The last retrieved block root has to match input end block root.
	// Covers the edge case if there's multiple blocks on the same end slot,
	// the end root may not be the last index in `blockRoots`.
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
		return nil, errUnknownBlock
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
		return nil, errUnknownState
	}

	if state.Slot() > slot {
		err := fmt.Errorf("expected state.slot %d < slot %d", state.Slot(), slot)
		return nil, err
	}

	if state.Slot() == slot {
		return state, nil
	}

	var err error
	for state.Slot() < slot {
		state, err = transition.ProcessSlot(ctx, state)
		if err != nil {
			return nil, errors.Wrap(err, "could not process slot")
		}
		if transition.CanProcessEpoch(state) {
			state, err = transition.ProcessEpochPrecompute(ctx, state)
			if err != nil {
				return nil, errors.Wrap(err, "could not process epoch with optimizations")
			}
		}
		if err := state.SetSlot(state.Slot() + 1); err != nil {
			return nil, err
		}
	}

	return state, nil
}

// This finds the last saved block in DB from searching backwards from input slot,
// it returns the block root and the slot of the block.
// This is used by both hot and cold state management.
func (s *State) lastSavedBlock(ctx context.Context, slot uint64) ([32]byte, uint64, error) {
	ctx, span := trace.StartSpan(ctx, "stateGen.lastSavedBlock")
	defer span.End()

	// Handle the genesis case where the input slot is 0.
	if slot == 0 {
		gRoot, err := s.genesisRoot(ctx)
		if err != nil {
			return [32]byte{}, 0, err
		}
		return gRoot, 0, nil
	}

	lastSaved, err := s.beaconDB.HighestSlotBlocksBelow(ctx, slot+1)
	if err != nil {
		return [32]byte{}, 0, err
	}

	// Given this is used to query canonical block. There should only be one saved canonical block of a given slot.
	if len(lastSaved) != 1 {
		return [32]byte{}, 0, fmt.Errorf("highest saved block does not equal to 1, it equals to %d", len(lastSaved))
	}
	if lastSaved[0] == nil || lastSaved[0].Block == nil {
		return [32]byte{}, 0, nil
	}
	r, err := stateutil.BlockRoot(lastSaved[0].Block)
	if err != nil {
		return [32]byte{}, 0, err
	}

	return r, lastSaved[0].Block.Slot, nil
}

// This finds the last saved state in DB from searching backwards from input slot,
// it returns the block root of the block which was used to produce the state.
// This is used by both hot and cold state management.
func (s *State) lastSavedState(ctx context.Context, slot uint64) (*state.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "stateGen.lastSavedState")
	defer span.End()

	// Handle the genesis case where the input slot is 0.
	if slot == 0 {
		return s.beaconDB.GenesisState(ctx)
	}

	lastSaved, err := s.beaconDB.HighestSlotStatesBelow(ctx, slot+1)
	if err != nil {
		return nil, err
	}

	// Given this is used to query canonical state. There should only be one saved canonical block of a given slot.
	if len(lastSaved) != 1 {
		return nil, fmt.Errorf("highest saved state does not equal to 1, it equals to %d", len(lastSaved))
	}
	if lastSaved[0] == nil {
		return nil, errUnknownState
	}

	return lastSaved[0], nil
}

// This returns the genesis root.
func (s *State) genesisRoot(ctx context.Context) ([32]byte, error) {
	b, err := s.beaconDB.GenesisBlock(ctx)
	if err != nil {
		return [32]byte{}, err
	}
	return stateutil.BlockRoot(b.Block)
}

// This returns the highest archived root based on input slot in the DB.
// If the archived root is not available at that exact input slot due to an event of skip block,
// this will look back and return the last available archived root (ie. the one with the highest slot below input slot).
func (s *State) archivedRoot(ctx context.Context, slot uint64) ([32]byte, error) {
	archivedIndex := slot / params.BeaconConfig().SlotsPerArchivedPoint
	for archivedIndex > 0 {
		if ctx.Err() != nil {
			return [32]byte{}, ctx.Err()
		}
		if s.beaconDB.HasArchivedPoint(ctx, archivedIndex) {
			return s.beaconDB.ArchivedPointRoot(ctx, archivedIndex), nil
		}
		archivedIndex--
	}

	if archivedIndex == 0 {
		return s.genesisRoot(ctx)
	}

	return [32]byte{}, errUnknownArchivedState
}

// This retrieves the archived state in the DB.
func (s *State) archivedState(ctx context.Context, slot uint64) (*state.BeaconState, error) {
	archivedRoot, err := s.archivedRoot(ctx, slot)
	if err != nil {
		return nil, err
	}
	return s.beaconDB.State(ctx, archivedRoot)
}

// This recomputes a state given the block root.
func (s *State) recoverStateByRoot(ctx context.Context, root [32]byte) (*state.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "stateGen.recoverStateByRoot")
	defer span.End()

	lastAncestorState, err := s.lastAncestorState(ctx, root)
	if err != nil {
		return nil, err
	}
	if lastAncestorState == nil {
		return nil, errUnknownState
	}

	targetBlk, err := s.beaconDB.Block(ctx, root)
	if err != nil {
		return nil, err
	}
	if targetBlk == nil {
		return nil, errUnknownBlock
	}
	blks, err := s.LoadBlocks(ctx, lastAncestorState.Slot()+1, targetBlk.Block.Slot, root)
	if err != nil {
		return nil, errors.Wrap(err, "could not load blocks for cold state using root")
	}

	return s.ReplayBlocks(ctx, lastAncestorState, blks, targetBlk.Block.Slot)
}

// This processes a state up to input slot.
func (s *State) processStateUpTo(ctx context.Context, state *state.BeaconState, slot uint64) (*state.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "stateGen.processStateUpTo")
	defer span.End()

	// Short circuit if the slot is already less than pre state.
	if state.Slot() >= slot {
		return state, nil
	}

	lastBlockRoot, lastBlockSlot, err := s.lastSavedBlock(ctx, slot)
	if err != nil {
		return nil, errors.Wrap(err, "could not get last saved block")
	}
	// Short circuit if no block was saved, replay using slots only.
	if lastBlockSlot == 0 {
		return s.ReplayBlocks(ctx, state, []*ethpb.SignedBeaconBlock{}, slot)
	}

	blks, err := s.LoadBlocks(ctx, state.Slot()+1, lastBlockSlot, lastBlockRoot)
	if err != nil {
		return nil, errors.Wrap(err, "could not load blocks")
	}
	state, err = s.ReplayBlocks(ctx, state, blks, slot)
	if err != nil {
		return nil, errors.Wrap(err, "could not replay blocks")
	}

	return state, nil
}
