package p2p

import (
	"context"
	"testing"

	bhost "github.com/libp2p/go-libp2p-blankhost"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	swarmt "github.com/libp2p/go-libp2p-swarm/testing"
)

func TestEnsurePeerConnections_reconnectsToPeer(t *testing.T) {
	ctx := context.Background()
	h := bhost.NewBlankHost(swarmt.GenSwarm(t, ctx))
	vipPeer := bhost.NewBlankHost(swarmt.GenSwarm(t, ctx))

	vipMAddrs, err := pstore.InfoToP2pAddrs(&pstore.PeerInfo{ID: vipPeer.ID(), Addrs: vipPeer.Addrs()})
	if err != nil {
		t.Fatal(err)
	}

	if len(h.Peerstore().Peers()) != 1 {
		t.Fatal("expected 1 peer")
	}

	ensurePeerConnections(ctx, h, vipMAddrs[0].String())

	if len(h.Peerstore().Peers()) != 2 {
		t.Fatal("expected 2 peers")
	}
}
