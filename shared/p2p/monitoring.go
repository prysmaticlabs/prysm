package p2p

import (
	"context"
	"time"

	host "github.com/libp2p/go-libp2p-host"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	peerCountMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "p2p_peer_count",
		Help: "The number of currently connected peers",
	})
)

func init() {
	prometheus.MustRegister(peerCountMetric)
}

func startPeerWatcher(ctx context.Context, h host.Host) {

	go (func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				peerCountMetric.Set(float64(peerCount(h)))

				// Wait 1 second to update again
				time.Sleep(1 * time.Second)
			}
		}
	})()
}

func peerCount(h host.Host) int {
	return len(h.Network().Peers())
}
