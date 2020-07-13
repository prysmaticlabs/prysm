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
	chainMock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	db "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/flags"
	p2ptest "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestRPCBeaconBlocksByRange_RPCHandlerReturnsBlocks(t *testing.T) {
	p1 := p2ptest.NewTestP2P(t)
	p2 := p2ptest.NewTestP2P(t)
	p1.Connect(p2)
	if len(p1.BHost.Network().Peers()) != 1 {
		t.Error("Expected peers to be connected")
	}
	d, _ := db.SetupDB(t)

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
	r := &Service{p2p: p1, db: d, blocksRateLimiter: leakybucket.NewCollector(0.000001, int64(req.Count*10), false),
		chain: &chainMock.ChainService{}}
	pcl := protocol.ID("/testing")

	var wg sync.WaitGroup
	wg.Add(1)
	p2.BHost.SetStreamHandler(pcl, func(stream network.Stream) {
		defer wg.Done()
		for i := req.StartSlot; i < req.StartSlot+req.Count*req.Step; i += req.Step {
			expectSuccess(t, r, stream)
			res := &ethpb.SignedBeaconBlock{}
			if err := r.p2p.Encoding().DecodeWithMaxLength(stream, res); err != nil {
				t.Error(err)
			}
			if (res.Block.Slot-req.StartSlot)%req.Step != 0 {
				t.Errorf("Received unexpected block slot %d", res.Block.Slot)
			}
		}
	})

	stream1, err := p1.BHost.NewStream(context.Background(), p2.BHost.ID(), pcl)
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

func TestRPCBeaconBlocksByRange_RPCHandlerReturnsSortedBlocks(t *testing.T) {
	p1 := p2ptest.NewTestP2P(t)
	p2 := p2ptest.NewTestP2P(t)
	p1.Connect(p2)
	if len(p1.BHost.Network().Peers()) != 1 {
		t.Error("Expected peers to be connected")
	}
	d, _ := db.SetupDB(t)

	req := &pb.BeaconBlocksByRangeRequest{
		StartSlot: 200,
		Step:      21,
		Count:     33,
	}

	endSlot := req.StartSlot + (req.Step * (req.Count - 1))
	// Populate the database with blocks that would match the request.
	for i := endSlot; i >= req.StartSlot; i -= req.Step {
		if err := d.SaveBlock(context.Background(), &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: i}}); err != nil {
			t.Fatal(err)
		}
	}

	// Start service with 160 as allowed blocks capacity (and almost zero capacity recovery).
	r := &Service{p2p: p1, db: d, blocksRateLimiter: leakybucket.NewCollector(0.000001, int64(req.Count*10), false),
		chain: &chainMock.ChainService{}}
	pcl := protocol.ID("/testing")

	var wg sync.WaitGroup
	wg.Add(1)
	p2.BHost.SetStreamHandler(pcl, func(stream network.Stream) {
		defer wg.Done()
		prevSlot := uint64(0)
		for i := req.StartSlot; i < req.StartSlot+req.Count*req.Step; i += req.Step {
			expectSuccess(t, r, stream)
			res := &ethpb.SignedBeaconBlock{}
			if err := r.p2p.Encoding().DecodeWithMaxLength(stream, res); err != nil {
				t.Error(err)
			}
			if res.Block.Slot < prevSlot {
				t.Errorf("Received block is unsorted with slot %d lower than previous slot %d", res.Block.Slot, prevSlot)
			}
			prevSlot = res.Block.Slot
		}
	})

	stream1, err := p1.BHost.NewStream(context.Background(), p2.BHost.ID(), pcl)
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

