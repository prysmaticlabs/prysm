package state_gen

import (
	"context"

	"github.com/prysmaticlabs/prysm/beacon-chain/state"
)

// StateByRoot retrieves the state from DB using input root.
func StateByRoot(ctx context.Context, root [32]byte) (*state.BeaconState, error) {
	return nil, nil
}

// StateBySlot retrieves the state from DB using input slot.
func StateBySlot(ctx context.Context, slot uint64) (*state.BeaconState, error) {
	return nil, nil
}
