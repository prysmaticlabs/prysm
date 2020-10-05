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
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
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
			expectSuccess(t, stream)
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
	expectedRoots := make([][32]byte, req.Count)
	// Populate the database with blocks that would match the request.
	for i, j := endSlot, req.Count-1; i >= req.StartSlot; i -= req.Step {
		blk := testutil.NewBeaconBlock()
		blk.Block.Slot = i
		rt, err := blk.Block.HashTreeRoot()
		require.NoError(t, err)
		expectedRoots[j] = rt
		require.NoError(t, d.SaveBlock(context.Background(), blk))
		j--
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
		require.Equal(t, uint64(len(expectedRoots)), req.Count, "Number of roots not expected")
		for i, j := req.StartSlot, 0; i < req.StartSlot+req.Count*req.Step; i += req.Step {
			expectSuccess(t, stream)
			res := &ethpb.SignedBeaconBlock{}
			assert.NoError(t, r.p2p.Encoding().DecodeWithMaxLength(stream, res))
			if res.Block.Slot < prevSlot {
				t.Errorf("Received block is unsorted with slot %d lower than previous slot %d", res.Block.Slot, prevSlot)
			}
			rt, err := res.Block.HashTreeRoot()
			require.NoError(t, err)
			assert.Equal(t, expectedRoots[j], rt, "roots not equal")
			prevSlot = res.Block.Slot
			j++
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
		blk := testutil.NewBeaconBlock()
		blk.Block.Slot = i

		// Save genesis block
		if i == 0 {
			rt, err := blk.Block.HashTreeRoot()
			require.NoError(t, err)
			require.NoError(t, d.SaveGenesisBlockRoot(context.Background(), rt))
		}
		require.NoError(t, d.SaveBlock(context.Background(), blk))
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
		expectSuccess(t, stream)
		res := &ethpb.SignedBeaconBlock{}
		assert.NoError(t, r.p2p.Encoding().DecodeWithMaxLength(stream, res))
		assert.Equal(t, uint64(0), res.Block.Slot, "genesis block was not returned")
		for i := req.StartSlot + req.Step; i < req.Count*req.Step; i += req.Step {
			expectSuccess(t, stream)
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
			block := testutil.NewBeaconBlock()
			block.Block.Slot = i
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
				expectSuccess(t, stream)
				res := testutil.NewBeaconBlock()
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
		require.LogsDoNotContain(t, hook, "Disconnecting bad peer")

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
		require.LogsContain(t, hook, "Disconnecting bad peer")

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
		require.LogsDoNotContain(t, hook, "Disconnecting bad peer")

		// One more request should result in overflow.
		hook.Reset()
		for i := 0; i < p2.Peers().Scorers().BadResponsesScorer().Params().Threshold; i++ {
			err := sendRequest(p1, p2, r, req, false)
			assert.ErrorContains(t, rateLimitedError, err)
		}
		require.LogsContain(t, hook, "Disconnecting bad peer")

		remainingCapacity := r.rateLimiter.limiterMap[topic].Remaining(p2.PeerID().String())
		expectedCapacity := int64(0) // Whole capacity is used.
		assert.Equal(t, expectedCapacity, remainingCapacity, "Unexpected rate limiting capacity")
	})
}

