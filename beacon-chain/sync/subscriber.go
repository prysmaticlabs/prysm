package sync

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"runtime/debug"
	"strings"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	pb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
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

var (
	// TODO(6449): Replace this with pubsub.ErrSubscriptionCancelled.
	errSubscriptionCancelled = errors.New("subscription cancelled by calling sub.Cancel()")
)

// subHandler represents handler for a given subscription.
type subHandler func(context.Context, proto.Message) error

// noopValidator is a no-op that only decodes the message, but does not check its contents.
func (s *Service) noopValidator(ctx context.Context, _ peer.ID, msg *pubsub.Message) pubsub.ValidationResult {
	m, err := s.decodePubsubMessage(msg)
	if err != nil {
		log.WithError(err).Error("Failed to decode message")
		return pubsub.ValidationReject
	}
	msg.ValidatorData = m
	return pubsub.ValidationAccept
}

// Register PubSub subscribers
func (s *Service) registerSubscribers() {
	s.subscribe(
		"/eth2/%x/beacon_block",
		s.validateBeaconBlockPubSub,
		s.beaconBlockSubscriber,
	)
	s.subscribe(
		"/eth2/%x/beacon_aggregate_and_proof",
		s.validateAggregateAndProof,
		s.beaconAggregateProofSubscriber,
	)
	s.subscribe(
		"/eth2/%x/voluntary_exit",
		s.validateVoluntaryExit,
		s.voluntaryExitSubscriber,
	)
	s.subscribe(
		"/eth2/%x/proposer_slashing",
		s.validateProposerSlashing,
		s.proposerSlashingSubscriber,
	)
	s.subscribe(
		"/eth2/%x/attester_slashing",
		s.validateAttesterSlashing,
		s.attesterSlashingSubscriber,
	)
	if featureconfig.Get().DisableDynamicCommitteeSubnets {
		s.subscribeStaticWithSubnets(
			"/eth2/%x/beacon_attestation_%d",
			s.validateCommitteeIndexBeaconAttestation,   /* validator */
			s.committeeIndexBeaconAttestationSubscriber, /* message handler */
		)
	} else {
		s.subscribeDynamicWithSubnets(
			"/eth2/%x/beacon_attestation_%d",
			s.validateCommitteeIndexBeaconAttestation,   /* validator */
			s.committeeIndexBeaconAttestationSubscriber, /* message handler */
		)
	}
}

// subscribe to a given topic with a given validator and subscription handler.
// The base protobuf message is used to initialize new messages for decoding.
func (s *Service) subscribe(topic string, validator pubsub.ValidatorEx, handle subHandler) *pubsub.Subscription {
	base := p2p.GossipTopicMappings[topic]
	if base == nil {
		panic(fmt.Sprintf("%s is not mapped to any message in GossipTopicMappings", topic))
	}
	return s.subscribeWithBase(base, s.addDigestToTopic(topic), validator, handle)
}

