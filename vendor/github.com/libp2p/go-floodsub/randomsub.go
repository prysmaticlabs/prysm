package floodsub

import (
	"context"

	pb "github.com/libp2p/go-floodsub/pb"

	host "github.com/libp2p/go-libp2p-host"
	peer "github.com/libp2p/go-libp2p-peer"
	protocol "github.com/libp2p/go-libp2p-protocol"
)

const (
	RandomSubID = protocol.ID("/randomsub/1.0.0")
)

var (
	RandomSubD = 6
)

// NewRandomSub returns a new PubSub object using RandomSubRouter as the router
func NewRandomSub(ctx context.Context, h host.Host, opts ...Option) (*PubSub, error) {
	rt := &RandomSubRouter{
		peers: make(map[peer.ID]protocol.ID),
	}
	return NewPubSub(ctx, h, rt, opts...)
}

// RandomSubRouter is a router that implements a random propagation strategy.
// For each message, it selects RandomSubD peers and forwards the message to them.
type RandomSubRouter struct {
	p     *PubSub
	peers map[peer.ID]protocol.ID
}

func (rs *RandomSubRouter) Protocols() []protocol.ID {
	return []protocol.ID{RandomSubID, FloodSubID}
}

func (rs *RandomSubRouter) Attach(p *PubSub) {
	rs.p = p
}

func (rs *RandomSubRouter) AddPeer(p peer.ID, proto protocol.ID) {
	rs.peers[p] = proto
}

func (rs *RandomSubRouter) RemovePeer(p peer.ID) {
	delete(rs.peers, p)
}

func (rs *RandomSubRouter) HandleRPC(rpc *RPC) {}

func (rs *RandomSubRouter) Publish(from peer.ID, msg *pb.Message) {
	tosend := make(map[peer.ID]struct{})
	rspeers := make(map[peer.ID]struct{})
	src := peer.ID(msg.GetFrom())

	for _, topic := range msg.GetTopicIDs() {
		tmap, ok := rs.p.topics[topic]
		if !ok {
			continue
		}

		for p := range tmap {
			if p == from || p == src {
				continue
			}

			if rs.peers[p] == FloodSubID {
				tosend[p] = struct{}{}
			} else {
				rspeers[p] = struct{}{}
			}
		}
	}

	if len(rspeers) > RandomSubD {
		xpeers := peerMapToList(rspeers)
		shufflePeers(xpeers)
		xpeers = xpeers[:RandomSubD]
		for _, p := range xpeers {
			tosend[p] = struct{}{}
		}
	} else {
		for p := range rspeers {
			tosend[p] = struct{}{}
		}
	}

	out := rpcWithMessages(msg)
	for p := range tosend {
		mch, ok := rs.p.peers[p]
		if !ok {
			continue
		}

		select {
		case mch <- out:
		default:
			log.Infof("dropping message to peer %s: queue full", p)
		}
	}
}

func (rs *RandomSubRouter) Join(topic string) {}

func (rs *RandomSubRouter) Leave(topic string) {}
