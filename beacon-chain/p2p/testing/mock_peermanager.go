package testing

import (
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
)

// MockPeerManager is mock of the PeerManager interface.
type MockPeerManager struct {
	Enr               *enr.Record
	PID               peer.ID
	BHost             host.Host
	DiscoveryAddr     []multiaddr.Multiaddr
	FailDiscoveryAddr bool
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

// DiscoveryAddresses .
func (m MockPeerManager) DiscoveryAddresses() ([]multiaddr.Multiaddr, error) {
	if m.FailDiscoveryAddr {
		return nil, errors.New("fail")
	}
	return m.DiscoveryAddr, nil
}

// RefreshENR .
func (m MockPeerManager) RefreshENR() {}

// FindPeersWithSubnet .
func (m MockPeerManager) FindPeersWithSubnet(_ context.Context, _ string, _, _ uint64) (bool, error) {
	return true, nil
}

// AddPingMethod .
func (m MockPeerManager) AddPingMethod(_ func(ctx context.Context, id peer.ID) error) {}
