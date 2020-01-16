package p2p

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	p2pTopicPeerCount = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "p2p_topic_peer_count",
		Help: "The number of peers subscribed to a given topic.",
	},
		[]string{"topic"})
	p2pPeerCount = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "p2p_peer_count",
		Help: "The number of peers in a given state.",
	},
		[]string{"state"})
)

func (s *Service) updateMetrics() {
	for topic := range GossipTopicMappings {
		topic += s.Encoding().ProtocolSuffix()
		p2pTopicPeerCount.WithLabelValues(topic).Set(float64(len(s.pubsub.ListPeers(topic))))
	}
	p2pPeerCount.WithLabelValues("Connected").Set(float64(len(s.peers.Connected())))
	p2pPeerCount.WithLabelValues("Disconnected").Set(float64(len(s.peers.Disconnected())))
	p2pPeerCount.WithLabelValues("Connecting").Set(float64(len(s.peers.Connecting())))
	p2pPeerCount.WithLabelValues("Disconnecting").Set(float64(len(s.peers.Disconnecting())))
	p2pPeerCount.WithLabelValues("Bad").Set(float64(len(s.peers.Bad())))
}
