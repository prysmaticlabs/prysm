package voluntaryexits

import (
	"context"

	types "github.com/prysmaticlabs/eth2-types"
	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	iface "github.com/prysmaticlabs/prysm/beacon-chain/state/interface"
)

// PoolMock is a fake implementation of PoolManager.
type PoolMock struct {
	Exits []*eth.SignedVoluntaryExit
}

// PendingExits --
func (m *PoolMock) PendingExits(_ iface.ReadOnlyBeaconState, _ types.Slot, _ bool) []*eth.SignedVoluntaryExit {
	return m.Exits
}

// InsertVoluntaryExit --
func (m *PoolMock) InsertVoluntaryExit(_ context.Context, _ iface.ReadOnlyBeaconState, exit *eth.SignedVoluntaryExit) {
	m.Exits = append(m.Exits, exit)
}

// MarkIncluded --
func (*PoolMock) MarkIncluded(_ *eth.SignedVoluntaryExit) {
	panic("implement me")
}
