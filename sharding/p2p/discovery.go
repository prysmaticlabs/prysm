package p2p

import (
	"context"
	"fmt"

	pb "github.com/ethereum/go-ethereum/sharding/p2p/proto"
	floodsub "github.com/libp2p/go-floodsub"
	host "github.com/libp2p/go-libp2p-host"
	peer "github.com/libp2p/go-libp2p-peer"
	ps "github.com/libp2p/go-libp2p-peerstore"
)

type thing struct {
	host host.Host
	gsub *floodsub.PubSub
	ctx  context.Context
}

func (t *thing) HandlePeerFound(pi ps.PeerInfo) {
	t.host.Peerstore().AddAddrs(pi.ID, pi.Addrs, ps.PermanentAddrTTL)
	if err := t.host.Connect(t.ctx, pi); err != nil {
		logger.Warn(fmt.Sprintf("Failed to connect to peer: %v", err))
	}

	logger.Debug(fmt.Sprintf("Peers now: %s", t.host.Peerstore().Peers()))
	logger.Debug(fmt.Sprintf("gsub has peers: %v", t.TopicPeerMap()))
}

func (t *thing) TopicPeerMap() map[pb.Message_Topic][]peer.ID {
	m := make(map[pb.Message_Topic][]peer.ID)
	for topic := range topicTypeMapping {
		peers := t.gsub.ListPeers(topic.String())
		m[topic] = peers
	}
	return m
}
