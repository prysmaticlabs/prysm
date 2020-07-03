package sync

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/protocol"
	db "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	p2ptest "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestPingRPCHandler_ReceivesPing(t *testing.T) {
	p1 := p2ptest.NewTestP2P(t)
	p2 := p2ptest.NewTestP2P(t)
	p1.Connect(p2)
	if len(p1.BHost.Network().Peers()) != 1 {
		t.Error("Expected peers to be connected")
	}
	p1.LocalMetadata = &pb.MetaData{
		SeqNumber: 2,
		Attnets:   []byte{'A', 'B'},
	}

	p2.LocalMetadata = &pb.MetaData{
		SeqNumber: 2,
		Attnets:   []byte{'C', 'D'},
	}

	// Set up a head state in the database with data we expect.
	d, _ := db.SetupDB(t)
	r := &Service{
		db:  d,
		p2p: p1,
	}

	p1.Peers().Add(new(enr.Record), p2.BHost.ID(), p2.BHost.Addrs()[0], network.DirUnknown)
	p1.Peers().SetMetadata(p2.BHost.ID(), p2.LocalMetadata)

	// Setup streams
	pcl := protocol.ID("/testing")
	var wg sync.WaitGroup
	wg.Add(1)
	p2.BHost.SetStreamHandler(pcl, func(stream network.Stream) {
		defer wg.Done()
		expectSuccess(t, r, stream)
		out := new(uint64)
		if err := r.p2p.Encoding().DecodeWithMaxLength(stream, out); err != nil {
			t.Fatal(err)
		}
		if *out != 2 {
			t.Fatalf("Wanted 2 but got %d as our sequence number", *out)
		}
	})
	stream1, err := p1.BHost.NewStream(context.Background(), p2.BHost.ID(), pcl)
	if err != nil {
		t.Fatal(err)
	}
	seqNumber := uint64(1)

	err = r.pingHandler(context.Background(), &seqNumber, stream1)
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

func TestPingRPCHandler_SendsPing(t *testing.T) {
	p1 := p2ptest.NewTestP2P(t)
	p2 := p2ptest.NewTestP2P(t)
	p1.Connect(p2)
	if len(p1.BHost.Network().Peers()) != 1 {
		t.Error("Expected peers to be connected")
	}
	p1.LocalMetadata = &pb.MetaData{
		SeqNumber: 2,
		Attnets:   []byte{'A', 'B'},
	}

	p2.LocalMetadata = &pb.MetaData{
		SeqNumber: 2,
		Attnets:   []byte{'C', 'D'},
	}

	// Set up a head state in the database with data we expect.
	d, _ := db.SetupDB(t)
	r := &Service{
		db:  d,
		p2p: p1,
	}

	p1.Peers().Add(new(enr.Record), p2.BHost.ID(), p2.BHost.Addrs()[0], network.DirUnknown)
	p1.Peers().SetMetadata(p2.BHost.ID(), p2.LocalMetadata)

	p2.Peers().Add(new(enr.Record), p1.BHost.ID(), p1.BHost.Addrs()[0], network.DirUnknown)
	p2.Peers().SetMetadata(p1.BHost.ID(), p1.LocalMetadata)

	r2 := &Service{
		db:  d,
		p2p: p2,
	}
	// Setup streams
	pcl := protocol.ID("/eth2/beacon_chain/req/ping/1/ssz_snappy")
	var wg sync.WaitGroup
	wg.Add(1)
	p2.BHost.SetStreamHandler(pcl, func(stream network.Stream) {
		defer wg.Done()
		out := new(uint64)
		if err := r2.p2p.Encoding().DecodeWithMaxLength(stream, out); err != nil {
			t.Fatal(err)
		}
		if *out != 2 {
			t.Fatalf("Wanted 2 but got %d as our sequence number", *out)
		}
		err := r2.pingHandler(context.Background(), out, stream)
		if err != nil {
			t.Fatal(err)
		}
	})

	err := r.sendPingRequest(context.Background(), p2.BHost.ID())
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
