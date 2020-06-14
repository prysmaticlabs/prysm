package p2p

import (
	"github.com/libp2p/go-libp2p-core/control"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/sirupsen/logrus"
)

// InterceptPeerDial tests whether we're permitted to Dial the specified peer.
func (s *Service) InterceptPeerDial(p peer.ID) (allow bool) {
	return true
}

// InterceptAddrDial tests whether we're permitted to dial the specified
// multiaddr for the given peer.
func (s *Service) InterceptAddrDial(peer.ID, multiaddr.Multiaddr) (allow bool) {
	return true
}

// InterceptAccept tests whether an incipient inbound connection is allowed.
func (s *Service) InterceptAccept(n network.ConnMultiaddrs) (allow bool) {
	if len(s.Peers().Active()) >= int(s.cfg.MaxPeers) {
		log.WithFields(logrus.Fields{"peer": n.RemoteMultiaddr(),
			"reason": "at peer limit"}).Trace("Not accepting inbound dial")
		return false
	}
	return true
}

// InterceptSecured tests whether a given connection, now authenticated,
// is allowed.
func (s *Service) InterceptSecured(network.Direction, peer.ID, network.ConnMultiaddrs) (allow bool) {
	return true
}

// InterceptUpgraded tests whether a fully capable connection is allowed.
func (s *Service) InterceptUpgraded(network.Conn) (allow bool, reason control.DisconnectReason) {
	return true, 0
}
