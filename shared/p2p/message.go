package p2p

import (
	"github.com/golang/protobuf/proto"
)

// Message represents a message received from an external peer.
type Message struct {
	// Peer represents the sender of the message.
	Peer Peer
	// Data can be any type of message found in sharding/p2p/proto package.
	Data proto.Message
}
