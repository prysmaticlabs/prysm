package p2p

import (
	"github.com/gogo/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/encoder"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// P2P represents the full p2p interface composed of all of the sub-interfaces.
type P2P interface {
	Broadcaster
	SetStreamHandler
	EncodingProvider
	PubSubProvider
	PeerManager
	HandshakeManager
}

// Broadcaster broadcasts messages to peers over the p2p pubsub protocol.
type Broadcaster interface {
	Broadcast(proto.Message)
}

// SetStreamHandler configures p2p to handle streams of a certain topic ID.
type SetStreamHandler interface {
	SetStreamHandler(topic string, handler network.StreamHandler)
}

// EncodingProvider provides p2p network encoding.
type EncodingProvider interface {
	Encoding() encoder.NetworkEncoding
}

// PubSubProvider provides the p2p pubsub protocol.
type PubSubProvider interface {
	PubSub() *pubsub.PubSub
}

// PeerManager abstracts some peer management methods from libp2p.
type PeerManager interface {
	Disconnect(peer.ID) error
}

// HandshakeManager abstracts certain methods regarding handshake records.
type HandshakeManager interface {
	AddHandshake(peer.ID, *pb.Hello)
}
