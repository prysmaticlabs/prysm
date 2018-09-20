package p2p_test

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/prysmaticlabs/prysm/shared/p2p"
)

// A basic adapter will complete its logic then call next. Some adapters
// may choose not to call next. For example, in the case of a rate
// limiter or blacklisting condition.
func reqLogger(next p2p.Handler) p2p.Handler {
	return func(ctx context.Context, msg p2p.Message) {
		fmt.Printf("Received message from %v\n", msg.Peer)
		next(ctx, msg)
	}
}

// Functions can return an adapter in order to capture configuration.
func adapterWithParams(i int) p2p.Adapter {
	return func(next p2p.Handler) p2p.Handler {
		return func(ctx context.Context, msg p2p.Message) {
			fmt.Printf("Magic number is %d\n", i)
			i++
			next(ctx, msg)
		}
	}
}

func ExampleServer_RegisterTopic() {
	adapters := []p2p.Adapter{reqLogger, adapterWithParams(5)}

	s, _ := p2p.NewServer()

	var topic string
	var message proto.Message

	s.RegisterTopic(topic, message, adapters...)

	ch := make(chan p2p.Message)
	sub := s.Subscribe(message, ch)
	defer sub.Unsubscribe()
}
