package testutil

import (
	"context"

	"github.com/prysmaticlabs/prysm/beacon-chain/state-native"
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
