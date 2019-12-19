package sync

import (
	"context"
	"fmt"
	"runtime/debug"
	"strings"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	"go.opencensus.io/trace"
)

const pubsubMessageTimeout = 10 * time.Second

// subHandler represents handler for a given subscription.
type subHandler func(context.Context, proto.Message) error

// noopValidator is a no-op that always returns true.
func (r *Service) noopValidator(ctx context.Context, _ peer.ID, msg *pubsub.Message) bool {
	if msg == nil || len(msg.TopicIDs) == 0 {
		return false
	}
	topic := msg.TopicIDs[0]
	topic = strings.TrimSuffix(topic, r.p2p.Encoding().ProtocolSuffix())
	base := p2p.GossipTopicMappings[topic]
	m := proto.Clone(base)
	if err := r.p2p.Encoding().Decode(msg.Data, m); err != nil {
		panic(err)
	}
	msg.VaidatorData = m
	return true
}

// Register PubSub subscribers
func (r *Service) registerSubscribers() {
	go func() {
		// Wait until chain start.
		stateChannel := make(chan *feed.Event, 1)
		stateSub := r.stateNotifier.StateFeed().Subscribe(stateChannel)
		defer stateSub.Unsubscribe()
		for r.chainStarted == false {
			select {
			case event := <-stateChannel:
				if event.Type == statefeed.Initialized {
					data := event.Data.(*statefeed.InitializedData)
					log.WithField("starttime", data.StartTime).Debug("Received state initialized event")
					if data.StartTime.After(roughtime.Now()) {
						stateSub.Unsubscribe()
						time.Sleep(roughtime.Until(data.StartTime))
					}
					r.chainStarted = true
				}
			case <-r.ctx.Done():
				log.Debug("Context closed, exiting goroutine")
				return
			case err := <-stateSub.Err():
				log.WithError(err).Error("Subscription to state notifier failed")
				return
			}
		}
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
func (r *Service) subscribe(topic string, validator pubsub.Validator, handle subHandler) {
	base := p2p.GossipTopicMappings[topic]
	if base == nil {
		panic(fmt.Sprintf("%s is not mapped to any message in GossipTopicMappings", topic))
	}

	topic += r.p2p.Encoding().ProtocolSuffix()
	log := log.WithField("topic", topic)

	if err := r.p2p.PubSub().RegisterTopicValidator(topic, validator); err != nil {
		// Configuring a topic validator would only return an error as a result of misconfiguration
		// and is not a runtime concern.
		panic(err)
	}

	sub, err := r.p2p.PubSub().Subscribe(topic)
	if err != nil {
		// Any error subscribing to a PubSub topic would be the result of a misconfiguration of
		// libp2p PubSub library. This should not happen at normal runtime, unless the config
		// changes to a fatal configuration.
		panic(err)
	}

	// Pipeline decodes the incoming subscription data, runs the validation, and handles the
	// message.
	pipeline := func(msg *pubsub.Message) {
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

		if msg.VaidatorData == nil {
			log.Warn("Received nil message on pubsub")
			// TODO: Increment counter!
			return
		}

		if err := handle(ctx, msg.VaidatorData.(proto.Message)); err != nil {
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

			messageReceivedCounter.WithLabelValues(topic + r.p2p.Encoding().ProtocolSuffix()).Inc()

			go pipeline(msg)
		}
	}

	go messageLoop()
}
