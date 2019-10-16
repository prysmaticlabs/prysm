package sync

import (
	"context"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	"go.opencensus.io/trace"
)

const oneYear = 365 * 24 * time.Hour
const pubsubMessageTimeout = 10 * time.Second

// prefix to add to keys, so that we can represent invalid objects
const invalid = "invalidObject"

// subHandler represents handler for a given subscription.
type subHandler func(context.Context, proto.Message) error

// validator should verify the contents of the message, propagate the message
// as expected, and return true or false to continue the message processing
// pipeline. FromSelf indicates whether or not this is a message received from our
// node in pubsub.
type validator func(ctx context.Context, msg proto.Message, broadcaster p2p.Broadcaster, fromSelf bool) (bool, error)

// noopValidator is a no-op that always returns true and does not propagate any
// message.
func noopValidator(_ context.Context, _ proto.Message, _ p2p.Broadcaster, _ bool) (bool, error) {
	return true, nil
}

// Register PubSub subscribers
func (r *RegularSync) registerSubscribers() {
	go func() {
		ch := make(chan time.Time)
		sub := r.chain.StateInitializedFeed().Subscribe(ch)
		defer sub.Unsubscribe()

		// Wait until chain start.
		genesis := <-ch
		if genesis.After(roughtime.Now()) {
			time.Sleep(roughtime.Until(genesis))
		}
		r.chainStarted = true
	}()
	r.subscribe(
		"/eth2/beacon_block",
		r.validateBeaconBlockPubSub,
		r.beaconBlockSubscriber,
	)
	r.subscribe(
		"/eth2/beacon_attestation",
		r.validateBeaconAttestation,
		r.beaconAttestationSubscriber,
	)
	r.subscribe(
		"/eth2/voluntary_exit",
		r.validateVoluntaryExit,
		r.voluntaryExitSubscriber,
	)
	r.subscribe(
		"/eth2/proposer_slashing",
		r.validateProposerSlashing,
		r.proposerSlashingSubscriber,
	)
	r.subscribe(
		"/eth2/attester_slashing",
		r.validateAttesterSlashing,
		r.attesterSlashingSubscriber,
	)
}

// subscribe to a given topic with a given validator and subscription handler.
// The base protobuf message is used to initialize new messages for decoding.
func (r *RegularSync) subscribe(topic string, validate validator, handle subHandler) {
	base := p2p.GossipTopicMappings[topic]
	if base == nil {
		panic(fmt.Sprintf("%s is not mapped to any message in GossipTopicMappings", topic))
	}

	topic += r.p2p.Encoding().ProtocolSuffix()
	log := log.WithField("topic", topic)

	sub, err := r.p2p.PubSub().Subscribe(topic)
	if err != nil {
		// Any error subscribing to a PubSub topic would be the result of a misconfiguration of
		// libp2p PubSub library. This should not happen at normal runtime, unless the config
		// changes to a fatal configuration.
		panic(err)
	}

	// Pipeline decodes the incoming subscription data, runs the validation, and handles the
	// message.
	pipeline := func(data []byte, fromSelf bool) {
		ctx, _ := context.WithTimeout(context.Background(), pubsubMessageTimeout)
		ctx, span := trace.StartSpan(ctx, "sync.pubsub")
		defer span.End()

		defer func() {
			if r := recover(); r != nil {
				traceutil.AnnotateError(span, fmt.Errorf("panic occurred: %v", r))
				log.WithField("error", r).Error("Panic occurred")
				debug.PrintStack()
			}
		}()

		span.AddAttributes(trace.StringAttribute("topic", topic))
		span.AddAttributes(trace.BoolAttribute("fromSelf", fromSelf))

		if data == nil {
			log.Warn("Received nil message on pubsub")
			return
		}

		if span.IsRecordingEvents() {
			id := hashutil.FastSum64(data)
			messageLen := int64(len(data))
			span.AddMessageReceiveEvent(int64(id), messageLen /*uncompressed*/, messageLen /*compressed*/)
		}

		msg := proto.Clone(base)
		if err := r.p2p.Encoding().Decode(data, msg); err != nil {
			traceutil.AnnotateError(span, err)
			log.WithError(err).Warn("Failed to decode pubsub message")
			return
		}

		valid, err := validate(ctx, msg, r.p2p, fromSelf)
		if err != nil {
			if !fromSelf {
				log.WithError(err).Error("Message failed to verify")
				messageFailedValidationCounter.WithLabelValues(topic).Inc()
			}
			return
		}
		if !valid {
			return
		}

		if err := handle(ctx, msg); err != nil {
			traceutil.AnnotateError(span, err)
			log.WithError(err).Error("Failed to handle p2p pubsub")
			messageFailedProcessingCounter.WithLabelValues(topic).Inc()
			return
		}
	}

	// The main message loop for receiving incoming messages from this subscription.
	messageLoop := func() {
		for {
			msg, err := sub.Next(r.ctx)
			if err != nil {
				log.WithError(err).Error("Subscription next failed")
				// TODO(3147): Mark status unhealthy.
				return
			}
			if !r.chainStarted {
				messageReceivedBeforeChainStartCounter.WithLabelValues(topic + r.p2p.Encoding().ProtocolSuffix()).Inc()
				continue
			}
			// Special validation occurs on messages received from ourselves.
			fromSelf := msg.GetFrom() == r.p2p.PeerID()

			messageReceivedCounter.WithLabelValues(topic + r.p2p.Encoding().ProtocolSuffix()).Inc()

			go pipeline(msg.Data, fromSelf)
		}
	}

	go messageLoop()
}
