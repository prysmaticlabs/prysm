package sync

import (
	"context"
	"fmt"
	"reflect"
	"runtime/debug"
	"strings"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	pb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/messagehandler"
	"github.com/prysmaticlabs/prysm/shared/p2putils"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/sliceutil"
	"github.com/prysmaticlabs/prysm/shared/slotutil"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	"go.opencensus.io/trace"
)

const pubsubMessageTimeout = 30 * time.Second

var maximumGossipClockDisparity = params.BeaconNetworkConfig().MaximumGossipClockDisparity

// subHandler represents handler for a given subscription.
type subHandler func(context.Context, proto.Message) error

// noopValidator is a no-op that only decodes the message, but does not check its contents.
func (r *Service) noopValidator(ctx context.Context, _ peer.ID, msg *pubsub.Message) bool {
	m, err := r.decodePubsubMessage(msg)
	if err != nil {
		log.WithError(err).Error("Failed to decode message")
		return false
	}
	msg.ValidatorData = m
	return true
}

// Register PubSub subscribers
func (r *Service) registerSubscribers() {
	r.subscribe(
		"/eth2/%x/beacon_block",
		r.validateBeaconBlockPubSub,
		r.beaconBlockSubscriber,
	)
	r.subscribe(
		"/eth2/%x/beacon_aggregate_and_proof",
		r.validateAggregateAndProof,
		r.beaconAggregateProofSubscriber,
	)
	r.subscribe(
		"/eth2/%x/voluntary_exit",
		r.validateVoluntaryExit,
		r.voluntaryExitSubscriber,
	)
	r.subscribe(
		"/eth2/%x/proposer_slashing",
		r.validateProposerSlashing,
		r.proposerSlashingSubscriber,
	)
	r.subscribe(
		"/eth2/%x/attester_slashing",
		r.validateAttesterSlashing,
		r.attesterSlashingSubscriber,
	)
	if featureconfig.Get().DisableDynamicCommitteeSubnets {
		r.subscribeDynamic(
			"/eth2/%x/committee_index%d_beacon_attestation",
			r.committeesCount,                           /* determineSubsLen */
			r.validateCommitteeIndexBeaconAttestation,   /* validator */
			r.committeeIndexBeaconAttestationSubscriber, /* message handler */
		)
	} else {
		r.subscribeDynamicWithSubnets(
			"/eth2/%x/committee_index%d_beacon_attestation",
			r.validateCommitteeIndexBeaconAttestation,   /* validator */
			r.committeeIndexBeaconAttestationSubscriber, /* message handler */
		)
	}
}

// subscribe to a given topic with a given validator and subscription handler.
// The base protobuf message is used to initialize new messages for decoding.
func (r *Service) subscribe(topic string, validator pubsub.Validator, handle subHandler) *pubsub.Subscription {
	base := p2p.GossipTopicMappings[topic]
	if base == nil {
		panic(fmt.Sprintf("%s is not mapped to any message in GossipTopicMappings", topic))
	}
	return r.subscribeWithBase(base, r.addDigestToTopic(topic), validator, handle)
}

func (r *Service) subscribeWithBase(base proto.Message, topic string, validator pubsub.Validator, handle subHandler) *pubsub.Subscription {
	topic += r.p2p.Encoding().ProtocolSuffix()
	log := log.WithField("topic", topic)

	if err := r.p2p.PubSub().RegisterTopicValidator(wrapAndReportValidation(topic, validator)); err != nil {
		log.WithError(err).Error("Failed to register validator")
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

		if msg.ValidatorData == nil {
			log.Error("Received nil message on pubsub")
			messageFailedProcessingCounter.WithLabelValues(topic).Inc()
			return
		}

		if err := handle(ctx, msg.ValidatorData.(proto.Message)); err != nil {
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
				// This should only happen when the context is cancelled or subscription is cancelled.
				log.WithError(err).Warn("Subscription next failed")
				return
			}

			if msg.ReceivedFrom == r.p2p.PeerID() {
				continue
			}

			go pipeline(msg)
		}
	}

	go messageLoop()
	return sub
}

