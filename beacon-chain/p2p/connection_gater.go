package p2p

import (
	"net"

	"github.com/libp2p/go-libp2p-core/control"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
	filter "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr-net"
	"github.com/sirupsen/logrus"
)

// InterceptPeerDial tests whether we're permitted to Dial the specified peer.
func (s *Service) InterceptPeerDial(p peer.ID) (allow bool) {
	return true
}

// InterceptAddrDial tests whether we're permitted to dial the specified
// multiaddr for the given peer.
func (s *Service) InterceptAddrDial(_ peer.ID, m multiaddr.Multiaddr) (allow bool) {
	return filterConnections(s.addrFilter, m)
}

// InterceptAccept tests whether an incipient inbound connection is allowed.
func (s *Service) InterceptAccept(n network.ConnMultiaddrs) (allow bool) {
	if s.isPeerAtLimit() {
		log.WithFields(logrus.Fields{"peer": n.RemoteMultiaddr(),
			"reason": "at peer limit"}).Trace("Not accepting inbound dial")
		return false
	}
	return filterConnections(s.addrFilter, n.RemoteMultiaddr())
}

// InterceptSecured tests whether a given connection, now authenticated,
// is allowed.
func (s *Service) InterceptSecured(_ network.Direction, _ peer.ID, n network.ConnMultiaddrs) (allow bool) {
	return true
}

// InterceptUpgraded tests whether a fully capable connection is allowed.
func (s *Service) InterceptUpgraded(n network.Conn) (allow bool, reason control.DisconnectReason) {
	return true, 0
}

// configureFilter looks at the provided allow lists and
// deny lists to appropriately create a filter.
func configureFilter(cfg *Config) (*filter.Filters, error) {
	addrFilter := filter.NewFilters()
	// Configure from provided allow list in the config.
	if cfg.AllowListCIDR != "" {
		_, ipnet, err := net.ParseCIDR(cfg.AllowListCIDR)
		if err != nil {
			return nil, err
		}
		addrFilter.AddFilter(*ipnet, filter.ActionAccept)
	}
	// Configure from provided deny list in the config.
	if len(cfg.DenyListCIDR) > 0 {
		for _, cidr := range cfg.DenyListCIDR {
			_, ipnet, err := net.ParseCIDR(cidr)
			if err != nil {
				return nil, err
			}
			addrFilter.AddFilter(*ipnet, filter.ActionDeny)
		}
	}
	return addrFilter, nil
}

// filterConnections checks the appropriate ip subnets from our
// filter and decides what to do with them. By default libp2p
// accepts all incoming dials, so if we have an allow list
// we will reject all inbound dials except for those in the
// appropriate ip subnets.
func filterConnections(f *filter.Filters, a filter.Multiaddr) bool {
	acceptedNets := f.FiltersForAction(filter.ActionAccept)
	restrictConns := len(acceptedNets) != 0

	// If we have an allow list added in, we by default reject all
	// connection attempts except for those coming in from the
	// appropriate ip subnets.
	if restrictConns {
		ip, err := manet.ToIP(a)
		if err != nil {
			log.Tracef("Multiaddress has invalid ip: %v", err)
			return false
		}
		found := false
		for _, ipnet := range acceptedNets {
			if ipnet.Contains(ip) {
				found = true
				break
			}
		}
		return found
	}
	return !f.AddrBlocked(a)
}
