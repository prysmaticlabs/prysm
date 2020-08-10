package testing

import (
	"context"

	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
)

// MockPeerManager is mock of the PeerManager interface.
type MockPeerManager struct {
	Enr   *enr.Record
	PID   peer.ID
	BHost host.Host
}

// Disconnect .
func (m *MockPeerManager) Disconnect(peer.ID) error {
	return nil
}

// PeerID .
func (m *MockPeerManager) PeerID() peer.ID {
	return m.PID
}

// Host .
func (m *MockPeerManager) Host() host.Host {
	return m.BHost
}

// ENR .
func (m MockPeerManager) ENR() *enr.Record {
	return m.Enr
}

// RefreshENR .
func (m MockPeerManager) RefreshENR() {}

// FindPeersWithSubnet .
func (m MockPeerManager) FindPeersWithSubnet(ctx context.Context, index uint64) (bool, error) {
	return true, nil
}

// AddPingMethod .
func (m MockPeerManager) AddPingMethod(reqFunc func(ctx context.Context, id peer.ID) error) {}
