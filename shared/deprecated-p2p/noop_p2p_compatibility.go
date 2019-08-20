package p2p

import (
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/encoder"
	ethpb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// This file exists to enable interop/compatibility between this deprecated library and the new
// p2p library. See issue #3147.

// AddHandshake not implemented.
func (s *Server) AddHandshake(_ peer.ID, _ *ethpb.Hello) {
	panic("not implemented")
}

// Handshakes not implemented.
func (s *Server) Handshakes() map[peer.ID]*ethpb.Hello {
	return nil
}

// Encoding not implemented.
func (s *Server) Encoding() encoder.NetworkEncoding {
	return nil
}

// PubSub not implemented.
func (s *Server) PubSub() *pubsub.PubSub {
	return s.gsub
}

// SetStreamHandler not implemented.
func (s *Server) SetStreamHandler(_ string, _ network.StreamHandler) {
	panic("not implemented")
}
