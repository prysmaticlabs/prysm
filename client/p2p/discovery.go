package p2p

import (
	"context"
	"time"

	host "github.com/libp2p/go-libp2p-host"
	peer "github.com/libp2p/go-libp2p-peer"
	ps "github.com/libp2p/go-libp2p-peerstore"
	mdns "github.com/libp2p/go-libp2p/p2p/discovery"
	pb "github.com/prysmaticlabs/prysm/proto/sharding/v1"
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
// TODO: add other discovery protocols such as DHT, etc.
func startDiscovery(ctx context.Context, host host.Host, gsub topicPeerLister) error {
	mdnsService, err := mdns.NewMdnsService(ctx, host, discoveryInterval, mDNSTag)
	if err != nil {
		return err
	}

	mdnsService.RegisterNotifee(&discovery{ctx, host, gsub})

	return nil
}

// topicPeerLister has a method to return connected peers on a given topic.
// This is implemented by floodsub.PubSub.
type topicPeerLister interface {
	ListPeers(string) []peer.ID
}

// Discovery implements mDNS notifee interface.
type discovery struct {
	ctx  context.Context
	host host.Host

	// Required for helper method.
	gsub topicPeerLister
}

// HandlePeerFound registers the peer with the host.
func (d *discovery) HandlePeerFound(pi ps.PeerInfo) {
	d.host.Peerstore().AddAddrs(pi.ID, pi.Addrs, ps.PermanentAddrTTL)
	if err := d.host.Connect(d.ctx, pi); err != nil {
		log.Warnf("Failed to connect to peer: %v", err)
	}

	log.WithFields(logrus.Fields{
		"peers": d.host.Peerstore().Peers(),
	}).Debug("Peers are now")

	log.WithFields(logrus.Fields{
		"peerMap": d.topicPeerMap(),
	}).Debug("Gsub has peers")
}

// topicPeerMap helper function for inspecting which peers are available for
// the p2p topics.
func (d *discovery) topicPeerMap() map[pb.Topic][]peer.ID {
	m := make(map[pb.Topic][]peer.ID)
	for topic := range topicTypeMapping {
		peers := d.gsub.ListPeers(topic.String())
		m[topic] = peers
	}
	return m
}
