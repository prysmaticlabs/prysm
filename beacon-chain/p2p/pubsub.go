package p2p

import (
	"context"
	"encoding/base64"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	pubsub_pb "github.com/libp2p/go-libp2p-pubsub/pb"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// JoinTopic will join PubSub topic, if not already joined.
func (s *Service) JoinTopic(topic string, opts ...pubsub.TopicOpt) (*pubsub.Topic, error) {
	s.joinedTopicsLock.Lock()
	defer s.joinedTopicsLock.Unlock()

	if _, ok := s.joinedTopics[topic]; !ok {
		topicHandle, err := s.pubsub.Join(topic, opts...)
		if err != nil {
			return nil, err
		}
		s.joinedTopics[topic] = topicHandle
	}

	return s.joinedTopics[topic], nil
}

// LeaveTopic closes topic and removes corresponding handler from list of joined topics.
// This method will return error if there are outstanding event handlers or subscriptions.
func (s *Service) LeaveTopic(topic string) error {
	s.joinedTopicsLock.Lock()
	defer s.joinedTopicsLock.Unlock()

	if t, ok := s.joinedTopics[topic]; ok {
		if err := t.Close(); err != nil {
			return err
		}
		delete(s.joinedTopics, topic)
	}
	return nil
}

// PublishToTopic joins (if necessary) and publishes a message to a PubSub topic.
func (s *Service) PublishToTopic(ctx context.Context, topic string, data []byte, opts ...pubsub.PubOpt) error {
	topicHandle, err := s.JoinTopic(topic)
	if err != nil {
		return err
	}

	// If feature flag isn't enabled, don't wait for peers to be present.
	if !featureconfig.Get().EnableAttBroadcastDiscoveryAttempts {
		return topicHandle.Publish(ctx, data, opts...)
	}

	// Wait for at least 1 peer to be available to receive the published message.
	for {
		if len(topicHandle.ListPeers()) > 0 {
			return topicHandle.Publish(ctx, data, opts...)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// SubscribeToTopic joins (if necessary) and subscribes to PubSub topic.
func (s *Service) SubscribeToTopic(topic string, opts ...pubsub.SubOpt) (*pubsub.Subscription, error) {
	topicHandle, err := s.JoinTopic(topic)
	if err != nil {
		return nil, err
	}
	scoringParams := topicScoreParams(topic)
	if scoringParams != nil {
		if err = topicHandle.SetScoreParams(scoringParams); err != nil {
			return nil, err
		}
	}
	return topicHandle.Subscribe(opts...)
}

func peerInspector(peerMap map[peer.ID]*pubsub.PeerScoreSnapshot) {
	for id, snap := range peerMap {
		log.Debugf("Peer id %s with score %f", id.String(), snap.Score)
	}
}

func peerScoringParams() (*pubsub.PeerScoreParams, *pubsub.PeerScoreThresholds) {
	thresholds := &pubsub.PeerScoreThresholds{
		GossipThreshold:             -4000,
		PublishThreshold:            -8000,
		GraylistThreshold:           -16000,
		AcceptPXThreshold:           100,
		OpportunisticGraftThreshold: 5,
	}
	scoreParams := &pubsub.PeerScoreParams{
		Topics:        make(map[string]*pubsub.TopicScoreParams),
		TopicScoreCap: 32.72,
		AppSpecificScore: func(p peer.ID) float64 {
			return 1
		},
		AppSpecificWeight:           1,
		IPColocationFactorWeight:    -1,
		IPColocationFactorThreshold: 10,
		IPColocationFactorWhitelist: nil,
		BehaviourPenaltyWeight:      -99,
		BehaviourPenaltyThreshold:   0,
		BehaviourPenaltyDecay:       0.994,
		DecayInterval:               1 * time.Second,
		DecayToZero:                 0.1,
		RetainScore:                 100,
	}
	return scoreParams, thresholds
}

func topicScoreParams(topic string) *pubsub.TopicScoreParams {
	switch true {
	case strings.Contains(topic, "beacon_block"):
		return defaultBlockTopicParams()
	case strings.Contains(topic, "beacon_aggregate_and_proof"):
		return defaultAggregateTopicParams()
	default:
		return nil
	}
}

// Content addressable ID function.
//
// ETH2 spec defines the message ID as:
//    message-id: base64(SHA256(message.data))
// where base64 is the URL-safe base64 alphabet with
// padding characters omitted.
func msgIDFunction(pmsg *pubsub_pb.Message) string {
	h := hashutil.Hash(pmsg.Data)
	return base64.RawURLEncoding.EncodeToString(h[:])
}

func setPubSubParameters() {
	pubsub.GossipSubDlo = 5
	pubsub.GossipSubHeartbeatInterval = 700 * time.Millisecond
	pubsub.GossipSubHistoryLength = 6
	pubsub.GossipSubHistoryGossip = 3
}

func defaultBlockTopicParams() *pubsub.TopicScoreParams {
	return &pubsub.TopicScoreParams{
		TopicWeight:                     0.5,
		TimeInMeshWeight:                0.0324,
		TimeInMeshQuantum:               1,
		TimeInMeshCap:                   300,
		FirstMessageDeliveriesWeight:    1,
		FirstMessageDeliveriesDecay:     0.9928,
		FirstMessageDeliveriesCap:       23,
		MeshMessageDeliveriesWeight:     -0.020408,
		MeshMessageDeliveriesDecay:      0.9928,
		MeshMessageDeliveriesCap:        35,
		MeshMessageDeliveriesThreshold:  139,
		MeshMessageDeliveriesWindow:     200 * time.Millisecond,
		MeshMessageDeliveriesActivation: time.Duration(8*params.BeaconConfig().SlotsPerEpoch*params.BeaconConfig().SecondsPerSlot) * time.Second,
		MeshFailurePenaltyWeight:        -0.02048,
		MeshFailurePenaltyDecay:         0.9928,
		InvalidMessageDeliveriesWeight:  -99,
		InvalidMessageDeliveriesDecay:   0.9994,
	}
}

func defaultAggregateTopicParams() *pubsub.TopicScoreParams {
	return &pubsub.TopicScoreParams{
		TopicWeight:                     0.5,
		TimeInMeshWeight:                0.0324,
		TimeInMeshQuantum:               1,
		TimeInMeshCap:                   300,
		FirstMessageDeliveriesWeight:    0.05,
		FirstMessageDeliveriesDecay:     0.631,
		FirstMessageDeliveriesCap:       463,
		MeshMessageDeliveriesWeight:     -0.0026,
		MeshMessageDeliveriesDecay:      0.631,
		MeshMessageDeliveriesCap:        98,
		MeshMessageDeliveriesThreshold:  390,
		MeshMessageDeliveriesWindow:     200 * time.Millisecond,
		MeshMessageDeliveriesActivation: time.Duration(4*params.BeaconConfig().SecondsPerSlot) * time.Second,
		MeshFailurePenaltyWeight:        -0.0026,
		MeshFailurePenaltyDecay:         0.631,
		InvalidMessageDeliveriesWeight:  -99,
		InvalidMessageDeliveriesDecay:   0.994,
	}
}
