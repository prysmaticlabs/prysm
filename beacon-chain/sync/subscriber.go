package sync

import (
	"context"
	"errors"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
)

// subHandler represents handler for a given subscription.
type subHandler func(context.Context, proto.Message) error

func notImplementedSubHandler(context.Context, proto.Message) error {
	return errors.New("not implemented")
}

// validator should verify the contents of the message, propagate the message
// as expected, and return true or false to continue the message processing
// pipeline.
type validator func(context.Context, proto.Message, p2p.Broadcaster) bool

// noopValidator is a no-op that always returns true and does not propagate any
// message.
func noopValidator(context.Context, proto.Message, p2p.Broadcaster) bool {
	return true
}

// subscribe to a given topic with a given validator and subscription handler.
// The base protobuf message is used to initialize new messages for decoding.
func (r *RegularSync) subscribe(topic string, base proto.Message, v validator, h subHandler) {
	sub, err := r.p2p.PubSub().Subscribe(topic + "/ssz") // TODO: Determine encoding suffix.
	if err != nil {
		panic(err)
	}

	go func() {
		for {
			msg, err := sub.Next(r.ctx)
			if err != nil {
				log.WithError(err).WithField("topic", topic).Error("Subscription next failed")
				// TODO: Mark unhealthy.
				return
			}

			go func() {
				// TODO: This doesn't need such a short timeout.
				ctx, cancel := context.WithTimeout(r.ctx, ttfbTimeout)
				defer cancel()

				b := msg.Data
				if b == nil {
					log.WithField("topic", topic).Warn("Received nil message on pubsub")
					return
				}

				n := proto.Clone(base)
				if err := r.p2p.Encoding().DecodeTo(b, n); err != nil {
					log.WithField("topic", topic).Warn("Failed to decode pubsub message")
					return
				}

				if !v(ctx, n, r.p2p) {
					log.WithField("topic", topic).
						WithField("message", n.String()).
						Debug("Message did not verify")

					// TODO: Increment metrics.
					return
				}

				if err := h(ctx, n); err != nil {
					// TODO: Increment metrics.

					return
				}
			}()
		}
	}()
}
