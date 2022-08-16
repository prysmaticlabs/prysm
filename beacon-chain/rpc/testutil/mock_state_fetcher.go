package testutil

import (
	"context"

	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
)

// MockFetcher is a fake implementation of statefetcher.Fetcher.
type MockFetcher struct {
	BeaconState     state.BeaconState
	BeaconStateRoot []byte
}

// State --
func (m *MockFetcher) State(context.Context, []byte) (state.BeaconState, error) {
	return m.BeaconState, nil
}

// StateRoot --
func (m *MockFetcher) StateRoot(context.Context, []byte) ([]byte, error) {
	return m.BeaconStateRoot, nil
}

func (m *MockFetcher) StateBySlot(context.Context, types.Slot) (state.BeaconState, error) {
	return m.BeaconState, nil
}
