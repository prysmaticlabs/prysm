package shared

import (
	"github.com/golang/protobuf/proto"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/p2p"
)

// Service is a struct that can be registered into a ServiceRegistry for
// easy dependency management.
type Service interface {
	// Start spawns any goroutines required by the service.
	Start()
	// Stop terminates all goroutines belonging to the service,
	// blocking until they are all terminated.
	Stop() error
}

// P2P defines a struct that can subscribe to feeds, request data, and broadcast data.
type P2P interface {
	Subscribe(msg proto.Message, channel chan p2p.Message) event.Subscription
	Send(msg proto.Message, peer p2p.Peer)
	Broadcast(msg proto.Message)
}
