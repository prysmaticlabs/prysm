package mock

import (
	"context"

	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
)

// MockStateManager is a fake implementation of StateManager.
type MockStateManager struct {
	StatesByRoot map[[32]byte]state.BeaconState
	StatesBySlot map[types.Slot]state.BeaconState
}

// NewMockService --
func NewMockService() *MockStateManager {
	return &MockStateManager{
		StatesByRoot: make(map[[32]byte]state.BeaconState),
		StatesBySlot: make(map[types.Slot]state.BeaconState),
	}
}

// StateByRootIfCachedNoCopy --
func (_ *MockStateManager) StateByRootIfCachedNoCopy(_ [32]byte) state.BeaconState {
	panic("implement me")
}

// Resume --
func (_ *MockStateManager) Resume(_ context.Context, _ state.BeaconState) (state.BeaconState, error) {
	panic("implement me")
}

// SaveFinalizedState --
func (_ *MockStateManager) SaveFinalizedState(_ types.Slot, _ [32]byte, _ state.BeaconState) {
	panic("implement me")
}

// MigrateToCold --
func (_ *MockStateManager) MigrateToCold(_ context.Context, _ [32]byte) error {
	panic("implement me")
}

// HasState --
func (_ *MockStateManager) HasState(_ context.Context, _ [32]byte) (bool, error) {
	panic("implement me")
}

// StateByRoot --
func (m *MockStateManager) StateByRoot(_ context.Context, blockRoot [32]byte) (state.BeaconState, error) {
	return m.StatesByRoot[blockRoot], nil
}

// StateByRootInitialSync --
func (_ *MockStateManager) StateByRootInitialSync(_ context.Context, _ [32]byte) (state.BeaconState, error) {
	panic("implement me")
}

// StateBySlot --
func (m *MockStateManager) StateBySlot(_ context.Context, slot types.Slot) (state.BeaconState, error) {
	return m.StatesBySlot[slot], nil
}

// SaveState --
func (_ *MockStateManager) SaveState(_ context.Context, _ [32]byte, _ state.BeaconState) error {
	panic("implement me")
}

// ForceCheckpoint --
func (_ *MockStateManager) ForceCheckpoint(_ context.Context, _ []byte) error {
	panic("implement me")
}

// EnableSaveHotStateToDB --
func (_ *MockStateManager) EnableSaveHotStateToDB(_ context.Context) {
	panic("implement me")
}

// DisableSaveHotStateToDB --
func (_ *MockStateManager) DisableSaveHotStateToDB(_ context.Context) error {
	panic("implement me")
}

// AddStateForRoot --
func (m *MockStateManager) AddStateForRoot(state state.BeaconState, blockRoot [32]byte) {
	m.StatesByRoot[blockRoot] = state
}

// AddStateForSlot --
func (m *MockStateManager) AddStateForSlot(state state.BeaconState, slot types.Slot) {
	m.StatesBySlot[slot] = state
}

// DeleteStateFromCaches --
func (m *MockStateManager) DeleteStateFromCaches(context.Context, [32]byte) error {
	return nil
}
