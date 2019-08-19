package p2p

import (
	"context"

	"github.com/gogo/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/prysmaticlabs/prysm/shared/event"
)

// Message represents a message received from an external peer.
// DEPRECATED: Do not use. This exists for backwards compatibility but may be removed.
type Message struct {
	// Ctx message context.
	Ctx context.Context
	// Peer represents the sender of the message.
	Peer peer.ID
	// Data can be any type of message found in sharding/p2p/proto package.
	Data proto.Message
}

// DEPRECATED: Do not use. This exists for backwards compatibility but may be removed.
type DeprecatedSubscriber interface {
	Subscribe(msg proto.Message, channel chan Message) event.Subscription
}
