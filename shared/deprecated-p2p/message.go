package p2p

import (
	"reflect"

	"github.com/gogo/protobuf/proto"
	newp2p "github.com/prysmaticlabs/prysm/beacon-chain/p2p"
)

// Message represents a message received from an external peer.
type Message = newp2p.Message

// messageType returns the underlying struct type for a given proto.message.
func messageType(msg proto.Message) reflect.Type {
	// proto.Message is a pointer and we need to dereference the pointer
	// and take the type of the original struct. Otherwise reflect.TypeOf will
	// create a new value of type **ethpb.BeaconBlockHashAnnounce for example.
	return reflect.ValueOf(msg).Elem().Type()
}