func TestRPCBeaconBlocksByRange_validateRangeRequest(t *testing.T) {
	slotsSinceGenesis := 1000
	r := &Service{chain: &chainMock.ChainService{
		Genesis: time.Now().Add(time.Second * time.Duration(-slotsSinceGenesis*int(params.BeaconConfig().SecondsPerSlot))),
	}}

	tests := []struct {
		name          string
		req           *pb.BeaconBlocksByRangeRequest
		expectedError string
		errorToLog    string
	}{
		{
			name: "Zero Count",
			req: &pb.BeaconBlocksByRangeRequest{
				Count: 0,
				Step:  1,
			},
			expectedError: reqError,
			errorToLog:    "validation did not fail with bad count",
		},
		{
			name: "Over limit Count",
			req: &pb.BeaconBlocksByRangeRequest{
				Count: params.BeaconNetworkConfig().MaxRequestBlocks + 1,
				Step:  1,
			},
			expectedError: reqError,
			errorToLog:    "validation did not fail with bad count",
		},
		{
			name: "Correct Count",
			req: &pb.BeaconBlocksByRangeRequest{
				Count: params.BeaconNetworkConfig().MaxRequestBlocks - 1,
				Step:  1,
			},
			errorToLog: "validation failed with correct count",
		},
		{
			name: "Zero Step",
			req: &pb.BeaconBlocksByRangeRequest{
				Step:  0,
				Count: 1,
			},
			expectedError: reqError,
			errorToLog:    "validation did not fail with bad step",
		},
		{
			name: "Over limit Step",
			req: &pb.BeaconBlocksByRangeRequest{
				Step:  rangeLimit + 1,
				Count: 1,
			},
			expectedError: reqError,
			errorToLog:    "validation did not fail with bad step",
		},
		{
			name: "Correct Step",
			req: &pb.BeaconBlocksByRangeRequest{
				Step:  rangeLimit - 1,
				Count: 2,
			},
			errorToLog: "validation failed with correct step",
		},
		{
			name: "Over Limit Start Slot",
			req: &pb.BeaconBlocksByRangeRequest{
				StartSlot: uint64(slotsSinceGenesis) + (2 * rangeLimit) + 1,
				Step:      1,
				Count:     1,
			},
			expectedError: reqError,
			errorToLog:    "validation did not fail with bad start slot",
		},
		{
			name: "Over Limit End Slot",
			req: &pb.BeaconBlocksByRangeRequest{
				Step:  1,
				Count: params.BeaconNetworkConfig().MaxRequestBlocks + 1,
			},
			expectedError: reqError,
			errorToLog:    "validation did not fail with bad end slot",
		},
		{
			name: "Exceed Range Limit",
			req: &pb.BeaconBlocksByRangeRequest{
				Step:  3,
				Count: uint64(slotsSinceGenesis / 2),
			},
			expectedError: reqError,
			errorToLog:    "validation did not fail with bad range",
		},
		{
			name: "Valid Request",
			req: &pb.BeaconBlocksByRangeRequest{
				Step:      1,
				Count:     params.BeaconNetworkConfig().MaxRequestBlocks - 1,
				StartSlot: 50,
			},
			errorToLog: "validation failed with valid params",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectedError != "" {
				assert.ErrorContains(t, tt.expectedError, r.validateRangeRequest(tt.req), tt.errorToLog)
			} else {
				assert.NoError(t, r.validateRangeRequest(tt.req), tt.errorToLog)
			}
		})
	}
}

func TestRPCBeaconBlocksByRange_EnforceResponseInvariants(t *testing.T) {
	d, _ := db.SetupDB(t)
	hook := logTest.NewGlobal()
	saveBlocks := func(req *pb.BeaconBlocksByRangeRequest) {
		// Populate the database with blocks that would match the request.
		for i := req.StartSlot; i < req.StartSlot+(req.Step*req.Count); i += req.Step {
			block := testutil.NewBeaconBlock()
			block.Block.Slot = i
			require.NoError(t, d.SaveBlock(context.Background(), block))
		}
	}
	pcl := protocol.ID("/testing")
	sendRequest := func(p1, p2 *p2ptest.TestP2P, r *Service,
		req *pb.BeaconBlocksByRangeRequest, processBlocks func([]*ethpb.SignedBeaconBlock)) error {
		var wg sync.WaitGroup
		wg.Add(1)
		p2.BHost.SetStreamHandler(pcl, func(stream network.Stream) {
			defer wg.Done()
			blocks := make([]*ethpb.SignedBeaconBlock, 0, req.Count)
			for i := req.StartSlot; i < req.StartSlot+req.Count*req.Step; i += req.Step {
				expectSuccess(t, stream)
				blk := testutil.NewBeaconBlock()
				assert.NoError(t, r.p2p.Encoding().DecodeWithMaxLength(stream, blk))
				if (blk.Block.Slot-req.StartSlot)%req.Step != 0 {
					t.Errorf("Received unexpected block slot %d", blk.Block.Slot)
				}
				blocks = append(blocks, blk)
			}
			processBlocks(blocks)
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

	t.Run("assert range", func(t *testing.T) {
		p1 := p2ptest.NewTestP2P(t)
		p2 := p2ptest.NewTestP2P(t)
		p1.Connect(p2)
		assert.Equal(t, 1, len(p1.BHost.Network().Peers()), "Expected peers to be connected")

		r := &Service{p2p: p1, db: d, chain: &chainMock.ChainService{}, rateLimiter: newRateLimiter(p1)}
		r.rateLimiter.limiterMap[string(pcl)] = leakybucket.NewCollector(0.000001, 640, false)
		req := &pb.BeaconBlocksByRangeRequest{
			StartSlot: 448,
			Step:      1,
			Count:     64,
		}
		saveBlocks(req)

		hook.Reset()
		err := sendRequest(p1, p2, r, req, func(blocks []*ethpb.SignedBeaconBlock) {
			assert.Equal(t, req.Count, uint64(len(blocks)))
			for _, blk := range blocks {
				if blk.Block.Slot < req.StartSlot || blk.Block.Slot >= req.StartSlot+req.Count*req.Step {
					t.Errorf("Block slot is out of range: %d is not within [%d, %d)",
						blk.Block.Slot, req.StartSlot, req.StartSlot+req.Count*req.Step)
				}
			}
		})
		assert.NoError(t, err)
		require.LogsDoNotContain(t, hook, "Disconnecting bad peer")
	})
}
