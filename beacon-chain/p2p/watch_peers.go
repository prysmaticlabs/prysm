package p2p

import (
	"context"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/p2p/peers"
)

// ensurePeerConnections will attempt to reestablish connection to the peers
// if there are currently no connections to that peer.
func ensurePeerConnections(ctx context.Context, h host.Host, peers *peers.Status, relayNodes ...string) {
	// every time reset peersToWatch, add RelayNodes and trust peers
	var peersToWatch []string
	trustedPeers := peers.GetTrustedPeers()
	peersToWatch = append(peersToWatch, relayNodes...)
	for _, trustedPeer := range trustedPeers {
		address, err := peers.Address(trustedPeer)

		// avoid invalid trusted peers
		if err != nil || address == nil {
			continue
		}

		// any more appropriate way ?
		peer := address.String() + "/p2p/" + trustedPeer.String()
		peersToWatch = append(peersToWatch, peer)
	}

	if len(peersToWatch) == 0 {
		return
	}
	for _, p := range peersToWatch {
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
