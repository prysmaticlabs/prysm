package p2p

import (
	"context"
	"testing"
	"time"

	swarmt "github.com/libp2p/go-libp2p-swarm/testing"
	mdns "github.com/libp2p/go-libp2p/p2p/discovery"
	bhost "github.com/libp2p/go-libp2p/p2p/host/basic"
)

var _ = mdns.Notifee(&discovery{})

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

func TestStartDiscovery_PeerFound(t *testing.T) {
	discoveryInterval = 50 * time.Millisecond // Short interval for testing.

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	a := bhost.New(swarmt.GenSwarm(t, ctx))
	err := startmDNSDiscovery(ctx, a)
	if err != nil {
		t.Errorf("Error when starting discovery: %v", err)
	}

	b := bhost.New(swarmt.GenSwarm(t, ctx))
	err = startmDNSDiscovery(ctx, b)
	if err != nil {
		t.Errorf("Error when starting discovery: %v", err)
	}

	// The two hosts should have found each other after 1+ intervals.
	time.Sleep(2 * discoveryInterval)

	expectPeers(t, a, 2)
	expectPeers(t, b, 2)
}
