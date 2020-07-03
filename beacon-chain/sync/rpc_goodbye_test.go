package sync

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/protocol"
	db "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	p2ptest "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestGoodByeRPCHandler_Disconnects_With_Peer(t *testing.T) {
	p1 := p2ptest.NewTestP2P(t)
	p2 := p2ptest.NewTestP2P(t)
	p1.Connect(p2)
	if len(p1.BHost.Network().Peers()) != 1 {
		t.Error("Expected peers to be connected")
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
		expectResetStream(t, r, stream)
	})
	stream1, err := p1.BHost.NewStream(context.Background(), p2.BHost.ID(), pcl)
	if err != nil {
		t.Fatal(err)
	}
	failureCode := codeClientShutdown

	err = r.goodbyeRPCHandler(context.Background(), &failureCode, stream1)
	if err != nil {
		t.Errorf("Unxpected error: %v", err)
	}

	if testutil.WaitTimeout(&wg, 1*time.Second) {
		t.Fatal("Did not receive stream within 1 sec")
	}

	conns := p1.BHost.Network().ConnsToPeer(p1.BHost.ID())
	if len(conns) > 0 {
		t.Error("Peer is still not disconnected despite sending a goodbye message")
	}
}

func TestSendGoodbye_SendsMessage(t *testing.T) {
	p1 := p2ptest.NewTestP2P(t)
	p2 := p2ptest.NewTestP2P(t)
	p1.Connect(p2)
	if len(p1.BHost.Network().Peers()) != 1 {
		t.Error("Expected peers to be connected")
	}

	// Set up a head state in the database with data we expect.
	d, _ := db.SetupDB(t)
	r := &Service{
		db:  d,
		p2p: p1,
	}
	failureCode := codeClientShutdown

	// Setup streams
	pcl := protocol.ID("/eth2/beacon_chain/req/goodbye/1/ssz_snappy")
	var wg sync.WaitGroup
	wg.Add(1)
	p2.BHost.SetStreamHandler(pcl, func(stream network.Stream) {
		defer wg.Done()
		out := new(uint64)
		if err := r.p2p.Encoding().DecodeWithMaxLength(stream, out); err != nil {
			t.Fatal(err)
		}
		if *out != failureCode {
			t.Fatalf("Wanted goodbye code of %d but got %d", failureCode, *out)
		}

	})

	err := r.sendGoodByeMessage(context.Background(), failureCode, p2.BHost.ID())
	if err != nil {
		t.Errorf("Unxpected error: %v", err)
	}

	if testutil.WaitTimeout(&wg, 1*time.Second) {
		t.Fatal("Did not receive stream within 1 sec")
	}

	conns := p1.BHost.Network().ConnsToPeer(p1.BHost.ID())
	if len(conns) > 0 {
		t.Error("Peer is still not disconnected despite sending a goodbye message")
	}
}

func TestSendGoodbye_DisconnectWithPeer(t *testing.T) {
	p1 := p2ptest.NewTestP2P(t)
	p2 := p2ptest.NewTestP2P(t)
	p1.Connect(p2)
	if len(p1.BHost.Network().Peers()) != 1 {
		t.Error("Expected peers to be connected")
	}

	// Set up a head state in the database with data we expect.
	d, _ := db.SetupDB(t)
	r := &Service{
		db:  d,
		p2p: p1,
	}
	failureCode := codeClientShutdown

	// Setup streams
	pcl := protocol.ID("/eth2/beacon_chain/req/goodbye/1/ssz_snappy")
	var wg sync.WaitGroup
	wg.Add(1)
	p2.BHost.SetStreamHandler(pcl, func(stream network.Stream) {
		defer wg.Done()
		out := new(uint64)
		if err := r.p2p.Encoding().DecodeWithMaxLength(stream, out); err != nil {
			t.Fatal(err)
		}
		if *out != failureCode {
			t.Fatalf("Wanted goodbye code of %d but got %d", failureCode, *out)
		}

	})

	err := r.sendGoodByeAndDisconnect(context.Background(), failureCode, p2.BHost.ID())
	if err != nil {
		t.Errorf("Unxpected error: %v", err)
	}
	conns := p1.BHost.Network().ConnsToPeer(p2.BHost.ID())
	if len(conns) > 0 {
		t.Error("Peer is still not disconnected despite sending a goodbye message")
	}

	if testutil.WaitTimeout(&wg, 1*time.Second) {
		t.Fatal("Did not receive stream within 1 sec")
	}

}
