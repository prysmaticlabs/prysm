package p2p

import (
	"github.com/gogo/protobuf/proto"
)

// Broadcaster represents a subset of the p2p.Server. This interface is useful
// for testing or when the calling code only needs access to the broadcast
// method.
type Broadcaster interface {
	Broadcast(proto.Message)
}
