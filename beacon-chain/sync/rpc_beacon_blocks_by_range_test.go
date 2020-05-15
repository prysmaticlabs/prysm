package sync

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/kevinms/leakybucket-go"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/protocol"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	db "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/flags"
	p2ptest "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestRPCBeaconBlocksByRange_RPCHandlerReturnsBlocks(t *testing.T) {
	p1 := p2ptest.NewTestP2P(t)
	p2 := p2ptest.NewTestP2P(t)
	p1.Connect(p2)
	if len(p1.Host.Network().Peers()) != 1 {
		t.Error("Expected peers to be connected")
	}
	d := db.SetupDB(t)

	req := &pb.BeaconBlocksByRangeRequest{
		StartSlot: 100,
		Step:      64,
		Count:     16,
	}

	// Populate the database with blocks that would match the request.
	for i := req.StartSlot; i < req.StartSlot+(req.Step*req.Count); i += req.Step {
		if err := d.SaveBlock(context.Background(), &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: i}}); err != nil {
			t.Fatal(err)
		}
	}

	// Start service with 160 as allowed blocks capacity (and almost zero capacity recovery).
	r := &Service{p2p: p1, db: d, blocksRateLimiter: leakybucket.NewCollector(0.000001, int64(req.Count*10), false)}
	pcl := protocol.ID("/testing")

	var wg sync.WaitGroup
	wg.Add(1)
	p2.Host.SetStreamHandler(pcl, func(stream network.Stream) {
		defer wg.Done()
		for i := req.StartSlot; i < req.Count*req.Step; i += req.Step {
			expectSuccess(t, r, stream)
			res := &ethpb.SignedBeaconBlock{}
			if err := r.p2p.Encoding().DecodeWithLength(stream, res); err != nil {
				t.Error(err)
			}
			if (res.Block.Slot-req.StartSlot)%req.Step != 0 {
				t.Errorf("Received unexpected block slot %d", res.Block.Slot)
			}
		}
	})

	stream1, err := p1.Host.NewStream(context.Background(), p2.Host.ID(), pcl)
	if err != nil {
		t.Fatal(err)
	}

	err = r.beaconBlocksByRangeRPCHandler(context.Background(), req, stream1)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Make sure that rate limiter doesn't limit capacity exceedingly.
	remainingCapacity := r.blocksRateLimiter.Remaining(p2.PeerID().String())
	expectedCapacity := int64(req.Count*10 - req.Count)
	if remainingCapacity != expectedCapacity {
		t.Fatalf("Unexpected rate limiting capacity, expected: %v, got: %v", expectedCapacity, remainingCapacity)
	}

	if testutil.WaitTimeout(&wg, 1*time.Second) {
		t.Fatal("Did not receive stream within 1 sec")
	}
}

func TestRPCBeaconBlocksByRange_ReturnsGenesisBlock(t *testing.T) {
	p1 := p2ptest.NewTestP2P(t)
	p2 := p2ptest.NewTestP2P(t)
	p1.Connect(p2)
	if len(p1.Host.Network().Peers()) != 1 {
		t.Error("Expected peers to be connected")
	}
	d := db.SetupDB(t)

	req := &pb.BeaconBlocksByRangeRequest{
		StartSlot: 0,
		Step:      1,
		Count:     4,
	}

	// Populate the database with blocks that would match the request.
	for i := req.StartSlot; i < req.StartSlot+(req.Step*req.Count); i++ {
		// Save genesis block
		if i == 0 {
			rt, err := stateutil.BlockRoot(&ethpb.BeaconBlock{Slot: i})
			if err != nil {
				t.Fatal(err)
			}
			if err := d.SaveGenesisBlockRoot(context.Background(), rt); err != nil {
				t.Fatal(err)
			}
		}
		if err := d.SaveBlock(context.Background(), &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: i}}); err != nil {
			t.Fatal(err)
		}
	}

	r := &Service{p2p: p1, db: d, blocksRateLimiter: leakybucket.NewCollector(10000, 10000, false)}
	pcl := protocol.ID("/testing")

	var wg sync.WaitGroup
	wg.Add(1)
	p2.Host.SetStreamHandler(pcl, func(stream network.Stream) {
		defer wg.Done()
		// check for genesis block
		expectSuccess(t, r, stream)
		res := &ethpb.SignedBeaconBlock{}
		if err := r.p2p.Encoding().DecodeWithLength(stream, res); err != nil {
			t.Error(err)
		}
		if res.Block.Slot != 0 {
			t.Fatal("genesis block was not returned")
		}
		for i := req.StartSlot + req.Step; i < req.Count*req.Step; i += req.Step {
			expectSuccess(t, r, stream)
			res := &ethpb.SignedBeaconBlock{}
			if err := r.p2p.Encoding().DecodeWithLength(stream, res); err != nil {
				t.Error(err)
			}
		}
	})

	stream1, err := p1.Host.NewStream(context.Background(), p2.Host.ID(), pcl)
	if err != nil {
		t.Fatal(err)
	}

	err = r.beaconBlocksByRangeRPCHandler(context.Background(), req, stream1)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if testutil.WaitTimeout(&wg, 1*time.Second) {
		t.Fatal("Did not receive stream within 1 sec")
	}
}

