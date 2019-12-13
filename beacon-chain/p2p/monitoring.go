package p2p

import (
	"time"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prysmaticlabs/prysm/shared/runutil"
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
	runutil.RunEvery(s.ctx, time.Duration(10*time.Second), s.updateP2PTopicPeerCount)
}

func peerCount(h host.Host) int {
	return len(h.Network().Peers())
}

func (s *Service) updateP2PTopicPeerCount() {
	for topic := range GossipTopicMappings {
		topic += s.Encoding().ProtocolSuffix()
		p2pTopicPeerCount.WithLabelValues(topic).Set(float64(len(s.pubsub.ListPeers(topic))))
	}
}
