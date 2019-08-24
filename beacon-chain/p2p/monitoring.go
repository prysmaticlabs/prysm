package p2p

import (
	host "github.com/libp2p/go-libp2p-host"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)


func registerMetrics(h host.Host) {
	promauto.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "p2p_peer_count",
		Help: "The number of currently connected peers",
	}, func() float64 {
		return float64(peerCount(h))
	})
}

func peerCount(h host.Host) int {
	return len(h.Network().Peers())
}
