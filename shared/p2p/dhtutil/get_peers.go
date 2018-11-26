// dhtutil provides utility functions for DHT/bootnode operations
package dhtutil

import (
	"context"

	ggio "github.com/gogo/protobuf/io"
	"github.com/libp2p/go-libp2p-host"
	dhtopts "github.com/libp2p/go-libp2p-kad-dht/opts"
	dhtpb "github.com/libp2p/go-libp2p-kad-dht/pb"
	peer "github.com/libp2p/go-libp2p-peer"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	"go.opencensus.io/trace"
)

var MAX_SIZE = 2000000

// PeerInfoFromDHT queries the DHT peer for a list of available peers.
func PeerInfoFromDHT(ctx context.Context, h host.Host, dhtPid peer.ID) ([]*pstore.PeerInfo, error) {
	ctx, span := trace.StartSpan(ctx, "dhtutil.PeerInfoFromDHT")
	defer span.End()

	var peers []*pstore.PeerInfo

	s, err := h.NewStream(ctx, dhtPid, dhtopts.ProtocolDHT)
	if err != nil {
		return nil, err
	}

	r := ggio.NewDelimitedReader(s, MAX_SIZE)
	w := ggio.NewDelimitedWriter(s)

	if err := w.WriteMsg(dhtpb.NewMessage(dhtpb.Message_FIND_NODE, []byte{}, 0)); err != nil {
		return nil, err
	}

	// wait for message
	var resp dhtpb.Message
	if err := r.ReadMsg(&resp); err != nil {
		return nil, err
	}

	for _, pbp := range resp.GetCloserPeers() {
		peers = append(peers, dhtpb.PBPeerToPeerInfo(pbp))
	}

	return peers, nil
}
