package p2p

import (
	"context"
	"testing"
	"time"

	floodsub "github.com/libp2p/go-floodsub"
	peer "github.com/libp2p/go-libp2p-peer"
	swarmt "github.com/libp2p/go-libp2p-swarm/testing"
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

func TestStartDiscovery_HandlePeerFound(t *testing.T) {
	discoveryInterval = 50 * time.Millisecond // Short interval for testing.

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	gsub := &fakeTopicPeerLister{}

	a := bhost.New(swarmt.GenSwarm(t, ctx))
	err := startDiscovery(ctx, a, gsub)
	if err != nil {
		t.Errorf("Error when starting discovery: %v", err)
	}

	b := bhost.New(swarmt.GenSwarm(t, ctx))
	err = startDiscovery(ctx, b, gsub)
	if err != nil {
		t.Errorf("Error when starting discovery: %v", err)
	}

	// The two hosts should have found each other after 1+ intervals.
	time.Sleep(2 * discoveryInterval)

	expectPeers(t, a, 2)
	expectPeers(t, b, 2)
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