func TestRPCBeaconBlocksByRange_ReturnsGenesisBlock(t *testing.T) {
	p1 := p2ptest.NewTestP2P(t)
	p2 := p2ptest.NewTestP2P(t)
	p1.Connect(p2)
	if len(p1.BHost.Network().Peers()) != 1 {
		t.Error("Expected peers to be connected")
	}
	d, _ := db.SetupDB(t)

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

	r := &Service{p2p: p1, db: d, blocksRateLimiter: leakybucket.NewCollector(10000, 10000, false), chain: &chainMock.ChainService{}}
	pcl := protocol.ID("/testing")

	var wg sync.WaitGroup
	wg.Add(1)
	p2.BHost.SetStreamHandler(pcl, func(stream network.Stream) {
		defer wg.Done()
		// check for genesis block
		expectSuccess(t, r, stream)
		res := &ethpb.SignedBeaconBlock{}
		if err := r.p2p.Encoding().DecodeWithMaxLength(stream, res); err != nil {
			t.Error(err)
		}
		if res.Block.Slot != 0 {
			t.Fatal("genesis block was not returned")
		}
		for i := req.StartSlot + req.Step; i < req.Count*req.Step; i += req.Step {
			expectSuccess(t, r, stream)
			res := &ethpb.SignedBeaconBlock{}
			if err := r.p2p.Encoding().DecodeWithMaxLength(stream, res); err != nil {
				t.Error(err)
			}
		}
	})

	stream1, err := p1.BHost.NewStream(context.Background(), p2.BHost.ID(), pcl)
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
	d, _ := db.SetupDB(t)
	hook := logTest.NewGlobal()
	saveBlocks := func(req *pb.BeaconBlocksByRangeRequest) {
		// Populate the database with blocks that would match the request.
		for i := req.StartSlot; i < req.StartSlot+(req.Step*req.Count); i += req.Step {
			block := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: i}}
			if err := d.SaveBlock(context.Background(), block); err != nil {
				t.Fatal(err)
			}
		}
	}
	sendRequest := func(p1, p2 *p2ptest.TestP2P, r *Service,
		req *pb.BeaconBlocksByRangeRequest, validateBlocks bool) error {
		var wg sync.WaitGroup
		wg.Add(1)
		pcl := protocol.ID("/testing")
		p2.BHost.SetStreamHandler(pcl, func(stream network.Stream) {
			defer wg.Done()
			if !validateBlocks {
				return
			}
			for i := req.StartSlot; i < req.StartSlot+req.Count*req.Step; i += req.Step {
				expectSuccess(t, r, stream)
				res := &ethpb.SignedBeaconBlock{}
				if err := r.p2p.Encoding().DecodeWithMaxLength(stream, res); err != nil {
					t.Error(err)
				}
				if (res.Block.Slot-req.StartSlot)%req.Step != 0 {
					t.Errorf("Received unexpected block slot %d", res.Block.Slot)
				}
			}
		})
		stream, err := p1.BHost.NewStream(context.Background(), p2.BHost.ID(), pcl)
		if err != nil {
			t.Fatal(err)
		}
		if err = r.beaconBlocksByRangeRPCHandler(context.Background(), req, stream); err != nil {
			return err
		}
		if testutil.WaitTimeout(&wg, 1*time.Second) {
			t.Fatal("Did not receive stream within 1 sec")
		}
		return nil
	}

	t.Run("high request count param and no overflow", func(t *testing.T) {
		p1 := p2ptest.NewTestP2P(t)
		p2 := p2ptest.NewTestP2P(t)
		p1.Connect(p2)
		if len(p1.BHost.Network().Peers()) != 1 {
			t.Error("Expected peers to be connected")
		}

		capacity := int64(flags.Get().BlockBatchLimit * 3)
		r := &Service{p2p: p1, db: d, blocksRateLimiter: leakybucket.NewCollector(0.000001, capacity, false), chain: &chainMock.ChainService{}}

		req := &pb.BeaconBlocksByRangeRequest{
			StartSlot: 100,
			Step:      5,
			Count:     uint64(capacity),
		}
		saveBlocks(req)

		hook.Reset()
		if err := sendRequest(p1, p2, r, req, true); err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		testutil.AssertLogsDoNotContain(t, hook, "Disconnecting bad peer")

		remainingCapacity := r.blocksRateLimiter.Remaining(p2.PeerID().String())
		expectedCapacity := int64(0) // Whole capacity is used, but no overflow.
		if remainingCapacity != expectedCapacity {
			t.Fatalf("Unexpected rate limiting capacity, expected: %v, got: %v", expectedCapacity, remainingCapacity)
		}
	})

	t.Run("high request count param and overflow", func(t *testing.T) {
		p1 := p2ptest.NewTestP2P(t)
		p2 := p2ptest.NewTestP2P(t)
		p1.Connect(p2)
		if len(p1.BHost.Network().Peers()) != 1 {
			t.Error("Expected peers to be connected")
		}

		capacity := int64(flags.Get().BlockBatchLimit * 3)
		r := &Service{p2p: p1, db: d, blocksRateLimiter: leakybucket.NewCollector(0.000001, capacity, false), chain: &chainMock.ChainService{}}

		req := &pb.BeaconBlocksByRangeRequest{
			StartSlot: 100,
			Step:      5,
			Count:     uint64(capacity + 1),
		}
		saveBlocks(req)

		hook.Reset()
		for i := 0; i < p2.Peers().Scorer().BadResponsesThreshold(); i++ {
			err := sendRequest(p1, p2, r, req, false)
			if err == nil || err.Error() != rateLimitedError {
				t.Errorf("Expected error not thrown, want: %v, got: %v", rateLimitedError, err)
			}
		}
		// Make sure that we were blocked indeed.
		testutil.AssertLogsContain(t, hook, "Disconnecting bad peer")

		remainingCapacity := r.blocksRateLimiter.Remaining(p2.PeerID().String())
		expectedCapacity := int64(0) // Whole capacity is used.
		if remainingCapacity != expectedCapacity {
			t.Fatalf("Unexpected rate limiting capacity, expected: %v, got: %v", expectedCapacity, remainingCapacity)
		}
	})

	t.Run("many requests with count set to max blocks per second", func(t *testing.T) {
		p1 := p2ptest.NewTestP2P(t)
		p2 := p2ptest.NewTestP2P(t)
		p1.Connect(p2)
		if len(p1.BHost.Network().Peers()) != 1 {
			t.Error("Expected peers to be connected")
		}

		capacity := int64(flags.Get().BlockBatchLimit * flags.Get().BlockBatchLimitBurstFactor)
		r := &Service{p2p: p1, db: d, blocksRateLimiter: leakybucket.NewCollector(0.000001, capacity, false), chain: &chainMock.ChainService{}}

		req := &pb.BeaconBlocksByRangeRequest{
			StartSlot: 100,
			Step:      1,
			Count:     uint64(flags.Get().BlockBatchLimit),
		}
		saveBlocks(req)

		hook.Reset()
		for i := 0; i < flags.Get().BlockBatchLimitBurstFactor; i++ {
			if err := sendRequest(p1, p2, r, req, true); err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		}
		testutil.AssertLogsDoNotContain(t, hook, "Disconnecting bad peer")

		// One more request should result in overflow.
		hook.Reset()
		for i := 0; i < p2.Peers().Scorer().BadResponsesThreshold(); i++ {
			err := sendRequest(p1, p2, r, req, false)
			if err == nil || err.Error() != rateLimitedError {
				t.Errorf("Expected error not thrown, want: %v, got: %v", rateLimitedError, err)
			}
		}
		testutil.AssertLogsContain(t, hook, "Disconnecting bad peer")

		remainingCapacity := r.blocksRateLimiter.Remaining(p2.PeerID().String())
		expectedCapacity := int64(0) // Whole capacity is used.
		if remainingCapacity != expectedCapacity {
			t.Fatalf("Unexpected rate limiting capacity, expected: %v, got: %v", expectedCapacity, remainingCapacity)
		}
	})
}
