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
)

func TestMetaDataRPCHandler_ReceivesMetadata(t *testing.T) {
	p1 := p2ptest.NewTestP2P(t)
	p2 := p2ptest.NewTestP2P(t)
	p1.Connect(p2)
	if len(p1.BHost.Network().Peers()) != 1 {
		t.Error("Expected peers to be connected")
	}
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
		if err := r.p2p.Encoding().DecodeWithLength(stream, out); err != nil {
			t.Fatal(err)
		}
		if !ssz.DeepEqual(p1.LocalMetadata, out) {
			t.Fatalf("Metadata unequal, received %v but wanted %v", out, p1.LocalMetadata)
		}
	})
	stream1, err := p1.BHost.NewStream(context.Background(), p2.BHost.ID(), pcl)
	if err != nil {
		t.Fatal(err)
	}

	err = r.metaDataHandler(context.Background(), new(interface{}), stream1)
	if err != nil {
		t.Errorf("Unxpected error: %v", err)
	}

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
	if len(p1.BHost.Network().Peers()) != 1 {
		t.Error("Expected peers to be connected")
	}
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

		err := r2.metaDataHandler(context.Background(), new(interface{}), stream)
		if err != nil {
			t.Fatal(err)
		}
	})

	metadata, err := r.sendMetaDataRequest(context.Background(), p2.BHost.ID())
	if err != nil {
		t.Errorf("Unxpected error: %v", err)
	}

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
