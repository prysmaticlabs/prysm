package dhtutil_test

import (
	"context"
	"testing"

	ggio "github.com/gogo/protobuf/io"
	"github.com/libp2p/go-libp2p-blankhost"
	dhtopts "github.com/libp2p/go-libp2p-kad-dht/opts"
	dhtpb "github.com/libp2p/go-libp2p-kad-dht/pb"
	inet "github.com/libp2p/go-libp2p-net"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	swarmt "github.com/libp2p/go-libp2p-swarm/testing"
	"github.com/prysmaticlabs/prysm/shared/p2p/dhtutil"
)

var testPeers = []pstore.PeerInfo{
	// TODO: Add a test peer
	pstore.PeerInfo{},
}

func makeFakeDHT(t *testing.T, ctx context.Context) pstore.PeerInfo {
	h := blankhost.NewBlankHost(swarmt.GenSwarm(t, ctx))
	h.SetStreamHandler(dhtopts.ProtocolDHT, func(s inet.Stream) {
		w := ggio.NewFullWriter(s)
		defer w.Close()

		msg := dhtpb.NewMessage(dhtpb.Message_FIND_NODE, []byte{}, 0)
		msg.CloserPeers = dhtpb.RawPeerInfosToPBPeers(testPeers)

		w.WriteMsg(msg)
	})

	return pstore.PeerInfo{
		ID:    h.ID(),
		Addrs: h.Addrs(),
	}
}

func TestPeerInfoFromDHT(t *testing.T) {
	ctx := context.Background()
	dhtPeerInfo := makeFakeDHT(t, ctx)
	h := blankhost.NewBlankHost(swarmt.GenSwarm(t, ctx))
	if err := h.Connect(ctx, dhtPeerInfo); err != nil {
		t.Fatalf("Failed to connect to fake dht: %v", err)
	}

	peerInfos, err := dhtutil.PeerInfoFromDHT(ctx, h, dhtPeerInfo.ID)
	if err != nil {
		t.Fatalf("Failed to get peer info from DHT: %v", err)
	}

	// TODO: Verify the test peer came back from DHT
	_ = peerInfos
}
