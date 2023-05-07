package testutil

import (
	"context"

	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
)

// MockStater is a fake implementation of lookup.Stater.
type MockStater struct {
	BeaconState     state.BeaconState
	BeaconStateRoot []byte
	StatesBySlot    map[primitives.Slot]state.BeaconState
}

// State --
func (m *MockStater) State(context.Context, []byte) (state.BeaconState, error) {
	return m.BeaconState, nil
}

// StateRoot --
func (m *MockStater) StateRoot(context.Context, []byte) ([]byte, error) {
	return m.BeaconStateRoot, nil
}

// StateBySlot --
func (m *MockStater) StateBySlot(_ context.Context, s primitives.Slot) (state.BeaconState, error) {
	return m.StatesBySlot[s], nil
}