// Wrap the pubsub validator with a metric monitoring function. This function increments the
// appropriate counter if the particular message fails to validate.
func wrapAndReportValidation(topic string, v pubsub.Validator) (string, pubsub.Validator) {
	return topic, func(ctx context.Context, pid peer.ID, msg *pubsub.Message) bool {
		defer messagehandler.HandlePanic(ctx, msg)
		ctx, _ = context.WithTimeout(ctx, pubsubMessageTimeout)
		messageReceivedCounter.WithLabelValues(topic).Inc()
		b := v(ctx, pid, msg)
		if !b {
			messageFailedValidationCounter.WithLabelValues(topic).Inc()
		}
		return b
	}
}

// subscribe to a dynamically changing list of subnets. This method expects a fmt compatible
// string for the topic name and the list of subnets for subscribed topics that should be
// maintained.
func (r *Service) subscribeDynamicWithSubnets(
	topicFormat string,
	validate pubsub.Validator,
	handle subHandler,
) {
	base := p2p.GossipTopicMappings[topicFormat]
	if base == nil {
		log.Fatalf("%s is not mapped to any message in GossipTopicMappings", topicFormat)
	}
	digest, err := r.forkDigest()
	if err != nil {
		log.WithError(err).Fatal("Could not compute fork digest")
	}
	subscriptions := make(map[uint64]*pubsub.Subscription, params.BeaconConfig().MaxCommitteesPerSlot)
	genesis := r.chain.GenesisTime()
	ticker := slotutil.GetSlotTicker(genesis, params.BeaconConfig().SecondsPerSlot)

	go func() {
		for {
			select {
			case <-r.ctx.Done():
				ticker.Done()
				return
			case currentSlot := <-ticker.C():
				if r.chainStarted && r.initialSync.Syncing() {
					continue
				}

				// Persistent subscriptions from validators
				persistentSubs := r.persistentCommitteeIndices()
				// Update desired topic indices for aggregator
				wantedSubs := r.aggregatorCommitteeIndices(currentSlot)

				// Combine subscriptions to get all requested subscriptions
				wantedSubs = sliceutil.SetUint64(append(persistentSubs, wantedSubs...))
				// Resize as appropriate.
				r.reValidateSubscriptions(subscriptions, wantedSubs, topicFormat, digest)

				// subscribe desired aggregator subnets.
				for _, idx := range wantedSubs {
					r.subscribeAggregatorSubnet(subscriptions, idx, base, digest, validate, handle)
				}
				// find desired subs for attesters
				attesterSubs := r.attesterCommitteeIndices(currentSlot)
				for _, idx := range attesterSubs {
					r.lookupAttesterSubnets(digest, idx)
				}
			}
		}
	}()
}

// subscribe to a dynamically increasing index of topics. This method expects a fmt compatible
// string for the topic name and a maxID to represent the number of subscribed topics that should be
// maintained. As the state feed emits a newly updated state, the maxID function will be called to
// determine the appropriate number of topics. This method supports only sequential number ranges
// for topics.
func (r *Service) subscribeDynamic(topicFormat string, determineSubsLen func() int, validate pubsub.Validator, handle subHandler) {
	base := p2p.GossipTopicMappings[topicFormat]
	if base == nil {
		log.Fatalf("%s is not mapped to any message in GossipTopicMappings", topicFormat)
	}
	digest, err := r.forkDigest()
	if err != nil {
		log.WithError(err).Fatal("Could not compute fork digest")
	}
	var subscriptions []*pubsub.Subscription

	stateChannel := make(chan *feed.Event, 1)
	stateSub := r.stateNotifier.StateFeed().Subscribe(stateChannel)
	go func() {
		for {
			select {
			case <-r.ctx.Done():
				stateSub.Unsubscribe()
				return
			case <-stateChannel:
				if r.chainStarted && r.initialSync.Syncing() {
					continue
				}
				// Update topic count.
				wantedSubs := determineSubsLen()
				// Resize as appropriate.
				if len(subscriptions) > wantedSubs { // Reduce topics
					var cancelSubs []*pubsub.Subscription
					subscriptions, cancelSubs = subscriptions[:wantedSubs-1], subscriptions[wantedSubs:]
					for i, sub := range cancelSubs {
						sub.Cancel()
						if err := r.p2p.PubSub().UnregisterTopicValidator(fmt.Sprintf(topicFormat, i+wantedSubs)); err != nil {
							log.WithError(err).Error("Failed to unregister topic validator")
						}
					}
				} else if len(subscriptions) < wantedSubs { // Increase topics
					for i := len(subscriptions); i < wantedSubs; i++ {
						sub := r.subscribeWithBase(base, fmt.Sprintf(topicFormat, digest, i), validate, handle)
						subscriptions = append(subscriptions, sub)
					}
				}
			}
		}
	}()
}

