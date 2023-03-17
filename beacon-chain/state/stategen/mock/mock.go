package mock

import (
	"context"

	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
)

// MockStateManager is a fake implementation of StateManager.
type MockStateManager struct {
	StatesByRoot map[[32]byte]state.BeaconState
	StatesBySlot map[primitives.Slot]state.BeaconState
}

// NewMockService --
func NewMockService() *MockStateManager {
	return &MockStateManager{
		StatesByRoot: make(map[[32]byte]state.BeaconState),
		StatesBySlot: make(map[primitives.Slot]state.BeaconState),
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
func (_ *MockStateManager) SaveFinalizedState(_ primitives.Slot, _ [32]byte, _ state.BeaconState) {
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

// BalancesByRoot --
func (*MockStateManager) ActiveNonSlashedBalancesByRoot(_ context.Context, _ [32]byte) ([]uint64, error) {
	return []uint64{}, nil
}

// StateByRootInitialSync --
func (_ *MockStateManager) StateByRootInitialSync(_ context.Context, _ [32]byte) (state.BeaconState, error) {
	panic("implement me")
}

// StateBySlot --
func (m *MockStateManager) StateBySlot(_ context.Context, slot primitives.Slot) (state.BeaconState, error) {
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
func (m *MockStateManager) AddStateForSlot(state state.BeaconState, slot primitives.Slot) {
	m.StatesBySlot[slot] = state
}

// DeleteStateFromCaches --
func (m *MockStateManager) DeleteStateFromCaches(context.Context, [32]byte) error {
	return nil
}
