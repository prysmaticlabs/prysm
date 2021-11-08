package stategen

import (
	"context"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
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

// Resume --
func (m *MockStateManager) Resume(_ context.Context, _ state.BeaconState) (state.BeaconState, error) {
	panic("implement me")
}

// SaveFinalizedState --
func (m *MockStateManager) SaveFinalizedState(_ types.Slot, _ [32]byte, _ state.BeaconState) {
	panic("implement me")
}

// MigrateToCold --
func (m *MockStateManager) MigrateToCold(_ context.Context, _ [32]byte) error {
	panic("implement me")
}

// ReplayBlocks --
func (m *MockStateManager) ReplayBlocks(
	_ context.Context,
	_ state.BeaconState,
	_ []block.SignedBeaconBlock,
	_ types.Slot,
) (state.BeaconState, error) {
	panic("implement me")
}

// LoadBlocks --
func (m *MockStateManager) LoadBlocks(
	_ context.Context,
	_, _ types.Slot,
	_ [32]byte,
) ([]block.SignedBeaconBlock, error) {
	panic("implement me")
}

// HasState --
func (m *MockStateManager) HasState(_ context.Context, _ [32]byte) (bool, error) {
	panic("implement me")
}

// HasStateInCache --
func (m *MockStateManager) HasStateInCache(_ context.Context, _ [32]byte) (bool, error) {
	panic("implement me")
}

// StateByRoot --
func (m *MockStateManager) StateByRoot(_ context.Context, blockRoot [32]byte) (state.BeaconState, error) {
	return m.StatesByRoot[blockRoot], nil
}

// StateByRootInitialSync --
func (m *MockStateManager) StateByRootInitialSync(_ context.Context, _ [32]byte) (state.BeaconState, error) {
	panic("implement me")
}

// StateBySlot --
func (m *MockStateManager) StateBySlot(_ context.Context, slot types.Slot) (state.BeaconState, error) {
	return m.StatesBySlot[slot], nil
}

// RecoverStateSummary --
func (m *MockStateManager) RecoverStateSummary(
	_ context.Context,
	_ [32]byte,
) (*ethpb.StateSummary, error) {
	panic("implement me")
}

// SaveState --
func (m *MockStateManager) SaveState(_ context.Context, _ [32]byte, _ state.BeaconState) error {
	panic("implement me")
}

// ForceCheckpoint --
func (m *MockStateManager) ForceCheckpoint(_ context.Context, _ []byte) error {
	panic("implement me")
}

// EnableSaveHotStateToDB --
func (m *MockStateManager) EnableSaveHotStateToDB(_ context.Context) {
	panic("implement me")
}

// DisableSaveHotStateToDB --
func (m *MockStateManager) DisableSaveHotStateToDB(_ context.Context) error {
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
