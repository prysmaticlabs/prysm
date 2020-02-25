package state

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"go.opencensus.io/trace"
)

// ExecuteStateTransitionStateGen applies state transition on input historical state and block for state gen usages.
// There's no signature verification involved given state gen only works with stored block and state in DB.
// If the objects are already in stored in DB, one can omit redundant signature checks and ssz hashing calculations.
// WARNING: This method should not be used for new block.
func ExecuteStateTransitionStateGen(
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
	state, err = ProcessSlotsStateGen(ctx, state, signed.Block.Slot)
	if err != nil {
		return nil, errors.Wrap(err, "could not process slot")
	}

	// Execute per block transition.
	// Given this is for state gen, a node only cares about the post state without proposer
	// and randao signature verifications.
	state, err = ProcessBlockForStateRoot(ctx, state, signed)
	if err != nil {
		return nil, errors.Wrap(err, "could not process block")
	}

	return state, nil
}

// ProcessSlotsStateGen to process old slots for state gen usages.
// There's no skip slot cache involved given state gen only works with already stored block and state in DB.
// WARNING: This method should not be used for new slot.
func ProcessSlotsStateGen(ctx context.Context, state *stateTrie.BeaconState, slot uint64) (*stateTrie.BeaconState, error) {
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
		state, err := ProcessSlot(ctx, state)
		if err != nil {
			return nil, errors.Wrap(err, "could not process slot")
		}
		if CanProcessEpoch(state) {
			state, err = ProcessEpochPrecompute(ctx, state)
			if err != nil {
				return nil, errors.Wrap(err, "could not process epoch with optimizations")
			}
		}
		state.SetSlot(state.Slot() + 1)
	}

	return state, nil
}
