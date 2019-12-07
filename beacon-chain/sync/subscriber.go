package sync

import (
	"context"
	"fmt"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/libp2p/go-libp2p-pubsub"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/statefeed"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
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
	// TODO(4154): Uncomment.
	//r.subscribeDynamic(
	//	"/eth2/committee_index/%d_beacon_attestation",
	//	r.currentCommitteeIndex,                     /* determineSubsLen */
	//	noopValidator,                               /* validator */
	//	r.committeeIndexBeaconAttestationSubscriber, /* message handler */
	//)
}

// subscribe to a given topic with a given validator and subscription handler.
// The base protobuf message is used to initialize new messages for decoding.
func (r *RegularSync) subscribe(topic string, validate validator, handle subHandler) {
	base := p2p.GossipTopicMappings[topic]
	if base == nil {
		panic(fmt.Sprintf("%s is not mapped to any message in GossipTopicMappings", topic))
	}

	topic += r.p2p.Encoding().ProtocolSuffix()

	sub, err := r.p2p.PubSub().Subscribe(topic)
	if err != nil {
		// Any error subscribing to a PubSub topic would be the result of a misconfiguration of
		// libp2p PubSub library. This should not happen at normal runtime, unless the config
		// changes to a fatal configuration.
		panic(err)
	}

	// Pipeline decodes the incoming subscription data, runs the validation, and handles the
	// message.
	pipe := &pipeline{
		ctx:          r.ctx,
		topic:        topic,
		base:         base,
		validate:     validate,
		handle:       handle,
		encoding:     r.p2p.Encoding(),
		self:         r.p2p.PeerID(),
		sub:          sub,
		broadcaster:  r.p2p,
		chainStarted: func() bool { return r.chainStarted },
	}

	go pipe.messageLoop()
}

// subscribe to a dynamically increasing index of topics. This method expects a fmt compatible
// string for the topic name and a maxID to represent the number of subscribed topics that should be
// maintained. As the state feed emits a newly updated state, the maxID function will be called to
// determine the appropriate number of topics. This method supports only sequential number ranges
// for topics.
func (r *RegularSync) subscribeDynamic(topicFormat string, determineSubsLen func() int, validate validator, handle subHandler) {
	base := p2p.GossipTopicMappings[topicFormat]
	if base == nil {
		panic(fmt.Sprintf("%s is not mapped to any message in GossipTopicMappings", topicFormat))
	}
	topicFormat += r.p2p.Encoding().ProtocolSuffix()

	var subscriptions []*pubsub.Subscription

	stateChannel := make(chan *statefeed.Event, 1)
	stateSub := r.stateNotifier.StateFeed().Subscribe(stateChannel)
	go func() {
		for {
			select {
			case <-r.ctx.Done():
				stateSub.Unsubscribe()
				return
			case <-stateChannel:
				// Update topic count
				wantedSubs := determineSubsLen()
				// Resize as appropriate.
				if len(subscriptions) > wantedSubs { // Reduce topics
					var cancelSubs []*pubsub.Subscription
					subscriptions, cancelSubs = subscriptions[:wantedSubs-1], subscriptions[wantedSubs:]
					for _, sub := range cancelSubs {
						sub.Cancel()
					}
				} else if len(subscriptions) < wantedSubs { // Increase topics
					for i := len(subscriptions) - 1; i < wantedSubs; i++ {
						sub, err := r.p2p.PubSub().Subscribe(fmt.Sprintf(topicFormat, i))
						if err != nil {
							// Panic is appropriate here since subscriptions should only fail if
							// pubsub was misconfigured.
							panic(err)
						}
						pipe := &pipeline{
							ctx:         r.ctx,
							topic:       fmt.Sprintf(topicFormat, i),
							base:        base,
							validate:    validate,
							handle:      handle,
							encoding:    r.p2p.Encoding(),
							self:        r.p2p.PeerID(),
							sub:         sub,
							broadcaster: r.p2p,
						}
						subscriptions = append(subscriptions, sub)
						go pipe.messageLoop()
					}
				}
			}
		}
	}()
}