func (s *Service) subscribeWithBase(base proto.Message, topic string, validator pubsub.ValidatorEx, handle subHandler) *pubsub.Subscription {
	topic += s.p2p.Encoding().ProtocolSuffix()
	log := log.WithField("topic", topic)

	if err := s.p2p.PubSub().RegisterTopicValidator(wrapAndReportValidation(topic, validator)); err != nil {
		log.WithError(err).Error("Failed to register validator")
	}

	sub, err := s.p2p.PubSub().Subscribe(topic)
	if err != nil {
		// Any error subscribing to a PubSub topic would be the result of a misconfiguration of
		// libp2p PubSub library. This should not happen at normal runtime, unless the config
		// changes to a fatal configuration.
		panic(err)
	}

	// Pipeline decodes the incoming subscription data, runs the validation, and handles the
	// message.
	pipeline := func(msg *pubsub.Message) {
		ctx, cancel := context.WithTimeout(context.Background(), pubsubMessageTimeout)
		defer cancel()
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
			msg, err := sub.Next(s.ctx)
			if err != nil {
				// This should only happen when the context is cancelled or subscription is cancelled.
				if err != errSubscriptionCancelled { // Only log a warning on unexpected errors.
					log.WithError(err).Warn("Subscription next failed")
				}
				return
			}

			if msg.ReceivedFrom == s.p2p.PeerID() {
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
func wrapAndReportValidation(topic string, v pubsub.ValidatorEx) (string, pubsub.ValidatorEx) {
	return topic, func(ctx context.Context, pid peer.ID, msg *pubsub.Message) pubsub.ValidationResult {
		defer messagehandler.HandlePanic(ctx, msg)
		ctx, cancel := context.WithTimeout(ctx, pubsubMessageTimeout)
		defer cancel()
		messageReceivedCounter.WithLabelValues(topic).Inc()
		b := v(ctx, pid, msg)
		if b == pubsub.ValidationReject {
			messageFailedValidationCounter.WithLabelValues(topic).Inc()
		}
		return b
	}
}

// subscribe to a static subnet  with the given topic and index.A given validator and subscription handler is
// used to handle messages from the subnet. The base protobuf message is used to initialize new messages for decoding.
func (s *Service) subscribeStaticWithSubnets(topic string, validator pubsub.ValidatorEx, handle subHandler) {
	base := p2p.GossipTopicMappings[topic]
	if base == nil {
		panic(fmt.Sprintf("%s is not mapped to any message in GossipTopicMappings", topic))
	}
	for i := uint64(0); i < params.BeaconNetworkConfig().AttestationSubnetCount; i++ {
		s.subscribeWithBase(base, s.addDigestAndIndexToTopic(topic, i), validator, handle)
	}
	genesis := s.chain.GenesisTime()
	ticker := slotutil.GetSlotTicker(genesis, params.BeaconConfig().SecondsPerSlot)

	go func() {
		for {
			select {
			case <-s.ctx.Done():
				ticker.Done()
				return
			case <-ticker.C():
				if s.chainStarted && s.initialSync.Syncing() {
					continue
				}
				// Check every slot that there are enough peers
				for i := uint64(0); i < params.BeaconNetworkConfig().AttestationSubnetCount; i++ {
					if !s.validPeersExist(topic, i) {
						log.Debugf("No peers found subscribed to attestation gossip subnet with "+
							"committee index %d. Searching network for peers subscribed to the subnet.", i)
						go func(idx uint64) {
							_, err := s.p2p.FindPeersWithSubnet(idx)
							if err != nil {
								log.Errorf("Could not search for peers: %v", err)
								return
							}
						}(i)
						return
					}
				}
			}
		}
	}()
}

// subscribe to a dynamically changing list of subnets. This method expects a fmt compatible
// string for the topic name and the list of subnets for subscribed topics that should be
// maintained.
func (s *Service) subscribeDynamicWithSubnets(
	topicFormat string,
	validate pubsub.ValidatorEx,
	handle subHandler,
) {
	base := p2p.GossipTopicMappings[topicFormat]
	if base == nil {
		log.Fatalf("%s is not mapped to any message in GossipTopicMappings", topicFormat)
	}
	digest, err := s.forkDigest()
	if err != nil {
		log.WithError(err).Fatal("Could not compute fork digest")
	}
	subscriptions := make(map[uint64]*pubsub.Subscription, params.BeaconConfig().MaxCommitteesPerSlot)
	genesis := s.chain.GenesisTime()
	ticker := slotutil.GetSlotTicker(genesis, params.BeaconConfig().SecondsPerSlot)

	go func() {
		for {
			select {
			case <-s.ctx.Done():
				ticker.Done()
				return
			case currentSlot := <-ticker.C():
				if s.chainStarted && s.initialSync.Syncing() {
					continue
				}

				// Persistent subscriptions from validators
				persistentSubs := s.persistentSubnetIndices()
				// Update desired topic indices for aggregator
				wantedSubs := s.aggregatorSubnetIndices(currentSlot)

				// Combine subscriptions to get all requested subscriptions
				wantedSubs = sliceutil.SetUint64(append(persistentSubs, wantedSubs...))
				// Resize as appropriate.
				s.reValidateSubscriptions(subscriptions, wantedSubs, topicFormat, digest)

				// subscribe desired aggregator subnets.
				for _, idx := range wantedSubs {
					s.subscribeAggregatorSubnet(subscriptions, idx, base, digest, validate, handle)
				}
				// find desired subs for attesters
				attesterSubs := s.attesterSubnetIndices(currentSlot)
				for _, idx := range attesterSubs {
					s.lookupAttesterSubnets(digest, idx)
				}
			}
		}
	}()
}

// revalidate that our currently connected subnets are valid.
func (s *Service) reValidateSubscriptions(subscriptions map[uint64]*pubsub.Subscription,
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
			fullTopic := fmt.Sprintf(topicFormat, digest, k) + s.p2p.Encoding().ProtocolSuffix()
			if err := s.p2p.PubSub().UnregisterTopicValidator(fullTopic); err != nil {
				log.WithError(err).Error("Failed to unregister topic validator")
			}
			delete(subscriptions, k)
		}
	}
}

// subscribe missing subnets for our aggregators.
func (s *Service) subscribeAggregatorSubnet(subscriptions map[uint64]*pubsub.Subscription, idx uint64,
	base proto.Message, digest [4]byte, validate pubsub.ValidatorEx, handle subHandler) {
	// do not subscribe if we have no peers in the same
	// subnet
	topic := p2p.GossipTypeMapping[reflect.TypeOf(&pb.Attestation{})]
	subnetTopic := fmt.Sprintf(topic, digest, idx)
	// check if subscription exists and if not subscribe the relevant subnet.
	if _, exists := subscriptions[idx]; !exists {
		subscriptions[idx] = s.subscribeWithBase(base, subnetTopic, validate, handle)
	}
	if !s.validPeersExist(subnetTopic, idx) {
		log.Debugf("No peers found subscribed to attestation gossip subnet with "+
			"committee index %d. Searching network for peers subscribed to the subnet.", idx)
		go func(idx uint64) {
			_, err := s.p2p.FindPeersWithSubnet(idx)
			if err != nil {
				log.Errorf("Could not search for peers: %v", err)
				return
			}
		}(idx)
		return
	}
}

// lookup peers for attester specific subnets.
func (s *Service) lookupAttesterSubnets(digest [4]byte, idx uint64) {
	topic := p2p.GossipTypeMapping[reflect.TypeOf(&pb.Attestation{})]
	subnetTopic := fmt.Sprintf(topic, digest, idx)
	if !s.validPeersExist(subnetTopic, idx) {
		log.Debugf("No peers found subscribed to attestation gossip subnet with "+
			"committee index %d. Searching network for peers subscribed to the subnet.", idx)
		go func(idx uint64) {
			// perform a search for peers with the desired committee index.
			_, err := s.p2p.FindPeersWithSubnet(idx)
			if err != nil {
				log.Errorf("Could not search for peers: %v", err)
				return
			}
		}(idx)
	}
}

// find if we have peers who are subscribed to the same subnet
func (s *Service) validPeersExist(subnetTopic string, idx uint64) bool {
	numOfPeers := s.p2p.PubSub().ListPeers(subnetTopic + s.p2p.Encoding().ProtocolSuffix())
	return len(s.p2p.Peers().SubscribedToSubnet(idx)) > 0 || len(numOfPeers) > 0
}

// Add fork digest to topic.
func (s *Service) addDigestToTopic(topic string) string {
	if !strings.Contains(topic, "%x") {
		log.Fatal("Topic does not have appropriate formatter for digest")
	}
	digest, err := s.forkDigest()
	if err != nil {
		log.WithError(err).Fatal("Could not compute fork digest")
	}
	return fmt.Sprintf(topic, digest)
}

// Add the digest and index to subnet topic.
func (s *Service) addDigestAndIndexToTopic(topic string, idx uint64) string {
	if !strings.Contains(topic, "%x") {
		log.Fatal("Topic does not have appropriate formatter for digest")
	}
	digest, err := s.forkDigest()
	if err != nil {
		log.WithError(err).Fatal("Could not compute fork digest")
	}
	return fmt.Sprintf(topic, digest, idx)
}

func (s *Service) forkDigest() ([4]byte, error) {
	genRoot := s.chain.GenesisValidatorRoot()
	return p2putils.CreateForkDigest(s.chain.GenesisTime(), genRoot[:])
}
