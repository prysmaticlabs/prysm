package p2p

import (
	"context"

	"github.com/libp2p/go-libp2p-core/host"
	peerstore "github.com/libp2p/go-libp2p-peerstore"
	"go.opencensus.io/trace"
)

// MakePeer from multiaddress string.
func MakePeer(addr string) (*peerstore.PeerInfo, error) {
	maddr, err := multiAddrFromString(addr)
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
	ctx, cancel := context.WithTimeout(ctx, maxDialTimeout)
	defer cancel()
	return h.Connect(ctx, *p)
}
