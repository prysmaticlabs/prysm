package testutil

import (
	"context"

	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
)

// MockStater is a fake implementation of lookup.Stater.
type MockStater struct {
	BeaconState     state.BeaconState
	BeaconStateRoot []byte
	StatesBySlot    map[primitives.Slot]state.BeaconState
	StatesByRoot    map[[32]byte]state.BeaconState
}

// State --
func (m *MockStater) State(_ context.Context, id []byte) (state.BeaconState, error) {
	if m.BeaconState != nil {
		return m.BeaconState, nil
	}
	return m.StatesByRoot[bytesutil.ToBytes32(id)], nil
}

// StateRoot --
func (m *MockStater) StateRoot(context.Context, []byte) ([]byte, error) {
	return m.BeaconStateRoot, nil
}

// StateBySlot --
func (m *MockStater) StateBySlot(_ context.Context, s primitives.Slot) (state.BeaconState, error) {
	return m.StatesBySlot[s], nil
}
