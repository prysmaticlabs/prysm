package p2p

import (
	"time"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	p2pTopicPeerCount = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "p2p_topic_peer_count",
		Help: "The number of peers subscribed to a topic",
	},
		[]string{"topic"})
)

func registerMetrics(s *Service) {

	// Metrics with a single value can use GaugeFunc, CounterFunc, etc.
	if err := prometheus.DefaultRegisterer.Register(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "p2p_peer_count",
		Help: "The number of currently connected peers",
	}, func() float64 {
		return float64(peerCount(s.host))
	})); err != nil {
		// This should only happen in tests.
		log.WithError(err).Error("Failed to register metric")
	}

	// Metrics with labels, polled every 10s.
	go func() {
		for {
			updateP2PTopicPeerCount(s)

			time.Sleep(10 * time.Second)
		}
	}()
}

func peerCount(h host.Host) int {
	return len(h.Network().Peers())
}

func updateP2PTopicPeerCount(s *Service) {
	for topic := range GossipTopicMappings {
		topic += s.Encoding().ProtocolSuffix()
		p2pTopicPeerCount.WithLabelValues(topic).Set(float64(len(s.pubsub.ListPeers(topic))))
	}
}
