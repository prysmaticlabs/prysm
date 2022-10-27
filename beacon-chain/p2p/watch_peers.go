package p2p

import (
	"context"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
)

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
		peerInfo, err := MakePeer(p)
		if err != nil {
			log.WithError(err).Error("Could not make peer")
			continue
		}

		c := h.Network().ConnsToPeer(peerInfo.ID)
		if len(c) == 0 {
			if err := connectWithTimeout(ctx, h, peerInfo); err != nil {
				log.WithField("peer", peerInfo.ID).WithField("addrs", peerInfo.Addrs).WithError(err).Errorf("Failed to reconnect to peer")
				continue
			}
		}
	}
}

func connectWithTimeout(ctx context.Context, h host.Host, peer *peer.AddrInfo) error {
	log.WithField("peer", peer.ID).Debug("No connections to peer, reconnecting")
	ctx, cancel := context.WithTimeout(ctx, maxDialTimeout)
	defer cancel()
	return h.Connect(ctx, *peer)
}
