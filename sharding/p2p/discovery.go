package p2p

import (
	"context"
	"fmt"
	"time"

	pb "github.com/ethereum/go-ethereum/sharding/p2p/proto"
	floodsub "github.com/libp2p/go-floodsub"
	host "github.com/libp2p/go-libp2p-host"
	peer "github.com/libp2p/go-libp2p-peer"
	ps "github.com/libp2p/go-libp2p-peerstore"
	mdns "github.com/libp2p/go-libp2p/p2p/discovery"
)

func startDiscovery(ctx context.Context, host host.Host, gsub *floodsub.PubSub) error {
	mdnsService, err := mdns.NewMdnsService(ctx, host, 60*time.Second, "")
	if err != nil {
		return err
	}

	mdnsService.RegisterNotifee(&discovery{host, gsub, ctx})

	return nil
}

type discovery struct {
	host host.Host
	gsub *floodsub.PubSub
	ctx  context.Context
}

func (d *discovery) HandlePeerFound(pi ps.PeerInfo) {
	d.host.Peerstore().AddAddrs(pi.ID, pi.Addrs, ps.PermanentAddrTTL)
	if err := d.host.Connect(d.ctx, pi); err != nil {
		logger.Warn(fmt.Sprintf("Failed to connect to peer: %v", err))
	}

	logger.Debug(fmt.Sprintf("Peers now: %s", d.host.Peerstore().Peers()))
	logger.Debug(fmt.Sprintf("gsub has peers: %v", d.topicPeerMap()))
}

func (d *discovery) topicPeerMap() map[pb.Message_Topic][]peer.ID {
	m := make(map[pb.Message_Topic][]peer.ID)
	for topic := range topicTypeMapping {
		peers := d.gsub.ListPeers(topic.String())
		m[topic] = peers
	}
	return m
}
