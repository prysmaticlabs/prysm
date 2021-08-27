package sync

import (
	"context"
	"fmt"
	"reflect"
	"runtime/debug"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/cmd/beacon-chain/flags"
	pb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/messagehandler"
	"github.com/prysmaticlabs/prysm/shared/p2putils"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/sliceutil"
	"github.com/prysmaticlabs/prysm/shared/slotutil"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	"go.opencensus.io/trace"
	"google.golang.org/protobuf/proto"
)

const pubsubMessageTimeout = 30 * time.Second

// subHandler represents handler for a given subscription.
type subHandler func(context.Context, proto.Message) error

// noopValidator is a no-op that only decodes the message, but does not check its contents.
func (s *Service) noopValidator(_ context.Context, _ peer.ID, msg *pubsub.Message) pubsub.ValidationResult {
	m, err := s.decodePubsubMessage(msg)
	if err != nil {
		log.WithError(err).Debug("Could not decode message")
		return pubsub.ValidationReject
	}
	msg.ValidatorData = m
	return pubsub.ValidationAccept
}

// Register PubSub subscribers
func (s *Service) registerSubscribers() {
	s.subscribe(
		p2p.BlockSubnetTopicFormat,
		s.validateBeaconBlockPubSub,
		s.beaconBlockSubscriber,
	)
	s.subscribe(
		p2p.AggregateAndProofSubnetTopicFormat,
		s.validateAggregateAndProof,
		s.beaconAggregateProofSubscriber,
	)
	s.subscribe(
		p2p.ExitSubnetTopicFormat,
		s.validateVoluntaryExit,
		s.voluntaryExitSubscriber,
	)
	s.subscribe(
		p2p.ProposerSlashingSubnetTopicFormat,
		s.validateProposerSlashing,
		s.proposerSlashingSubscriber,
	)
	s.subscribe(
		p2p.AttesterSlashingSubnetTopicFormat,
		s.validateAttesterSlashing,
		s.attesterSlashingSubscriber,
	)
	if flags.Get().SubscribeToAllSubnets {
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
	base := p2p.GossipTopicMappings(topic, 0)
	if base == nil {
		panic(fmt.Sprintf("%s is not mapped to any message in GossipTopicMappings", topic))
	}
	return s.subscribeWithBase(s.addDigestToTopic(topic), validator, handle)
}

func (s *Service) subscribeWithBase(topic string, validator pubsub.ValidatorEx, handle subHandler) *pubsub.Subscription {
	topic += s.cfg.P2P.Encoding().ProtocolSuffix()
	log := log.WithField("topic", topic)

	if err := s.cfg.P2P.PubSub().RegisterTopicValidator(s.wrapAndReportValidation(topic, validator)); err != nil {
		log.WithError(err).Error("Could not register validator for topic")
		return nil
	}

	sub, err := s.cfg.P2P.SubscribeToTopic(topic)
	if err != nil {
		// Any error subscribing to a PubSub topic would be the result of a misconfiguration of
		// libp2p PubSub library or a subscription request to a topic that fails to match the topic
		// subscription filter.
		log.WithError(err).Error("Could not subscribe topic")
		return nil
	}

	// Pipeline decodes the incoming subscription data, runs the validation, and handles the
	// message.
	pipeline := func(msg *pubsub.Message) {
		ctx, cancel := context.WithTimeout(s.ctx, pubsubMessageTimeout)
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
			log.Debug("Received nil message on pubsub")
			messageFailedProcessingCounter.WithLabelValues(topic).Inc()
			return
		}

		if err := handle(ctx, msg.ValidatorData.(proto.Message)); err != nil {
			traceutil.AnnotateError(span, err)
			log.WithError(err).Debug("Could not handle p2p pubsub")
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
				if err != pubsub.ErrSubscriptionCancelled { // Only log a warning on unexpected errors.
					log.WithError(err).Warn("Subscription next failed")
				}
				// Cancel subscription in the event of an error, as we are
				// now exiting topic event loop.
				sub.Cancel()
				return
			}

			if msg.ReceivedFrom == s.cfg.P2P.PeerID() {
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
func (s *Service) wrapAndReportValidation(topic string, v pubsub.ValidatorEx) (string, pubsub.ValidatorEx) {
	return topic, func(ctx context.Context, pid peer.ID, msg *pubsub.Message) (res pubsub.ValidationResult) {
		defer messagehandler.HandlePanic(ctx, msg)
		res = pubsub.ValidationIgnore // Default: ignore any message that panics.
		ctx, cancel := context.WithTimeout(ctx, pubsubMessageTimeout)
		defer cancel()
		messageReceivedCounter.WithLabelValues(topic).Inc()
		if msg.Topic == nil {
			messageFailedValidationCounter.WithLabelValues(topic).Inc()
			return pubsub.ValidationReject
		}
		// Ignore any messages received before chainstart.
		if s.chainStarted.IsNotSet() {
			messageFailedValidationCounter.WithLabelValues(topic).Inc()
			return pubsub.ValidationIgnore
		}
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
	base := p2p.GossipTopicMappings(topic, 0)
	if base == nil {
		panic(fmt.Sprintf("%s is not mapped to any message in GossipTopicMappings", topic))
	}
	for i := uint64(0); i < params.BeaconNetworkConfig().AttestationSubnetCount; i++ {
		s.subscribeWithBase(s.addDigestAndIndexToTopic(topic, i), validator, handle)
	}
	genesis := s.cfg.Chain.GenesisTime()
	ticker := slotutil.NewSlotTicker(genesis, params.BeaconConfig().SecondsPerSlot)

	go func() {
		for {
			select {
			case <-s.ctx.Done():
				ticker.Done()
				return
			case <-ticker.C():
				if s.chainStarted.IsSet() && s.cfg.InitialSync.Syncing() {
					continue
				}
				// Check every slot that there are enough peers
				for i := uint64(0); i < params.BeaconNetworkConfig().AttestationSubnetCount; i++ {
					if !s.validPeersExist(s.addDigestAndIndexToTopic(topic, i)) {
						log.Debugf("No peers found subscribed to attestation gossip subnet with "+
							"committee index %d. Searching network for peers subscribed to the subnet.", i)
						_, err := s.cfg.P2P.FindPeersWithSubnet(
							s.ctx,
							s.addDigestAndIndexToTopic(topic, i),
							i,
							params.BeaconNetworkConfig().MinimumPeersInSubnet,
						)
						if err != nil {
							log.WithError(err).Debug("Could not search for peers")
							return
						}
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
	base := p2p.GossipTopicMappings(topicFormat, 0)
	if base == nil {
		log.Fatalf("%s is not mapped to any message in GossipTopicMappings", topicFormat)
	}
	digest, err := s.forkDigest()
	if err != nil {
		log.WithError(err).Fatal("Could not compute fork digest")
	}
	subscriptions := make(map[uint64]*pubsub.Subscription, params.BeaconConfig().MaxCommitteesPerSlot)
	genesis := s.cfg.Chain.GenesisTime()
	ticker := slotutil.NewSlotTicker(genesis, params.BeaconConfig().SecondsPerSlot)

	go func() {
		for {
			select {
			case <-s.ctx.Done():
				ticker.Done()
				return
			case currentSlot := <-ticker.C():
				if s.chainStarted.IsSet() && s.cfg.InitialSync.Syncing() {
					continue
				}
				wantedSubs := s.retrievePersistentSubs(currentSlot)
				// Resize as appropriate.
				s.reValidateSubscriptions(subscriptions, wantedSubs, topicFormat, digest)

				// subscribe desired aggregator subnets.
				for _, idx := range wantedSubs {
					s.subscribeAggregatorSubnet(subscriptions, idx, digest, validate, handle)
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
			fullTopic := fmt.Sprintf(topicFormat, digest, k) + s.cfg.P2P.Encoding().ProtocolSuffix()
			if err := s.cfg.P2P.PubSub().UnregisterTopicValidator(fullTopic); err != nil {
				log.WithError(err).Error("Could not unregister topic validator")
			}
			delete(subscriptions, k)
		}
	}
}

// subscribe missing subnets for our aggregators.
func (s *Service) subscribeAggregatorSubnet(
	subscriptions map[uint64]*pubsub.Subscription,
	idx uint64,
	digest [4]byte,
	validate pubsub.ValidatorEx,
	handle subHandler,
) {
	// do not subscribe if we have no peers in the same
	// subnet
	topic := p2p.GossipTypeMapping[reflect.TypeOf(&pb.Attestation{})]
	subnetTopic := fmt.Sprintf(topic, digest, idx)
	// check if subscription exists and if not subscribe the relevant subnet.
	if _, exists := subscriptions[idx]; !exists {
		subscriptions[idx] = s.subscribeWithBase(subnetTopic, validate, handle)
	}
	if !s.validPeersExist(subnetTopic) {
		log.Debugf("No peers found subscribed to attestation gossip subnet with "+
			"committee index %d. Searching network for peers subscribed to the subnet.", idx)
		_, err := s.cfg.P2P.FindPeersWithSubnet(s.ctx, subnetTopic, idx, params.BeaconNetworkConfig().MinimumPeersInSubnet)
		if err != nil {
			log.WithError(err).Debug("Could not search for peers")
		}
	}
}

// lookup peers for attester specific subnets.
func (s *Service) lookupAttesterSubnets(digest [4]byte, idx uint64) {
	topic := p2p.GossipTypeMapping[reflect.TypeOf(&pb.Attestation{})]
	subnetTopic := fmt.Sprintf(topic, digest, idx)
	if !s.validPeersExist(subnetTopic) {
		log.Debugf("No peers found subscribed to attestation gossip subnet with "+
			"committee index %d. Searching network for peers subscribed to the subnet.", idx)
		// perform a search for peers with the desired committee index.
		_, err := s.cfg.P2P.FindPeersWithSubnet(s.ctx, subnetTopic, idx, params.BeaconNetworkConfig().MinimumPeersInSubnet)
		if err != nil {
			log.WithError(err).Debug("Could not search for peers")
		}
	}
}

// find if we have peers who are subscribed to the same subnet
func (s *Service) validPeersExist(subnetTopic string) bool {
	numOfPeers := s.cfg.P2P.PubSub().ListPeers(subnetTopic + s.cfg.P2P.Encoding().ProtocolSuffix())
	return uint64(len(numOfPeers)) >= params.BeaconNetworkConfig().MinimumPeersInSubnet
}

func (s *Service) retrievePersistentSubs(currSlot types.Slot) []uint64 {
	// Persistent subscriptions from validators
	persistentSubs := s.persistentSubnetIndices()
	// Update desired topic indices for aggregator
	wantedSubs := s.aggregatorSubnetIndices(currSlot)

	// Combine subscriptions to get all requested subscriptions
	return sliceutil.SetUint64(append(persistentSubs, wantedSubs...))
}

// filters out required peers for the node to function, not
// pruning peers who are in our attestation subnets.
func (s *Service) filterNeededPeers(pids []peer.ID) []peer.ID {
	// Exit early if nothing to filter.
	if len(pids) == 0 {
		return pids
	}
	digest, err := s.forkDigest()
	if err != nil {
		log.WithError(err).Error("Could not compute fork digest")
		return pids
	}
	currSlot := s.cfg.Chain.CurrentSlot()
	wantedSubs := s.retrievePersistentSubs(currSlot)
	wantedSubs = sliceutil.SetUint64(append(wantedSubs, s.attesterSubnetIndices(currSlot)...))
	topic := p2p.GossipTypeMapping[reflect.TypeOf(&pb.Attestation{})]

	// Map of peers in subnets
	peerMap := make(map[peer.ID]bool)

	for _, sub := range wantedSubs {
		subnetTopic := fmt.Sprintf(topic, digest, sub) + s.cfg.P2P.Encoding().ProtocolSuffix()
		peers := s.cfg.P2P.PubSub().ListPeers(subnetTopic)
		if len(peers) > int(params.BeaconNetworkConfig().MinimumPeersInSubnet) {
			// In the event we have more than the minimum, we can
			// mark the remaining as viable for pruning.
			peers = peers[:params.BeaconNetworkConfig().MinimumPeersInSubnet]
		}
		// Add peer to peer map.
		for _, p := range peers {
			// Even if the peer id has
			// already been seen we still set
			// it, as the outcome is the same.
			peerMap[p] = true
		}
	}

	// Clear out necessary peers from the peers to prune.
	newPeers := make([]peer.ID, 0, len(pids))

	for _, pid := range pids {
		if peerMap[pid] {
			continue
		}
		newPeers = append(newPeers, pid)
	}
	return newPeers
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
	genRoot := s.cfg.Chain.GenesisValidatorRoot()
	return p2putils.CreateForkDigest(s.cfg.Chain.GenesisTime(), genRoot[:])
}
