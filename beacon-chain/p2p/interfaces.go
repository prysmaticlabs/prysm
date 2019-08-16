package p2p

import (
	"github.com/gogo/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/network"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/encoder"
)

// P2P represents the full p2p interface composed of all of the sub-interfaces.
type P2P interface {
	Broadcaster
	SetStreamHandler
	EncodingProvider
	PubSubProvider
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
