package p2p

import (
	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/shared/event"
)

// Feed is a one to many subscription feed of the argument type.
//
// Messages received via p2p protocol are sent to subscribers by these event
// feeds. Message consumers should not use event feeds to reply to or broadcast
// messages. The p2p server will not relay them to peers. Rather, use the
// Send() or Broadcast() method on p2p.Server.
//
// Event feeds from p2p will always be of type p2p.Message. The message
// contains information about the sender, aka the peer, and the message payload
// itself.
//
//   feed, err := ps.Feed(&pb.MyMessage{})
//   ch := make(chan p2p.Message, 100) // Choose a reasonable buffer size!
//   sub := feed.Subscribe(ch)
//
//   // Wait until my message comes from a peer.
//   msg := <- ch
//   fmt.Printf("Message received: %v", msg.Data)
func (s *Server) Feed(msg proto.Message) Feed {
	t := messageType(msg)

	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.feeds[t] == nil {
		s.feeds[t] = new(event.Feed)
	}

	return s.feeds[t]
}

// Feed implements one-to-many subscriptions where the carrier of events is a channel.
// Values sent to a Feed are delivered to all subscribed channels simultaneously.
//
// Feeds can only be used with a single type. The type is determined by the first Send or
// Subscribe operation. Subsequent calls to these methods panic if the type does not
// match.
//
// Implemented by https://github.com/ethereum/go-ethereum/blob/HEAD/event/feed.go
type Feed interface {
	Subscribe(channel interface{}) event.Subscription
	Send(value interface{}) (nsent int)
}
