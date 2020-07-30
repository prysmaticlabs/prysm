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
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestRPCBeaconBlocksByRange_RPCHandlerReturnsBlocks(t *testing.T) {
	p1 := p2ptest.NewTestP2P(t)
	p2 := p2ptest.NewTestP2P(t)
	p1.Connect(p2)
	assert.Equal(t, 1, len(p1.BHost.Network().Peers()), "Expected peers to be connected")
	d, _ := db.SetupDB(t)

	req := &pb.BeaconBlocksByRangeRequest{
		StartSlot: 100,
		Step:      64,
		Count:     16,
	}

	// Populate the database with blocks that would match the request.
	for i := req.StartSlot; i < req.StartSlot+(req.Step*req.Count); i += req.Step {
		blk := testutil.NewBeaconBlock()
		blk.Block.Slot = i
		require.NoError(t, d.SaveBlock(context.Background(), blk))
	}

	// Start service with 160 as allowed blocks capacity (and almost zero capacity recovery).
	r := &Service{p2p: p1, db: d, chain: &chainMock.ChainService{}, rateLimiter: newRateLimiter(p1)}
	pcl := protocol.ID("/testing")
	topic := string(pcl)
	r.rateLimiter.limiterMap[topic] = leakybucket.NewCollector(0.000001, int64(req.Count*10), false)
	var wg sync.WaitGroup
	wg.Add(1)
	p2.BHost.SetStreamHandler(pcl, func(stream network.Stream) {
		defer wg.Done()
		for i := req.StartSlot; i < req.StartSlot+req.Count*req.Step; i += req.Step {
			expectSuccess(t, r, stream)
			res := testutil.NewBeaconBlock()
			assert.NoError(t, r.p2p.Encoding().DecodeWithMaxLength(stream, res))
			if (res.Block.Slot-req.StartSlot)%req.Step != 0 {
				t.Errorf("Received unexpected block slot %d", res.Block.Slot)
			}
		}
	})

	stream1, err := p1.BHost.NewStream(context.Background(), p2.BHost.ID(), pcl)
	require.NoError(t, err)

	err = r.beaconBlocksByRangeRPCHandler(context.Background(), req, stream1)
	require.NoError(t, err)

	// Make sure that rate limiter doesn't limit capacity exceedingly.
	remainingCapacity := r.rateLimiter.limiterMap[topic].Remaining(p2.PeerID().String())
	expectedCapacity := int64(req.Count*10 - req.Count)
	require.Equal(t, expectedCapacity, remainingCapacity, "Unexpected rate limiting capacity")

	if testutil.WaitTimeout(&wg, 1*time.Second) {
		t.Fatal("Did not receive stream within 1 sec")
	}
}

func TestRPCBeaconBlocksByRange_RPCHandlerReturnsSortedBlocks(t *testing.T) {
	p1 := p2ptest.NewTestP2P(t)
	p2 := p2ptest.NewTestP2P(t)
	p1.Connect(p2)
	assert.Equal(t, 1, len(p1.BHost.Network().Peers()), "Expected peers to be connected")
	d, _ := db.SetupDB(t)

	req := &pb.BeaconBlocksByRangeRequest{
		StartSlot: 200,
		Step:      21,
		Count:     33,
	}

	endSlot := req.StartSlot + (req.Step * (req.Count - 1))
	// Populate the database with blocks that would match the request.
	for i := endSlot; i >= req.StartSlot; i -= req.Step {
		require.NoError(t, d.SaveBlock(context.Background(), &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: i}}))
	}

	// Start service with 160 as allowed blocks capacity (and almost zero capacity recovery).
	r := &Service{p2p: p1, db: d, rateLimiter: newRateLimiter(p1),
		chain: &chainMock.ChainService{}}
	pcl := protocol.ID("/testing")
	topic := string(pcl)
	r.rateLimiter.limiterMap[topic] = leakybucket.NewCollector(0.000001, int64(req.Count*10), false)

	var wg sync.WaitGroup
	wg.Add(1)
	p2.BHost.SetStreamHandler(pcl, func(stream network.Stream) {
		defer wg.Done()
		prevSlot := uint64(0)
		for i := req.StartSlot; i < req.StartSlot+req.Count*req.Step; i += req.Step {
			expectSuccess(t, r, stream)
			res := &ethpb.SignedBeaconBlock{}
			assert.NoError(t, r.p2p.Encoding().DecodeWithMaxLength(stream, res))
			if res.Block.Slot < prevSlot {
				t.Errorf("Received block is unsorted with slot %d lower than previous slot %d", res.Block.Slot, prevSlot)
			}
			prevSlot = res.Block.Slot
		}
	})

	stream1, err := p1.BHost.NewStream(context.Background(), p2.BHost.ID(), pcl)
	require.NoError(t, err)
	require.NoError(t, r.beaconBlocksByRangeRPCHandler(context.Background(), req, stream1))

	if testutil.WaitTimeout(&wg, 1*time.Second) {
		t.Fatal("Did not receive stream within 1 sec")
	}
}

