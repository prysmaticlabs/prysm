package p2p

import (
	"context"
	"time"

	iaddr "github.com/ipfs/go-ipfs-addr"
	host "github.com/libp2p/go-libp2p-host"
	ps "github.com/libp2p/go-libp2p-peerstore"
	mdns "github.com/libp2p/go-libp2p/p2p/discovery"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

var log = logrus.WithField("prefix", "p2p")

// Discovery interval for multicast DNS querying.
var discoveryInterval = 1 * time.Minute

// mDNSTag is the name of the mDNS service.
var mDNSTag = mdns.ServiceTag

// TODO(287): add other discovery protocols such as DHT, etc.
// startmDNSDiscovery supports discovery via multicast DNS peer discovery.
func startmDNSDiscovery(ctx context.Context, host host.Host) error {
	mdnsService, err := mdns.NewMdnsService(ctx, host, discoveryInterval, mDNSTag)
	if err != nil {
		return err
	}

	mdnsService.RegisterNotifee(&discovery{ctx, host})
	return nil
}

// startDHTDiscovery supports discovery via DHT.
func startDHTDiscovery(ctx context.Context, host host.Host, bootstrapAddr string) error {
	ctx, span := trace.StartSpan(ctx, "p2p_startDHTDiscovery")
	defer span.End()

	peerinfo, err := peerInfoFromAddr(bootstrapAddr)
	if err != nil {
		return err
	}

	err = host.Connect(ctx, *peerinfo)
	return err
}

func peerInfoFromAddr(address string) (*ps.PeerInfo, error) {
	addr, err := iaddr.ParseString(address)
	if err != nil {
		return nil, err
	}
	return ps.InfoFromP2pAddr(addr.Multiaddr())
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
		"peers": d.host.Network().Peers(),
	}).Debug("Peers are now")
}
