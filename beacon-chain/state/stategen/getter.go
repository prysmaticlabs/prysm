package stategen

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"go.opencensus.io/trace"
)

// StateByRoot retrieves the state from DB using input block root.
// It retrieves state from the hot section if the state summary slot
// is below the split point cut off.
func (s *State) StateByRoot(ctx context.Context, blockRoot [32]byte) (*state.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "stateGen.StateByRoot")
	defer span.End()

	summary, err := s.beaconDB.StateSummary(ctx, blockRoot)
	if err != nil {
		return nil, errors.Wrap(err, "could not get state summary")
	}

	if summary.Slot < s.splitInfo.slot {
		return s.loadColdStateByRoot(ctx, blockRoot)
	}

	return s.loadHotStateByRoot(ctx, blockRoot)
}

// StateBySlot retrieves the state from DB using input slot.
// It retrieves state from the cold section if the input slot
// is below the split point cut off.
// Note: `StateByRoot` is preferred over this. Retrieving state
// by root `StateByRoot` is more performant than retrieving by slot.
func (s *State) StateBySlot(ctx context.Context, slot uint64) (*state.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "stateGen.StateBySlot")
	defer span.End()

	if slot < s.splitInfo.slot {
		return s.loadColdIntermediateStateBySlot(ctx, slot)
	}

	return s.loadHotStateBySlot(ctx, slot)
}
