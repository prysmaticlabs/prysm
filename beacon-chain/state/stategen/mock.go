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

// StateByRootIfCached
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

// ReplayBlocks --
func (_ *MockStateManager) ReplayBlocks(
	_ context.Context,
	_ state.BeaconState,
	_ []block.SignedBeaconBlock,
	_ types.Slot,
) (state.BeaconState, error) {
	panic("implement me")
}

// LoadBlocks --
func (_ *MockStateManager) LoadBlocks(
	_ context.Context,
	_, _ types.Slot,
	_ [32]byte,
) ([]block.SignedBeaconBlock, error) {
	panic("implement me")
}

// HasState --
func (_ *MockStateManager) HasState(_ context.Context, _ [32]byte) (bool, error) {
	panic("implement me")
}

// HasStateInCache --
func (_ *MockStateManager) HasStateInCache(_ context.Context, _ [32]byte) (bool, error) {
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

// RecoverStateSummary --
func (_ *MockStateManager) RecoverStateSummary(
	_ context.Context,
	_ [32]byte,
) (*ethpb.StateSummary, error) {
	panic("implement me")
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
