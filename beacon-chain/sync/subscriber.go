package sync

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/gogo/protobuf/proto"

	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
)

// subHandler represents handler for a given subscription.
type subHandler func(context.Context, proto.Message) error

func notImplementedSubHandler(_ context.Context, _ proto.Message) error {
	return errors.New("not implemented")
}

// validator should verify the contents of the message, propagate the message
// as expected, and return true or false to continue the message processing
// pipeline.
type validator func(context.Context, proto.Message, p2p.Broadcaster) bool

// noopValidator is a no-op that always returns true and does not propagate any
// message.
func noopValidator(_ context.Context, _ proto.Message, _ p2p.Broadcaster) bool {
	return true
}

// subscribe to a given topic with a given validator and subscription handler.
// The base protobuf message is used to initialize new messages for decoding.
func (r *RegularSync) subscribe(topic string, v validator, h subHandler) {
	base := GossipTopicMappings[topic]
	if base == nil {
		panic(fmt.Sprintf("%s is not mapped to any message in GossipTopicMappings", topic))
	}

	topic += r.p2p.Encoding().ProtocolSuffix()
	log := log.WithField("topic", topic)

	sub, err := r.p2p.PubSub().Subscribe(topic)
	if err != nil {
		panic(err)
	}

	// Pipeline decodes the incoming subscription data, runs the validation, and handles the
	// message.
	pipeline := func(data []byte) {
		if data == nil {
			log.Warn("Received nil message on pubsub")
			return
		}

		n := proto.Clone(base)
		if err := r.p2p.Encoding().Decode(bytes.NewBuffer(data), n); err != nil {
			log.WithError(err).Warn("Failed to decode pubsub message")
			return
		}

		if !v(r.ctx, n, r.p2p) {
			log.WithField("message", n.String()).
				Debug("Message did not verify")

			// TODO: Increment metrics.
			return
		}

		if err := h(r.ctx, n); err != nil {
			// TODO: Increment metrics.
			log.WithError(err).Error("Failed to handle p2p pubsub")
			return
		}
	}

	// The main message loop for receiving incoming messages from this subscription.
	messageLoop := func() {
		for {
			msg, err := sub.Next(r.ctx)
			if err != nil {
				log.WithError(err).Error("Subscription next failed")
				// TODO: Mark unhealthy.
				return
			}

			go pipeline(msg.Data)
		}
	}

	go messageLoop()
}
