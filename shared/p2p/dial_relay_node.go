package p2p

import (
	"context"

	host "github.com/libp2p/go-libp2p-host"
	peerstore "github.com/libp2p/go-libp2p-peerstore"
	"github.com/multiformats/go-multiaddr"
	"go.opencensus.io/trace"
)

// MakePeer from multiaddress string.
func MakePeer(addr string) (*peerstore.PeerInfo, error) {
	maddr, err := multiaddr.NewMultiaddr(addr)
	if err != nil {
		return nil, err
	}
	return peerstore.InfoFromP2pAddr(maddr)
}

func dialRelayNode(ctx context.Context, h host.Host, relayAddr string) error {
	ctx, span := trace.StartSpan(ctx, "p2p_dialRelayNode")
	defer span.End()

	p, err := MakePeer(relayAddr)
	if err != nil {
		return err
	}

	return h.Connect(ctx, *p)
}
