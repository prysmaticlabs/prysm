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
func (_ *MockHost) ID() peer.ID {
	return ""
}

// Peerstore --
func (_ *MockHost) Peerstore() peerstore.Peerstore {
	return nil
}

// Addrs --
func (m *MockHost) Addrs() []ma.Multiaddr {
	return m.Addresses
}

// Network --
func (_ *MockHost) Network() network.Network {
	return nil
}

// Mux --
func (_ *MockHost) Mux() protocol.Switch {
	return nil
}

// Connect --
func (_ *MockHost) Connect(ctx context.Context, pi peer.AddrInfo) error {
	return nil
}

// SetStreamHandler --
func (_ *MockHost) SetStreamHandler(pid protocol.ID, handler network.StreamHandler) {}

// SetStreamHandlerMatch --
func (_ *MockHost) SetStreamHandlerMatch(protocol.ID, func(string) bool, network.StreamHandler) {}

// RemoveStreamHandler --
func (_ *MockHost) RemoveStreamHandler(pid protocol.ID) {}

// NewStream --
func (_ *MockHost) NewStream(ctx context.Context, p peer.ID, pids ...protocol.ID) (network.Stream, error) {
	return nil, nil
}

// Close --
func (_ *MockHost) Close() error {
	return nil
}

// ConnManager --
func (_ *MockHost) ConnManager() connmgr.ConnManager {
	return nil
}

// EventBus --
func (_ *MockHost) EventBus() event.Bus {
	return nil
}
