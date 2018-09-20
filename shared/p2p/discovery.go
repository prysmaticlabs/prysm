package p2p

import (
	"context"
	"time"

	host "github.com/libp2p/go-libp2p-host"
	ps "github.com/libp2p/go-libp2p-peerstore"
	mdns "github.com/libp2p/go-libp2p/p2p/discovery"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "p2p")

// Discovery interval for multicast DNS querying.
var discoveryInterval = 1 * time.Minute

// mDNSTag is the name of the mDNS service.
var mDNSTag = mdns.ServiceTag

// startDiscovery protocols. Currently, this supports discovery via multicast
// DNS peer discovery.
//
// TODO(287): add other discovery protocols such as DHT, etc.
func startDiscovery(ctx context.Context, host host.Host) error {
	mdnsService, err := mdns.NewMdnsService(ctx, host, discoveryInterval, mDNSTag)
	if err != nil {
		return err
	}

	mdnsService.RegisterNotifee(&discovery{ctx, host})
	return nil
}

// Discovery implements mDNS notifee interface.
type discovery struct {
	ctx  context.Context
	host host.Host
}

// HandlePeerFound registers the peer with the host.
func (d *discovery) HandlePeerFound(pi ps.PeerInfo) {
	log.WithFields(logrus.Fields{
		"peer addrs": pi.Addrs,
		"peer id":    pi.ID,
	}).Debug("Attempting to connect to a peer")

	d.host.Peerstore().AddAddrs(pi.ID, pi.Addrs, ps.PermanentAddrTTL)
	if err := d.host.Connect(d.ctx, pi); err != nil {
		log.Warnf("Failed to connect to peer: %v", err)
	}

	log.WithFields(logrus.Fields{
		"peers": d.host.Peerstore().Peers(),
	}).Debug("Peers are now")
}
