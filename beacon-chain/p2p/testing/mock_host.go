package testing

import (
	"context"

	"github.com/libp2p/go-libp2p-core/connmgr"
	"github.com/libp2p/go-libp2p-core/event"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	"github.com/libp2p/go-libp2p-core/protocol"
	ma "github.com/multiformats/go-multiaddr"
)

// MockHost is a fake implementation of libp2p2's Host interface.
type MockHost struct {
	Addresses []ma.Multiaddr
}

// ID --
func (m *MockHost) ID() peer.ID {
	return ""
}

// Peerstore --
func (m *MockHost) Peerstore() peerstore.Peerstore {
	return nil
}

// Addrs --
func (m *MockHost) Addrs() []ma.Multiaddr {
	return m.Addresses
}

// Network --
func (m *MockHost) Network() network.Network {
	return nil
}

// Mux --
func (m *MockHost) Mux() protocol.Switch {
	return nil
}

// Connect --
func (m *MockHost) Connect(ctx context.Context, pi peer.AddrInfo) error {
	return nil
}

// SetStreamHandler --
func (m *MockHost) SetStreamHandler(pid protocol.ID, handler network.StreamHandler) {}

// SetStreamHandlerMatch --
func (m *MockHost) SetStreamHandlerMatch(protocol.ID, func(string) bool, network.StreamHandler) {}

// RemoveStreamHandler --
func (m *MockHost) RemoveStreamHandler(pid protocol.ID) {}

// NewStream --
func (m *MockHost) NewStream(ctx context.Context, p peer.ID, pids ...protocol.ID) (network.Stream, error) {
	return nil, nil
}

// Close --
func (m *MockHost) Close() error {
	return nil
}

// ConnManager --
func (m *MockHost) ConnManager() connmgr.ConnManager {
	return nil
}

// EventBus --
func (m *MockHost) EventBus() event.Bus {
	return nil
}