func TestRPCBeaconBlocksByRange_ReturnsGenesisBlock(t *testing.T) {
	p1 := p2ptest.NewTestP2P(t)
	p2 := p2ptest.NewTestP2P(t)
	p1.Connect(p2)
	assert.Equal(t, 1, len(p1.BHost.Network().Peers()), "Expected peers to be connected")
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
			require.NoError(t, err)
			require.NoError(t, d.SaveGenesisBlockRoot(context.Background(), rt))
		}
		require.NoError(t, d.SaveBlock(context.Background(), &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: i}}))
	}

	r := &Service{p2p: p1, db: d, chain: &chainMock.ChainService{}, rateLimiter: newRateLimiter(p1)}
	pcl := protocol.ID("/testing")
	topic := string(pcl)
	r.rateLimiter.limiterMap[topic] = leakybucket.NewCollector(10000, 10000, false)

	var wg sync.WaitGroup
	wg.Add(1)
	p2.BHost.SetStreamHandler(pcl, func(stream network.Stream) {
		defer wg.Done()
		// check for genesis block
		expectSuccess(t, r, stream)
		res := &ethpb.SignedBeaconBlock{}
		assert.NoError(t, r.p2p.Encoding().DecodeWithMaxLength(stream, res))
		assert.Equal(t, uint64(0), res.Block.Slot, "genesis block was not returned")
		for i := req.StartSlot + req.Step; i < req.Count*req.Step; i += req.Step {
			expectSuccess(t, r, stream)
			res := &ethpb.SignedBeaconBlock{}
			assert.NoError(t, r.p2p.Encoding().DecodeWithMaxLength(stream, res))
		}
	})

	stream1, err := p1.BHost.NewStream(context.Background(), p2.BHost.ID(), pcl)
	require.NoError(t, err)
	require.NoError(t, r.beaconBlocksByRangeRPCHandler(context.Background(), req, stream1))

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
			require.NoError(t, d.SaveBlock(context.Background(), block))
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
				assert.NoError(t, r.p2p.Encoding().DecodeWithMaxLength(stream, res))
				if (res.Block.Slot-req.StartSlot)%req.Step != 0 {
					t.Errorf("Received unexpected block slot %d", res.Block.Slot)
				}
			}
		})
		stream, err := p1.BHost.NewStream(context.Background(), p2.BHost.ID(), pcl)
		require.NoError(t, err)
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
		assert.Equal(t, 1, len(p1.BHost.Network().Peers()), "Expected peers to be connected")

		capacity := int64(flags.Get().BlockBatchLimit * 3)
		r := &Service{p2p: p1, db: d, chain: &chainMock.ChainService{}, rateLimiter: newRateLimiter(p1)}

		pcl := protocol.ID("/testing")
		topic := string(pcl)
		r.rateLimiter.limiterMap[topic] = leakybucket.NewCollector(0.000001, capacity, false)
		req := &pb.BeaconBlocksByRangeRequest{
			StartSlot: 100,
			Step:      5,
			Count:     uint64(capacity),
		}
		saveBlocks(req)

		hook.Reset()
		assert.NoError(t, sendRequest(p1, p2, r, req, true))
		testutil.AssertLogsDoNotContain(t, hook, "Disconnecting bad peer")

		remainingCapacity := r.rateLimiter.limiterMap[topic].Remaining(p2.PeerID().String())
		expectedCapacity := int64(0) // Whole capacity is used, but no overflow.
		assert.Equal(t, expectedCapacity, remainingCapacity, "Unexpected rate limiting capacity")
	})

	t.Run("high request count param and overflow", func(t *testing.T) {
		p1 := p2ptest.NewTestP2P(t)
		p2 := p2ptest.NewTestP2P(t)
		p1.Connect(p2)
		assert.Equal(t, 1, len(p1.BHost.Network().Peers()), "Expected peers to be connected")

		capacity := int64(flags.Get().BlockBatchLimit * 3)
		r := &Service{p2p: p1, db: d, chain: &chainMock.ChainService{}, rateLimiter: newRateLimiter(p1)}

		pcl := protocol.ID("/testing")
		topic := string(pcl)
		r.rateLimiter.limiterMap[topic] = leakybucket.NewCollector(0.000001, capacity, false)

		req := &pb.BeaconBlocksByRangeRequest{
			StartSlot: 100,
			Step:      5,
			Count:     uint64(capacity + 1),
		}
		saveBlocks(req)

		hook.Reset()
		for i := 0; i < p2.Peers().Scorers().BadResponsesScorer().Params().Threshold; i++ {
			err := sendRequest(p1, p2, r, req, false)
			assert.ErrorContains(t, rateLimitedError, err)
		}
		// Make sure that we were blocked indeed.
		testutil.AssertLogsContain(t, hook, "Disconnecting bad peer")

		remainingCapacity := r.rateLimiter.limiterMap[topic].Remaining(p2.PeerID().String())
		expectedCapacity := int64(0) // Whole capacity is used.
		assert.Equal(t, expectedCapacity, remainingCapacity, "Unexpected rate limiting capacity")
	})

	t.Run("many requests with count set to max blocks per second", func(t *testing.T) {
		p1 := p2ptest.NewTestP2P(t)
		p2 := p2ptest.NewTestP2P(t)
		p1.Connect(p2)
		assert.Equal(t, 1, len(p1.BHost.Network().Peers()), "Expected peers to be connected")

		capacity := int64(flags.Get().BlockBatchLimit * flags.Get().BlockBatchLimitBurstFactor)
		r := &Service{p2p: p1, db: d, chain: &chainMock.ChainService{}, rateLimiter: newRateLimiter(p1)}
		pcl := protocol.ID("/testing")
		topic := string(pcl)
		r.rateLimiter.limiterMap[topic] = leakybucket.NewCollector(0.000001, capacity, false)

		req := &pb.BeaconBlocksByRangeRequest{
			StartSlot: 100,
			Step:      1,
			Count:     uint64(flags.Get().BlockBatchLimit),
		}
		saveBlocks(req)

		hook.Reset()
		for i := 0; i < flags.Get().BlockBatchLimitBurstFactor; i++ {
			assert.NoError(t, sendRequest(p1, p2, r, req, true))
		}
		testutil.AssertLogsDoNotContain(t, hook, "Disconnecting bad peer")

		// One more request should result in overflow.
		hook.Reset()
		for i := 0; i < p2.Peers().Scorers().BadResponsesScorer().Params().Threshold; i++ {
			err := sendRequest(p1, p2, r, req, false)
			assert.ErrorContains(t, rateLimitedError, err)
		}
		testutil.AssertLogsContain(t, hook, "Disconnecting bad peer")

		remainingCapacity := r.rateLimiter.limiterMap[topic].Remaining(p2.PeerID().String())
		expectedCapacity := int64(0) // Whole capacity is used.
		assert.Equal(t, expectedCapacity, remainingCapacity, "Unexpected rate limiting capacity")
	})
}
