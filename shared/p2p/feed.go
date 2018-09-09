package p2p

import (
	"github.com/ethereum/go-ethereum/event"
	"github.com/golang/protobuf/proto"
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
func (s *Server) Feed(msg proto.Message) *event.Feed {
	t := messageType(msg)

	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.feeds[t] == nil {
		s.feeds[t] = new(event.Feed)
	}

	return s.feeds[t]
}
