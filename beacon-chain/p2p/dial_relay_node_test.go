package p2p

import (
	"context"
	"fmt"
	"testing"

	bh "github.com/libp2p/go-libp2p-blankhost"
	swarmt "github.com/libp2p/go-libp2p-swarm/testing"
)

func TestMakePeer_InvalidMultiaddress(t *testing.T) {
	_, err := MakePeer("/ip4")
	if err == nil {
		t.Error("Expect error when invalid multiaddress was provided")
	}
}

func TestMakePeer_OK(t *testing.T) {
	a, err := MakePeer("/ip4/127.0.0.1/tcp/5678/p2p/QmUn6ycS8Fu6L462uZvuEfDoSgYX6kqP4aSZWMa7z1tWAX")
	if err != nil {
		t.Fatalf("Unexpected error when making a valid peer: %v", err)
	}

	if a.ID.Pretty() != "QmUn6ycS8Fu6L462uZvuEfDoSgYX6kqP4aSZWMa7z1tWAX" {
		t.Errorf("Unexpected peer ID %v", a.ID.Pretty())
	}
}

func TestDialRelayNode_InvalidPeerString(t *testing.T) {
	if err := dialRelayNode(context.Background(), nil, "/ip4"); err == nil {
		t.Fatal("Expected to fail with invalid peer string, but there was no error")
	}
}

func TestDialRelayNode_OK(t *testing.T) {
	ctx := context.Background()
	relay := bh.NewBlankHost(swarmt.GenSwarm(t, ctx))
	host := bh.NewBlankHost(swarmt.GenSwarm(t, ctx))

	relayAddr := fmt.Sprintf("%s/p2p/%s", relay.Addrs()[0], relay.ID().Pretty())

	if err := dialRelayNode(ctx, host, relayAddr); err != nil {
		t.Errorf("Unexpected error when dialing relay node %v", err)
	}

	if host.Peerstore().PeerInfo(relay.ID()).ID != relay.ID() {
		t.Error("Host peerstore does not have peer info on relay node")
	}
}
