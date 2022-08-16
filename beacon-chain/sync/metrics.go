package sync

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/v3/cmd/beacon-chain/flags"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	pb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
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
			Name: "p2p_subscribed_topic_peer_total",
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
	duplicatesRemovedCounter = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "number_of_duplicates_removed",
			Help: "Count the number of times a duplicate signature set has been removed.",
		},
	)
	numberOfSetsAggregated = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "number_of_sets_aggregated",
			Help:    "Count the number of times different sets have been successfully aggregated in a batch.",
			Buckets: []float64{10, 50, 100, 200, 400, 800, 1600, 3200},
		},
	)
	rpcBlocksByRangeResponseLatency = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "rpc_blocks_by_range_response_latency_milliseconds",
			Help:    "Captures total time to respond to rpc blocks by range requests in a milliseconds distribution",
			Buckets: []float64{5, 10, 50, 100, 150, 250, 500, 1000, 2000},
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
	if s.cfg.chain.GenesisTime().IsZero() {
		return
	}
	// We update the dynamic subnet topics.
	digest, err := s.currentForkDigest()
	if err != nil {
		log.WithError(err).Debugf("Could not compute fork digest")
	}
	indices := s.aggregatorSubnetIndices(s.cfg.chain.CurrentSlot())
	syncIndices := cache.SyncSubnetIDs.GetAllSubnets(slots.ToEpoch(s.cfg.chain.CurrentSlot()))
	attTopic := p2p.GossipTypeMapping[reflect.TypeOf(&pb.Attestation{})]
	syncTopic := p2p.GossipTypeMapping[reflect.TypeOf(&pb.SyncCommitteeMessage{})]
	attTopic += s.cfg.p2p.Encoding().ProtocolSuffix()
	syncTopic += s.cfg.p2p.Encoding().ProtocolSuffix()
	if flags.Get().SubscribeToAllSubnets {
		for i := uint64(0); i < params.BeaconNetworkConfig().AttestationSubnetCount; i++ {
			s.collectMetricForSubnet(attTopic, digest, i)
		}
		for i := uint64(0); i < params.BeaconConfig().SyncCommitteeSubnetCount; i++ {
			s.collectMetricForSubnet(syncTopic, digest, i)
		}
	} else {
		for _, committeeIdx := range indices {
			s.collectMetricForSubnet(attTopic, digest, committeeIdx)
		}
		for _, committeeIdx := range syncIndices {
			s.collectMetricForSubnet(syncTopic, digest, committeeIdx)
		}
	}

	// We update all other gossip topics.
	for _, topic := range p2p.AllTopics() {
		// We already updated attestation subnet topics.
		if strings.Contains(topic, p2p.GossipAttestationMessage) || strings.Contains(topic, p2p.GossipSyncCommitteeMessage) {
			continue
		}
		topic += s.cfg.p2p.Encoding().ProtocolSuffix()
		if !strings.Contains(topic, "%x") {
			topicPeerCount.WithLabelValues(topic).Set(float64(len(s.cfg.p2p.PubSub().ListPeers(topic))))
			continue
		}
		formattedTopic := fmt.Sprintf(topic, digest)
		topicPeerCount.WithLabelValues(formattedTopic).Set(float64(len(s.cfg.p2p.PubSub().ListPeers(formattedTopic))))
	}

	for _, topic := range s.cfg.p2p.PubSub().GetTopics() {
		subscribedTopicPeerCount.WithLabelValues(topic).Set(float64(len(s.cfg.p2p.PubSub().ListPeers(topic))))
	}
}

func (s *Service) collectMetricForSubnet(topic string, digest [4]byte, index uint64) {
	formattedTopic := fmt.Sprintf(topic, digest, index)
	topicPeerCount.WithLabelValues(formattedTopic).Set(float64(len(s.cfg.p2p.PubSub().ListPeers(formattedTopic))))
}
