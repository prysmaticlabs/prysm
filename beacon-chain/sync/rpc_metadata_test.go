package sync

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/prysmaticlabs/go-ssz"
	db "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	p2ptest "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestMetaDataRPCHandler_ReceivesMetadata(t *testing.T) {
	p1 := p2ptest.NewTestP2P(t)
	p2 := p2ptest.NewTestP2P(t)
	p1.Connect(p2)
	assert.Equal(t, 1, len(p1.BHost.Network().Peers()), "Expected peers to be connected")
	bitfield := [8]byte{'A', 'B'}
	p1.LocalMetadata = &pb.MetaData{
		SeqNumber: 2,
		Attnets:   bitfield[:],
	}

	// Set up a head state in the database with data we expect.
	d, _ := db.SetupDB(t)
	r := &Service{
		db:  d,
		p2p: p1,
	}

	// Setup streams
	pcl := protocol.ID("/testing")
	var wg sync.WaitGroup
	wg.Add(1)
	p2.BHost.SetStreamHandler(pcl, func(stream network.Stream) {
		defer wg.Done()
		expectSuccess(t, r, stream)
		out := new(pb.MetaData)
		require.NoError(t, r.p2p.Encoding().DecodeWithMaxLength(stream, out))
		require.DeepEqual(t, p1.LocalMetadata, out, "Metadata unequal")
	})
	stream1, err := p1.BHost.NewStream(context.Background(), p2.BHost.ID(), pcl)
	require.NoError(t, err)

	assert.NoError(t, r.metaDataHandler(context.Background(), new(interface{}), stream1))

	if testutil.WaitTimeout(&wg, 1*time.Second) {
		t.Fatal("Did not receive stream within 1 sec")
	}

	conns := p1.BHost.Network().ConnsToPeer(p2.BHost.ID())
	if len(conns) == 0 {
		t.Error("Peer is disconnected despite receiving a valid ping")
	}
}

func TestMetadataRPCHandler_SendsMetadata(t *testing.T) {
	p1 := p2ptest.NewTestP2P(t)
	p2 := p2ptest.NewTestP2P(t)
	p1.Connect(p2)
	assert.Equal(t, 1, len(p1.BHost.Network().Peers()), "Expected peers to be connected")
	bitfield := [8]byte{'A', 'B'}
	p2.LocalMetadata = &pb.MetaData{
		SeqNumber: 2,
		Attnets:   bitfield[:],
	}

	// Set up a head state in the database with data we expect.
	d, _ := db.SetupDB(t)
	r := &Service{
		db:  d,
		p2p: p1,
	}

	r2 := &Service{
		db:  d,
		p2p: p2,
	}

	// Setup streams
	pcl := protocol.ID(p2p.RPCMetaDataTopic + r.p2p.Encoding().ProtocolSuffix())
	var wg sync.WaitGroup
	wg.Add(1)
	p2.BHost.SetStreamHandler(pcl, func(stream network.Stream) {
		defer wg.Done()
		require.NoError(t, r2.metaDataHandler(context.Background(), new(interface{}), stream))
	})

	metadata, err := r.sendMetaDataRequest(context.Background(), p2.BHost.ID())
	assert.NoError(t, err)

	if !ssz.DeepEqual(metadata, p2.LocalMetadata) {
		t.Fatalf("Metadata unequal, received %v but wanted %v", metadata, p2.LocalMetadata)
	}

	if testutil.WaitTimeout(&wg, 1*time.Second) {
		t.Fatal("Did not receive stream within 1 sec")
	}

	conns := p1.BHost.Network().ConnsToPeer(p2.BHost.ID())
	if len(conns) == 0 {
		t.Error("Peer is disconnected despite receiving a valid ping")
	}
}
