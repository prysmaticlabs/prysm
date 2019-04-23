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

// starPeerWatcher updates the peer count metric and calls to reconnect any VIP
// peers such as the bootnode peer or relay node peer.
func startPeerWatcher(ctx context.Context, h host.Host, reconnectPeers ...string) {

	go (func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				peerCountMetric.Set(float64(peerCount(h)))
				ensurePeerConnections(ctx, h, reconnectPeers...)

				// Wait 5 second to update again
				time.Sleep(5 * time.Second)
			}
		}
	})()
}

func peerCount(h host.Host) int {
	return len(h.Network().Peers())
}

// ensurePeerConnections will attempt to reestablish connection to the peers
// if there are currently no connections to that peer.
func ensurePeerConnections(ctx context.Context, h host.Host, peers ...string) {
	if len(peers) == 0 {
		return
	}
	for _, p := range peers {
		if p == "" {
			continue
		}
		peer, err := MakePeer(p)
		if err != nil {
			log.Errorf("Could not make peer: %v", err)
			continue
		}

		c := h.Network().ConnsToPeer(peer.ID)
		if len(c) == 0 {
			log.WithField("peer", peer.ID).Debug("No connections to peer, reconnecting")
			ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()
			if err := h.Connect(ctx, *peer); err != nil {
				log.Errorf("Failed to reconnect to peer %v", err)
				continue
			}
		}
	}
}
