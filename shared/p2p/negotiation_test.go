package p2p

import (
	"context"
	"testing"
	"time"

	bhost "github.com/libp2p/go-libp2p-blankhost"
	libp2pnet "github.com/libp2p/go-libp2p-net"
	peer "github.com/libp2p/go-libp2p-peer"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	swarmt "github.com/libp2p/go-libp2p-swarm/testing"
)

func TestNegotiation_AcceptsValidPeer(t *testing.T) {
	ctx := context.Background()
	hostA := bhost.NewBlankHost(swarmt.GenSwarm(t, ctx))
	hostB := bhost.NewBlankHost(swarmt.GenSwarm(t, ctx))

	address := "same"
	setHandshakeHandler(hostA, address)
	setHandshakeHandler(hostB, address)

	setupPeerNegotiation(hostA, address, []peer.ID{})
	setupPeerNegotiation(hostB, address, []peer.ID{})

	if err := hostA.Connect(ctx, pstore.PeerInfo{ID: hostB.ID(), Addrs: hostB.Addrs()}); err != nil {
		t.Fatal(err)
	}

	// Allow short delay for async negotiation.
	time.Sleep(200 * time.Millisecond)
	if hostA.Network().Connectedness(hostB.ID()) != libp2pnet.Connected {
		t.Error("hosts are not connected")
	}
}

func TestNegotiation_DisconnectsDifferentDepositContract(t *testing.T) {
	ctx := context.Background()
	hostA := bhost.NewBlankHost(swarmt.GenSwarm(t, ctx))
	hostB := bhost.NewBlankHost(swarmt.GenSwarm(t, ctx))

	setHandshakeHandler(hostA, "hostA")
	setHandshakeHandler(hostB, "hostB")

	setupPeerNegotiation(hostA, "hostA", []peer.ID{})
	setupPeerNegotiation(hostB, "hostB", []peer.ID{})

	if err := hostA.Connect(ctx, pstore.PeerInfo{ID: hostB.ID(), Addrs: hostB.Addrs()}); err != nil {
		t.Fatal(err)
	}

	// Allow short delay for async negotiation.
	time.Sleep(200 * time.Millisecond)
	if hostA.Network().Connectedness(hostB.ID()) == libp2pnet.Connected {
		t.Error("hosts are connected, but should not be connected")
	}
}
