package testutil

import (
	"context"

	iface "github.com/prysmaticlabs/prysm/beacon-chain/state/interface"
)

// MockStateFetcher is a fake implementation of IStateFetcher.
type MockStateFetcher struct {
	BeaconState iface.BeaconState
}

func (m *MockStateFetcher) State(context.Context, []byte) (iface.BeaconState, error) {
	return m.BeaconState, nil
}
