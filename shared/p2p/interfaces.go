package p2p

import (
	"context"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/shared/event"
)

// Broadcaster represents a subset of the p2p.Server. This interface is useful
// for testing or when the calling code only needs access to the broadcast
// method.
type Broadcaster interface {
	Broadcast(context.Context, proto.Message)
}

// Subscriber represents a subset of the p2p.Server. This interface is useful
// for testing or when the calling code only needs access to the subscribe
// method.
type Subscriber interface {
	Subscribe(msg proto.Message, channel chan Message) event.Subscription
}
