package stategen

import (
	"context"

	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	ethereum_beacon_p2p_v1 "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// MockService is a fake implementation of Service.
type MockService struct {
	StatesByRoot map[[32]byte]*state.BeaconState
	StatesBySlot map[uint64]*state.BeaconState
}

// NewMockService --
func NewMockService() *MockService {
	return &MockService{
		StatesByRoot: make(map[[32]byte]*state.BeaconState),
		StatesBySlot: make(map[uint64]*state.BeaconState),
	}
}

// Resume --
func (m *MockService) Resume(ctx context.Context) (*state.BeaconState, error) {
	panic("implement me")
}

// SaveFinalizedState --
func (m *MockService) SaveFinalizedState(fSlot uint64, fRoot [32]byte, fState *state.BeaconState) {
	panic("implement me")
}

// MigrateToCold --
func (m *MockService) MigrateToCold(ctx context.Context, fRoot [32]byte) error {
	panic("implement me")
}

// ReplayBlocks --
func (m *MockService) ReplayBlocks(
	ctx context.Context,
	state *state.BeaconState,
	signed []*eth.SignedBeaconBlock,
	targetSlot uint64,
) (*state.BeaconState, error) {
	panic("implement me")
}

// LoadBlocks --
func (m *MockService) LoadBlocks(
	ctx context.Context,
	startSlot, endSlot uint64,
	endBlockRoot [32]byte,
) ([]*eth.SignedBeaconBlock, error) {
	panic("implement me")
}

// HasState --
func (m *MockService) HasState(ctx context.Context, blockRoot [32]byte) (bool, error) {
	panic("implement me")
}

// HasStateInCache --
func (m *MockService) HasStateInCache(ctx context.Context, blockRoot [32]byte) (bool, error) {
	panic("implement me")
}

// StateByRoot --
func (m *MockService) StateByRoot(ctx context.Context, blockRoot [32]byte) (*state.BeaconState, error) {
	panic("implement me")
}

// StateByRootInitialSync --
func (m *MockService) StateByRootInitialSync(ctx context.Context, blockRoot [32]byte) (*state.BeaconState, error) {
	panic("implement me")
}

// StateBySlot --
func (m *MockService) StateBySlot(ctx context.Context, slot uint64) (*state.BeaconState, error) {
	panic("implement me")
}

// RecoverStateSummary --
func (m *MockService) RecoverStateSummary(
	ctx context.Context,
	blockRoot [32]byte,
) (*ethereum_beacon_p2p_v1.StateSummary, error) {
	panic("implement me")
}

// SaveState --
func (m *MockService) SaveState(ctx context.Context, root [32]byte, st *state.BeaconState) error {
	panic("implement me")
}

// ForceCheckpoint --
func (m *MockService) ForceCheckpoint(ctx context.Context, root []byte) error {
	panic("implement me")
}

// EnableSaveHotStateToDB --
func (m *MockService) EnableSaveHotStateToDB(_ context.Context) {
	panic("implement me")
}

// DisableSaveHotStateToDB --
func (m *MockService) DisableSaveHotStateToDB(ctx context.Context) error {
	panic("implement me")
}

// AddStateForRoot --
func (m *MockService) AddStateForRoot(state *state.BeaconState, blockRoot [32]byte) {
	m.StatesByRoot[blockRoot] = state
}

// AddStateForSlot --
func (m *MockService) AddStateForSlot(state *state.BeaconState, slot uint64) {
	m.StatesBySlot[slot] = state
}
