package p2p

import (
	"testing"

	floodsub "github.com/libp2p/go-floodsub"
	peer "github.com/libp2p/go-libp2p-peer"
	mdns "github.com/libp2p/go-libp2p/p2p/discovery"
	bhost "github.com/libp2p/go-libp2p/p2p/host/basic"
)

var _ = mdns.Notifee(&discovery{})
var _ = topicPeerLister(&floodsub.PubSub{})

var _ = topicPeerLister(&fakeTopicPeerLister{})

type fakeTopicPeerLister struct {
}

func (f *fakeTopicPeerLister) ListPeers(topic string) []peer.ID {
	return nil
}

func expectPeers(t *testing.T, h *bhost.BasicHost, n int) {
	if len(h.Peerstore().Peers()) != n {
		t.Errorf(
			"Expected %d peer for host %v, but has %d peers. They are: %v.",
			n,
			h.ID(),
			len(h.Peerstore().Peers()),
			h.Peerstore().Peers(),
		)
	}
}
