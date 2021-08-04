package sync

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/cmd/beacon-chain/flags"
	pb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

var (
	topicPeerCount = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "p2p_topic_peer_count",
			Help: "The number of peers subscribed to a given topic.",
		}, []string{"topic"},
	)
	subscribedTopicPeerCount = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "p2p_subscribed_topic_peer_count",
			Help: "The number of peers subscribed to topics that a host node is also subscribed to.",
		}, []string{"topic"},
	)
	messageReceivedCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "p2p_message_received_total",
			Help: "Count of messages received.",
		},
		[]string{"topic"},
	)
	messageFailedValidationCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "p2p_message_failed_validation_total",
			Help: "Count of messages that failed validation.",
		},
		[]string{"topic"},
	)
	messageIgnoredValidationCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "p2p_message_ignored_validation_total",
			Help: "Count of messages that were ignored in validation.",
		},
		[]string{"topic"},
	)
	messageFailedProcessingCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "p2p_message_failed_processing_total",
			Help: "Count of messages that passed validation but failed processing.",
		},
		[]string{"topic"},
	)
	numberOfTimesResyncedCounter = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "number_of_times_resynced",
			Help: "Count the number of times a node resyncs.",
		},
	)

	arrivalBlockPropagationHistogram = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "block_arrival_latency_milliseconds",
			Help:    "Captures blocks propagation time. Blocks arrival in milliseconds distribution",
			Buckets: []float64{250, 500, 1000, 1500, 2000, 4000, 8000, 16000},
		},
	)
)

func (s *Service) updateMetrics() {
	// do not update metrics if genesis time
	// has not been initialized
	if s.cfg.Chain.GenesisTime().IsZero() {
		return
	}
	// We update the dynamic subnet topics.
	digest, err := s.currentForkDigest()
	if err != nil {
		log.WithError(err).Debugf("Could not compute fork digest")
	}
	indices := s.aggregatorSubnetIndices(s.cfg.Chain.CurrentSlot())
	syncIndices := cache.SyncSubnetIDs.GetAllSubnets(helpers.SlotToEpoch(s.cfg.Chain.CurrentSlot()))
	attTopic := p2p.GossipTypeMapping[reflect.TypeOf(&pb.Attestation{})]
	syncTopic := p2p.GossipTypeMapping[reflect.TypeOf(&pb.SyncCommitteeMessage{})]
	attTopic += s.cfg.P2P.Encoding().ProtocolSuffix()
	syncTopic += s.cfg.P2P.Encoding().ProtocolSuffix()
	if flags.Get().SubscribeToAllSubnets {
		for i := uint64(0); i < params.BeaconNetworkConfig().AttestationSubnetCount; i++ {
			formattedTopic := fmt.Sprintf(attTopic, digest, i)
			topicPeerCount.WithLabelValues(formattedTopic).Set(float64(len(s.cfg.P2P.PubSub().ListPeers(formattedTopic))))
		}
		for i := uint64(0); i < params.BeaconConfig().SyncCommitteeSubnetCount; i++ {
			formattedTopic := fmt.Sprintf(syncTopic, digest, i)
			topicPeerCount.WithLabelValues(formattedTopic).Set(float64(len(s.cfg.P2P.PubSub().ListPeers(formattedTopic))))
		}
	} else {
		for _, committeeIdx := range indices {
			formattedTopic := fmt.Sprintf(attTopic, digest, committeeIdx)
			topicPeerCount.WithLabelValues(formattedTopic).Set(float64(len(s.cfg.P2P.PubSub().ListPeers(formattedTopic))))
		}
		for _, committeeIdx := range syncIndices {
			formattedTopic := fmt.Sprintf(syncTopic, digest, committeeIdx)
			topicPeerCount.WithLabelValues(formattedTopic).Set(float64(len(s.cfg.P2P.PubSub().ListPeers(formattedTopic))))
		}
	}

	// We update all other gossip topics.
	for _, topic := range p2p.AllTopics() {
		// We already updated attestation subnet topics.
		if strings.Contains(topic, p2p.GossipAttestationMessage) || strings.Contains(topic, p2p.GossipSyncCommitteeMessage) {
			continue
		}
		topic += s.cfg.P2P.Encoding().ProtocolSuffix()
		if !strings.Contains(topic, "%x") {
			topicPeerCount.WithLabelValues(topic).Set(float64(len(s.cfg.P2P.PubSub().ListPeers(topic))))
			continue
		}
		formattedTopic := fmt.Sprintf(topic, digest)
		topicPeerCount.WithLabelValues(formattedTopic).Set(float64(len(s.cfg.P2P.PubSub().ListPeers(formattedTopic))))
	}

	for _, topic := range s.cfg.P2P.PubSub().GetTopics() {
		subscribedTopicPeerCount.WithLabelValues(topic).Set(float64(len(s.cfg.P2P.PubSub().ListPeers(topic))))
	}
}
