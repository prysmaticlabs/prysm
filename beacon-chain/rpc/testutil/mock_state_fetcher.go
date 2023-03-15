package testutil

import (
	"context"

	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
)

// MockFetcher is a fake implementation of statefetcher.Fetcher.
type MockFetcher struct {
	BeaconState     state.BeaconState
	BeaconStateRoot []byte
	StatesBySlot    map[primitives.Slot]state.BeaconState
}

// State --
func (m *MockFetcher) State(context.Context, []byte) (state.BeaconState, error) {
	return m.BeaconState, nil
}

// StateRoot --
func (m *MockFetcher) StateRoot(context.Context, []byte) ([]byte, error) {
	return m.BeaconStateRoot, nil
}

func (m *MockFetcher) StateBySlot(_ context.Context, s primitives.Slot) (state.BeaconState, error) {
	return m.StatesBySlot[s], nil
}
