package stategen

import (
	"context"

	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"go.opencensus.io/trace"
)

// StateByRoot retrieves the state from DB using input root.
// It retrieves state from the cold section if the cold state
// summary exists in DB by default.
func (s *State) StateByRoot(ctx context.Context, blockRoot [32]byte) (*state.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "stateGen.StateByRoot")
	defer span.End()

	if s.beaconDB.HasColdStateSummary(ctx, blockRoot) {
		return s.loadColdStateByRoot(ctx, blockRoot)
	}

	return s.loadHotStateByRoot(ctx, blockRoot)
}

// StateBySlot retrieves the state from DB using input slot.
// It retrieves state from the cold section if the input slot
// is below the splitting point of hot and cold.
// Do not use this unless it's necessary. Retrieving state
// by root `StateByRoot` is more performant than by slot.
func (s *State) StateBySlot(ctx context.Context, slot uint64) (*state.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "stateGen.StateBySlot")
	defer span.End()

	if slot < s.splitInfo.slot {
		return s.loadColdIntermediateStateWithSlot(ctx, slot)
	}

	return s.loadHotIntermediateStateWithSlot(ctx, slot)
}
