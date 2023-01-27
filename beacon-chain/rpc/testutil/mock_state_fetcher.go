package testutil

import (
	"context"

	"github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/state/types"
)

// MockFetcher is a fake implementation of statefetcher.Fetcher.
type MockFetcher struct {
	BeaconState     types.BeaconState
	BeaconStateRoot []byte
	StatesBySlot    map[primitives.Slot]types.BeaconState
}

// State --
func (m *MockFetcher) State(context.Context, []byte) (types.BeaconState, error) {
	return m.BeaconState, nil
}

// StateRoot --
func (m *MockFetcher) StateRoot(context.Context, []byte) ([]byte, error) {
	return m.BeaconStateRoot, nil
}

func (m *MockFetcher) StateBySlot(_ context.Context, s primitives.Slot) (types.BeaconState, error) {
	return m.StatesBySlot[s], nil
}
