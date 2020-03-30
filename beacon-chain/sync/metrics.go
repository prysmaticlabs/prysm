package sync

import (
	"fmt"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
)

var (
	topicPeerCount = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "p2p_topic_peer_count",
			Help: "The number of peers subscribed to a given topic.",
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
	numberOfBlocksRecoveredFromAtt = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "beacon_blocks_recovered_from_attestation_total",
			Help: "Count the number of times a missing block recovered from attestation vote.",
		},
	)
	numberOfBlocksNotRecoveredFromAtt = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "beacon_blocks_not_recovered_from_attestation_total",
			Help: "Count the number of times a missing block not recovered and pruned from attestation vote.",
		},
	)
	numberOfAttsRecovered = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "beacon_attestations_recovered_total",
			Help: "Count the number of times attestation recovered because of missing block",
		},
	)
	numberOfAttsNotRecovered = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "beacon_attestations_not_recovered_total",
			Help: "Count the number of times attestation not recovered and pruned because of missing block",
		},
	)
)

func (r *Service) updateMetrics() {
	for topic := range p2p.GossipTopicMappings {
		topic += r.p2p.Encoding().ProtocolSuffix()
		if !strings.Contains(topic, "%x") {
			topicPeerCount.WithLabelValues(topic).Set(float64(len(r.p2p.ListPeers(topic))))
			continue
		}
		digest, err := r.p2p.ForkDigest()
		if err != nil {
			log.WithError(err).Errorf("Could not compute fork digest")
		}
		topic = fmt.Sprintf(topic, digest)
		topicPeerCount.WithLabelValues(topic).Set(float64(len(r.p2p.ListPeers(topic))))
	}
}
