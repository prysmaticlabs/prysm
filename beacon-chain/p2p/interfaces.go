package p2p

import (
	"context"

	"github.com/gogo/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/encoder"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/peers"
)

// P2P represents the full p2p interface composed of all of the sub-interfaces.
type P2P interface {
	Broadcaster
	SetStreamHandler
	EncodingProvider
	PubSubProvider
	PeerManager
	Sender
	ConnectionHandler
	PeersProvider
}

// Broadcaster broadcasts messages to peers over the p2p pubsub protocol.
type Broadcaster interface {
	Broadcast(context.Context, proto.Message) error
}

// SetStreamHandler configures p2p to handle streams of a certain topic ID.
type SetStreamHandler interface {
	SetStreamHandler(topic string, handler network.StreamHandler)
}

// ConnectionHandler configures p2p to handle connections with a peer.
type ConnectionHandler interface {
	AddConnectionHandler(f func(ctx context.Context, id peer.ID) error)
	AddDisconnectionHandler(f func(ctx context.Context, id peer.ID) error)
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
	PeerID() peer.ID
}

// Sender abstracts the sending functionality from libp2p.
type Sender interface {
	Send(context.Context, interface{}, peer.ID) (network.Stream, error)
}

// PeersProvider abstracts obtaining our current list of known peers status.
type PeersProvider interface {
	Peers() *peers.Status
}
