package p2p_test

import (
	"context"
	"fmt"

	"github.com/prysmaticlabs/prysm/shared/p2p"
)

// A basic adapter will complete its logic then call next. Some adapters
// may choose not to call next. For example, in the case of a rate
// limiter or blacklisting condition.
func reqLogger(ctx context.Context, msg p2p.Message, next p2p.Handler) {
	fmt.Println("Received message from %s", msg.Peer)
	next(ctx, msg)
}

// Functions can return an adapter in order to capture configuration.
func adapterWithParams(i int) p2p.Adapter {
	return func(ctx context.Context, msg p2p.Message, next p2p.Handler) {
		fmt.Println("Magic number is %d", i)
		i++
		next(ctx, msg)
	}
}

func ExampleServer_RegisterTopic() {
	adapters := []p2p.Adapter{reqLogger, adapterWithParams(5)}

	s, _ := p2p.NewServer()

	// TODO: Figure out the topic. Is it a protobuf topic, string, or int?
	var topic string
	var message interface{}

	s.RegisterTopic(topic, message, adapters)

	ch := make(chan p2p.Message)
	sub := s.Subscribe(message, ch)
	defer sub.Unsubscribe()
	// TODO: Show more of how the chan is used.
}
