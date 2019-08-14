package p2p

import (
	"github.com/gogo/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/network"
	pubsub "github.com/libp2p/go-libp2p-pubsub"

	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/encoder"
)

type Composite interface {
	Broadcaster
	SetStreamHandler
	EncodingProvider
	PubSubProvider
}

type Broadcaster interface {
	Broadcast(proto.Message)
}

type SetStreamHandler interface {
	SetStreamHandler(topic string, handler network.StreamHandler)
}

type EncodingProvider interface {
	Encoding() encoder.NetworkEncoding
}

type PubSubProvider interface {
	PubSub() *pubsub.PubSub
}
