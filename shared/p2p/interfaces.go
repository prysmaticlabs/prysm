package p2p

import (
	"github.com/gogo/protobuf/proto"
)

type Broadcaster interface {
	Broadcast(proto.Message)
}