// revalidate that our currently connected subnets are valid.
func (r *Service) reValidateSubscriptions(subscriptions map[uint64]*pubsub.Subscription,
	wantedSubs []uint64, topicFormat string, digest [4]byte) {
	for k, v := range subscriptions {
		var wanted bool
		for _, idx := range wantedSubs {
			if k == idx {
				wanted = true
				break
			}
		}
		if !wanted && v != nil {
			v.Cancel()
			fullTopic := fmt.Sprintf(topicFormat, digest, k) + r.p2p.Encoding().ProtocolSuffix()
			if err := r.p2p.PubSub().UnregisterTopicValidator(fullTopic); err != nil {
				log.WithError(err).Error("Failed to unregister topic validator")
			}
			delete(subscriptions, k)
		}
	}
}

// subscribe missing subnets for our aggregators.
func (r *Service) subscribeAggregatorSubnet(subscriptions map[uint64]*pubsub.Subscription, idx uint64,
	base proto.Message, digest [4]byte, validate pubsub.Validator, handle subHandler) {
	// do not subscribe if we have no peers in the same
	// subnet
	topic := p2p.GossipTypeMapping[reflect.TypeOf(&pb.Attestation{})]
	subnetTopic := fmt.Sprintf(topic, digest, idx)
	// check if subscription exists and if not subscribe the relevant subnet.
	if _, exists := subscriptions[idx]; !exists {
		subscriptions[idx] = r.subscribeWithBase(base, subnetTopic, validate, handle)
	}
	if !r.validPeersExist(subnetTopic, idx) {
		log.Debugf("No peers found subscribed to attestation gossip subnet with "+
			"committee index %d. Searching network for peers subscribed to the subnet.", idx)
		go func(idx uint64) {
			_, err := r.p2p.FindPeersWithSubnet(idx)
			if err != nil {
				log.Errorf("Could not search for peers: %v", err)
				return
			}
		}(idx)
		return
	}
}

// lookup peers for attester specific subnets.
func (r *Service) lookupAttesterSubnets(digest [4]byte, idx uint64) {
	topic := p2p.GossipTypeMapping[reflect.TypeOf(&pb.Attestation{})]
	subnetTopic := fmt.Sprintf(topic, digest, idx)
	if !r.validPeersExist(subnetTopic, idx) {
		log.Debugf("No peers found subscribed to attestation gossip subnet with "+
			"committee index %d. Searching network for peers subscribed to the subnet.", idx)
		go func(idx uint64) {
			// perform a search for peers with the desired committee index.
			_, err := r.p2p.FindPeersWithSubnet(idx)
			if err != nil {
				log.Errorf("Could not search for peers: %v", err)
				return
			}
		}(idx)
	}
}

// find if we have peers who are subscribed to the same subnet
func (r *Service) validPeersExist(subnetTopic string, idx uint64) bool {
	numOfPeers := r.p2p.PubSub().ListPeers(subnetTopic + r.p2p.Encoding().ProtocolSuffix())
	return len(r.p2p.Peers().SubscribedToSubnet(idx)) > 0 || len(numOfPeers) > 0
}

// Add fork digest to topic.
func (r *Service) addDigestToTopic(topic string) string {
	if !strings.Contains(topic, "%x") {
		log.Fatal("Topic does not have appropriate formatter for digest")
	}
	digest, err := r.forkDigest()
	if err != nil {
		log.WithError(err).Fatal("Could not compute fork digest")
	}
	return fmt.Sprintf(topic, digest)
}

func (r *Service) forkDigest() ([4]byte, error) {
	genRoot := r.chain.GenesisValidatorRoot()
	return p2putils.CreateForkDigest(r.chain.GenesisTime(), genRoot[:])
}
