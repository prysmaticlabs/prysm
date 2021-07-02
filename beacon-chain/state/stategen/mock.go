package stategen

import (
	"context"

	types "github.com/prysmaticlabs/eth2-types"
	ethereum_beacon_p2p_v1 "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/interfaces"
)

// MockStateManager is a fake implementation of StateManager.
type MockStateManager struct {
	StatesByRoot map[[32]byte]interfaces.BeaconState
	StatesBySlot map[types.Slot]interfaces.BeaconState
}

// NewMockService --
func NewMockService() *MockStateManager {
	return &MockStateManager{
		StatesByRoot: make(map[[32]byte]interfaces.BeaconState),
		StatesBySlot: make(map[types.Slot]interfaces.BeaconState),
	}
}

// Resume --
func (m *MockStateManager) Resume(ctx context.Context) (interfaces.BeaconState, error) {
	panic("implement me")
}

// SaveFinalizedState --
func (m *MockStateManager) SaveFinalizedState(fSlot types.Slot, fRoot [32]byte, fState interfaces.BeaconState) {
	panic("implement me")
}

// MigrateToCold --
func (m *MockStateManager) MigrateToCold(ctx context.Context, fRoot [32]byte) error {
	panic("implement me")
}

// ReplayBlocks --
func (m *MockStateManager) ReplayBlocks(
	ctx context.Context,
	state interfaces.BeaconState,
	signed []interfaces.SignedBeaconBlock,
	targetSlot types.Slot,
) (interfaces.BeaconState, error) {
	panic("implement me")
}

// LoadBlocks --
func (m *MockStateManager) LoadBlocks(
	ctx context.Context,
	startSlot, endSlot types.Slot,
	endBlockRoot [32]byte,
) ([]interfaces.SignedBeaconBlock, error) {
	panic("implement me")
}

// HasState --
func (m *MockStateManager) HasState(ctx context.Context, blockRoot [32]byte) (bool, error) {
	panic("implement me")
}

// HasStateInCache --
func (m *MockStateManager) HasStateInCache(ctx context.Context, blockRoot [32]byte) (bool, error) {
	panic("implement me")
}

// StateByRoot --
func (m *MockStateManager) StateByRoot(ctx context.Context, blockRoot [32]byte) (interfaces.BeaconState, error) {
	return m.StatesByRoot[blockRoot], nil
}

// StateByRootInitialSync --
func (m *MockStateManager) StateByRootInitialSync(ctx context.Context, blockRoot [32]byte) (interfaces.BeaconState, error) {
	panic("implement me")
}

// StateBySlot --
func (m *MockStateManager) StateBySlot(ctx context.Context, slot types.Slot) (interfaces.BeaconState, error) {
	return m.StatesBySlot[slot], nil
}

// RecoverStateSummary --
func (m *MockStateManager) RecoverStateSummary(
	ctx context.Context,
	blockRoot [32]byte,
) (*ethereum_beacon_p2p_v1.StateSummary, error) {
	panic("implement me")
}

// SaveState --
func (m *MockStateManager) SaveState(ctx context.Context, root [32]byte, st interfaces.BeaconState) error {
	panic("implement me")
}

// ForceCheckpoint --
func (m *MockStateManager) ForceCheckpoint(ctx context.Context, root []byte) error {
	panic("implement me")
}

// EnableSaveHotStateToDB --
func (m *MockStateManager) EnableSaveHotStateToDB(_ context.Context) {
	panic("implement me")
}

// DisableSaveHotStateToDB --
func (m *MockStateManager) DisableSaveHotStateToDB(ctx context.Context) error {
	panic("implement me")
}

// AddStateForRoot --
func (m *MockStateManager) AddStateForRoot(state interfaces.BeaconState, blockRoot [32]byte) {
	m.StatesByRoot[blockRoot] = state
}

// AddStateForSlot --
func (m *MockStateManager) AddStateForSlot(state interfaces.BeaconState, slot types.Slot) {
	m.StatesBySlot[slot] = state
}
