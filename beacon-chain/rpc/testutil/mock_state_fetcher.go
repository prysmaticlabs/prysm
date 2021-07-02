package testutil

import (
	"context"

	"github.com/prysmaticlabs/prysm/shared/interfaces"
)

// MockFetcher is a fake implementation of statefetcher.Fetcher.
type MockFetcher struct {
	BeaconState     interfaces.BeaconState
	BeaconStateRoot []byte
}

// State --
func (m *MockFetcher) State(context.Context, []byte) (interfaces.BeaconState, error) {
	return m.BeaconState, nil
}

// StateRoot --
func (m *MockFetcher) StateRoot(context.Context, []byte) ([]byte, error) {
	return m.BeaconStateRoot, nil
}