func TestRPCBeaconBlocksByRange_RPCHandlerRateLimitOverflow(t *testing.T) {
	d := db.SetupDB(t)

	saveBlocks := func(req *pb.BeaconBlocksByRangeRequest) {
		// Populate the database with blocks that would match the request.
		for i := req.StartSlot; i < req.StartSlot+(req.Step*req.Count); i += req.Step {
			if err := d.SaveBlock(context.Background(), &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: i}}); err != nil {
				t.Fatal(err)
			}
		}
	}

	streamHandlerGenerator := func(r *Service, wg *sync.WaitGroup, req *pb.BeaconBlocksByRangeRequest) func(stream network.Stream) {
		return func(stream network.Stream) {
			defer wg.Done()
			for i := req.StartSlot; i < req.Count*req.Step; i += req.Step {
				expectSuccess(t, r, stream)
				res := &ethpb.SignedBeaconBlock{}
				if err := r.p2p.Encoding().DecodeWithLength(stream, res); err != nil {
					t.Error(err)
				}
				if (res.Block.Slot-req.StartSlot)%req.Step != 0 {
					t.Errorf("Received unexpected block slot %d", res.Block.Slot)
				}
			}
		}
	}

	t.Run("big count param, no overflow", func(t *testing.T) {
		p1 := p2ptest.NewTestP2P(t)
		p2 := p2ptest.NewTestP2P(t)
		p1.Connect(p2)
		if len(p1.Host.Network().Peers()) != 1 {
			t.Error("Expected peers to be connected")
		}

		capacity := int64(flags.Get().BlockBatchLimit * 3)
		req := &pb.BeaconBlocksByRangeRequest{
			StartSlot: 100,
			Step:      1,
			Count:     uint64(capacity),
		}

		saveBlocks(req)
		r := &Service{p2p: p1, db: d, blocksRateLimiter: leakybucket.NewCollector(0.000001, capacity, false)}
		pcl := protocol.ID("/testing")

		var wg sync.WaitGroup
		wg.Add(1)
		p2.Host.SetStreamHandler(pcl, streamHandlerGenerator(r, &wg, req))

		stream1, err := p1.Host.NewStream(context.Background(), p2.Host.ID(), pcl)
		if err != nil {
			t.Fatal(err)
		}

		err = r.beaconBlocksByRangeRPCHandler(context.Background(), req, stream1)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		remainingCapacity := r.blocksRateLimiter.Remaining(p2.PeerID().String())
		expectedCapacity := int64(0) // Whole capacity is used, but no overflow.
		if remainingCapacity != expectedCapacity {
			t.Fatalf("Unexpected rate limiting capacity, expected: %v, got: %v", expectedCapacity, remainingCapacity)
		}

		if testutil.WaitTimeout(&wg, 1*time.Second) {
			t.Fatal("Did not receive stream within 1 sec")
		}
	})

	t.Run("big count param, overflow", func(t *testing.T) {
		p1 := p2ptest.NewTestP2P(t)
		p2 := p2ptest.NewTestP2P(t)
		p1.Connect(p2)
		if len(p1.Host.Network().Peers()) != 1 {
			t.Error("Expected peers to be connected")
		}

		capacity := int64(flags.Get().BlockBatchLimit * 3)
		req := &pb.BeaconBlocksByRangeRequest{
			StartSlot: 100,
			Step:      1,
			Count:     uint64(capacity + 1),
		}
		saveBlocks(req)

		r := &Service{p2p: p1, db: d, blocksRateLimiter: leakybucket.NewCollector(0.000001, capacity, false)}
		pcl := protocol.ID("/testing")

		var wg sync.WaitGroup
		wg.Add(1)
		p2.Host.SetStreamHandler(pcl, streamHandlerGenerator(r, &wg, req))

		stream1, err := p1.Host.NewStream(context.Background(), p2.Host.ID(), pcl)
		if err != nil {
			t.Fatal(err)
		}

		err = r.beaconBlocksByRangeRPCHandler(context.Background(), req, stream1)
		if err == nil || err.Error() != rateLimitedError {
			t.Errorf("Expected error not thrown, want: %v, got: %v", rateLimitedError, err)
		}

		remainingCapacity := r.blocksRateLimiter.Remaining(p2.PeerID().String())
		expectedCapacity := int64(0) // Whole capacity is used.
		if remainingCapacity != expectedCapacity {
			t.Fatalf("Unexpected rate limiting capacity, expected: %v, got: %v", expectedCapacity, remainingCapacity)
		}

		if testutil.WaitTimeout(&wg, 1*time.Second) {
			t.Fatal("Did not receive stream within 1 sec")
		}
	})
}
