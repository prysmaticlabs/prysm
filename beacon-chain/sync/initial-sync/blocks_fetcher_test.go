package initialsync

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/kevinms/leakybucket-go"
	core "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	dbtest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/flags"
	p2pm "github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	p2pt "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	p2ppb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
	"github.com/prysmaticlabs/prysm/shared/sliceutil"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/sirupsen/logrus"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestBlocksFetcher_InitStartStop(t *testing.T) {
	mc, p2p, _ := initializeTestServices(t, []uint64{}, []*peerData{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	fetcher := newBlocksFetcher(
		ctx,
		&blocksFetcherConfig{
			headFetcher: mc,
			p2p:         p2p,
		},
	)

	t.Run("check for leaked goroutines", func(t *testing.T) {
		err := fetcher.start()
		if err != nil {
			t.Error(err)
		}
		fetcher.stop() // should block up until all resources are reclaimed
		select {
		case <-fetcher.requestResponses():
		default:
			t.Error("fetchResponses channel is leaked")
		}
	})

	t.Run("re-starting of stopped fetcher", func(t *testing.T) {
		if err := fetcher.start(); err == nil {
			t.Errorf("expected error not returned: %v", errFetcherCtxIsDone)
		}
	})

	t.Run("multiple stopping attempts", func(t *testing.T) {
		fetcher := newBlocksFetcher(
			context.Background(),
			&blocksFetcherConfig{
				headFetcher: mc,
				p2p:         p2p,
			})
		if err := fetcher.start(); err != nil {
			t.Error(err)
		}

		fetcher.stop()
		fetcher.stop()
	})

	t.Run("cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		fetcher := newBlocksFetcher(
			ctx,
			&blocksFetcherConfig{
				headFetcher: mc,
				p2p:         p2p,
			})
		if err := fetcher.start(); err != nil {
			t.Error(err)
		}

		cancel()
		fetcher.stop()
	})
}

func TestBlocksFetcher_RoundRobin(t *testing.T) {
	blockBatchLimit := uint64(flags.Get().BlockBatchLimit)
	requestsGenerator := func(start, end uint64, batchSize uint64) []*fetchRequestParams {
		var requests []*fetchRequestParams
		for i := start; i <= end; i += batchSize {
			requests = append(requests, &fetchRequestParams{
				start: i,
				count: batchSize,
			})
		}
		return requests
	}
	tests := []struct {
		name               string
		expectedBlockSlots []uint64
		peers              []*peerData
		requests           []*fetchRequestParams
	}{
		{
			name:               "Single peer with all blocks",
			expectedBlockSlots: makeSequence(1, 3*blockBatchLimit),
			peers: []*peerData{
				{
					blocks:         makeSequence(1, 3*blockBatchLimit),
					finalizedEpoch: helpers.SlotToEpoch(3 * blockBatchLimit),
					headSlot:       3 * blockBatchLimit,
				},
			},
			requests: requestsGenerator(1, 3*blockBatchLimit, blockBatchLimit),
		},
		{
			name:               "Single peer with all blocks (many small requests)",
			expectedBlockSlots: makeSequence(1, 3*blockBatchLimit),
			peers: []*peerData{
				{
					blocks:         makeSequence(1, 3*blockBatchLimit),
					finalizedEpoch: helpers.SlotToEpoch(3 * blockBatchLimit),
					headSlot:       3 * blockBatchLimit,
				},
			},
			requests: requestsGenerator(1, 3*blockBatchLimit, blockBatchLimit/4),
		},
		{
			name:               "Multiple peers with all blocks",
			expectedBlockSlots: makeSequence(1, 3*blockBatchLimit),
			peers: []*peerData{
				{
					blocks:         makeSequence(1, 3*blockBatchLimit),
					finalizedEpoch: helpers.SlotToEpoch(3 * blockBatchLimit),
					headSlot:       3 * blockBatchLimit,
				},
				{
					blocks:         makeSequence(1, 3*blockBatchLimit),
					finalizedEpoch: helpers.SlotToEpoch(3 * blockBatchLimit),
					headSlot:       3 * blockBatchLimit,
				},
				{
					blocks:         makeSequence(1, 3*blockBatchLimit),
					finalizedEpoch: helpers.SlotToEpoch(3 * blockBatchLimit),
					headSlot:       3 * blockBatchLimit,
				},
			},
			requests: requestsGenerator(1, 3*blockBatchLimit, blockBatchLimit),
		},
		{
			name:               "Multiple peers with skipped slots",
			expectedBlockSlots: append(makeSequence(1, 64), makeSequence(500, 640)...), // up to 18th epoch
			peers: []*peerData{
				{
					blocks:         append(makeSequence(1, 64), makeSequence(500, 640)...),
					finalizedEpoch: 18,
					headSlot:       640,
				},
				{
					blocks:         append(makeSequence(1, 64), makeSequence(500, 640)...),
					finalizedEpoch: 18,
					headSlot:       640,
				},
				{
					blocks:         append(makeSequence(1, 64), makeSequence(500, 640)...),
					finalizedEpoch: 18,
					headSlot:       640,
				},
				{
					blocks:         append(makeSequence(1, 64), makeSequence(500, 640)...),
					finalizedEpoch: 18,
					headSlot:       640,
				},
				{
					blocks:         append(makeSequence(1, 64), makeSequence(500, 640)...),
					finalizedEpoch: 18,
					headSlot:       640,
				},
			},
			requests: []*fetchRequestParams{
				{
					start: 1,
					count: blockBatchLimit,
				},
				{
					start: blockBatchLimit + 1,
					count: blockBatchLimit,
				},
				{
					start: 2*blockBatchLimit + 1,
					count: blockBatchLimit,
				},
				{
					start: 500,
					count: 53,
				},
				{
					start: 553,
					count: 200,
				},
			},
		},
		{
			name:               "Multiple peers with failures",
			expectedBlockSlots: makeSequence(1, 2*blockBatchLimit),
			peers: []*peerData{
				{
					blocks:         makeSequence(1, 320),
					finalizedEpoch: 8,
					headSlot:       320,
				},
				{
					blocks:         makeSequence(1, 320),
					finalizedEpoch: 8,
					headSlot:       320,
					failureSlots:   makeSequence(1, 32), // first epoch
				},
				{
					blocks:         makeSequence(1, 320),
					finalizedEpoch: 8,
					headSlot:       320,
				},
				{
					blocks:         makeSequence(1, 320),
					finalizedEpoch: 8,
					headSlot:       320,
				},
			},
			requests: []*fetchRequestParams{
				{
					start: 1,
					count: blockBatchLimit,
				},
				{
					start: blockBatchLimit + 1,
					count: blockBatchLimit,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache.initializeRootCache(tt.expectedBlockSlots, t)

			beaconDB := dbtest.SetupDB(t)

			p := p2pt.NewTestP2P(t)
			connectPeers(t, p, tt.peers, p.Peers())
			cache.RLock()
			genesisRoot := cache.rootCache[0]
			cache.RUnlock()

			err := beaconDB.SaveBlock(context.Background(), &eth.SignedBeaconBlock{
				Block: &eth.BeaconBlock{
					Slot: 0,
				}})
			if err != nil {
				t.Fatal(err)
			}

			st, err := stateTrie.InitializeFromProto(&p2ppb.BeaconState{})
			if err != nil {
				t.Fatal(err)
			}
			mc := &mock.ChainService{
				State: st,
				Root:  genesisRoot[:],
				DB:    beaconDB,
			}

			ctx, cancel := context.WithCancel(context.Background())
			fetcher := newBlocksFetcher(
				ctx,
				&blocksFetcherConfig{
					headFetcher: mc,
					p2p:         p,
				})

			err = fetcher.start()
			if err != nil {
				t.Error(err)
			}

			var wg sync.WaitGroup
			wg.Add(len(tt.requests)) // how many block requests we are going to make
			go func() {
				wg.Wait()
				log.Debug("Stopping fetcher")
				fetcher.stop()
			}()

			processFetchedBlocks := func() ([]*eth.SignedBeaconBlock, error) {
				defer cancel()
				var unionRespBlocks []*eth.SignedBeaconBlock

				for {
					select {
					case resp, ok := <-fetcher.requestResponses():
						if !ok { // channel closed, aggregate
							return unionRespBlocks, nil
						}

						if resp.err != nil {
							log.WithError(resp.err).Debug("Block fetcher returned error")
						} else {
							unionRespBlocks = append(unionRespBlocks, resp.blocks...)
							if len(resp.blocks) == 0 {
								log.WithFields(logrus.Fields{
									"start": resp.start,
									"count": resp.count,
								}).Debug("Received empty slot")
							}
						}

						wg.Done()
					}
				}
			}

			maxExpectedBlocks := uint64(0)
			for _, requestParams := range tt.requests {
				err = fetcher.scheduleRequest(context.Background(), requestParams.start, requestParams.count)
				if err != nil {
					t.Error(err)
				}
				maxExpectedBlocks += requestParams.count
			}

			blocks, err := processFetchedBlocks()
			if err != nil {
				t.Error(err)
			}

			sort.Slice(blocks, func(i, j int) bool {
				return blocks[i].Block.Slot < blocks[j].Block.Slot
			})

			slots := make([]uint64, len(blocks))
			for i, block := range blocks {
				slots[i] = block.Block.Slot
			}

			log.WithFields(logrus.Fields{
				"blocksLen": len(blocks),
				"slots":     slots,
			}).Debug("Finished block fetching")

			if len(blocks) > int(maxExpectedBlocks) {
				t.Errorf("Too many blocks returned. Wanted %d got %d", maxExpectedBlocks, len(blocks))
			}
			if len(blocks) != len(tt.expectedBlockSlots) {
				t.Errorf("Processes wrong number of blocks. Wanted %d got %d", len(tt.expectedBlockSlots), len(blocks))
			}
			var receivedBlockSlots []uint64
			for _, blk := range blocks {
				receivedBlockSlots = append(receivedBlockSlots, blk.Block.Slot)
			}
			missing := sliceutil.NotUint64(
				sliceutil.IntersectionUint64(tt.expectedBlockSlots, receivedBlockSlots), tt.expectedBlockSlots)
			if len(missing) > 0 {
				t.Errorf("Missing blocks at slots %v", missing)
			}
		})
	}
}

func TestBlocksFetcher_scheduleRequest(t *testing.T) {
	blockBatchLimit := uint64(flags.Get().BlockBatchLimit)
	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		fetcher := newBlocksFetcher(ctx, &blocksFetcherConfig{
			headFetcher: nil,
			p2p:         nil,
		})
		cancel()
		if err := fetcher.scheduleRequest(ctx, 1, blockBatchLimit); err == nil {
			t.Errorf("expected error: %v", errFetcherCtxIsDone)
		}
	})
}
func TestBlocksFetcher_handleRequest(t *testing.T) {
	// Handle using default configuration.
	t.Run("default config", func(t *testing.T) {
		_handleRequest(t)
	})

	// Now handle using previous implementation, w/o WRR.
	t.Run("previous config", func(t *testing.T) {
		resetCfg := featureconfig.InitWithReset(&featureconfig.Flags{
			EnableInitSyncWeightedRoundRobin: false,
		})
		defer resetCfg()
		_handleRequest(t)
	})
}

// TODO(6024): Move to TestBlocksFetcher_handleRequest when EnableInitSyncWeightedRoundRobin is released.
func _handleRequest(t *testing.T) {
	blockBatchLimit := uint64(flags.Get().BlockBatchLimit)
	chainConfig := struct {
		expectedBlockSlots []uint64
		peers              []*peerData
	}{
		expectedBlockSlots: makeSequence(1, blockBatchLimit),
		peers: []*peerData{
			{
				blocks:         makeSequence(1, 320),
				finalizedEpoch: 8,
				headSlot:       320,
			},
			{
				blocks:         makeSequence(1, 320),
				finalizedEpoch: 8,
				headSlot:       320,
			},
		},
	}

	mc, p2p, _ := initializeTestServices(t, chainConfig.expectedBlockSlots, chainConfig.peers)

	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		fetcher := newBlocksFetcher(ctx, &blocksFetcherConfig{
			headFetcher: mc,
			p2p:         p2p,
		})

		cancel()
		response := fetcher.handleRequest(ctx, 1, blockBatchLimit)
		if response.err == nil {
			t.Errorf("expected error: %v", errFetcherCtxIsDone)
		}
	})

	t.Run("receive blocks", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		fetcher := newBlocksFetcher(ctx, &blocksFetcherConfig{
			headFetcher: mc,
			p2p:         p2p,
		})

		requestCtx, reqCancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer reqCancel()
		go func() {
			response := fetcher.handleRequest(requestCtx, 1 /* start */, blockBatchLimit /* count */)
			select {
			case <-ctx.Done():
			case fetcher.fetchResponses <- response:
			}
		}()

		var blocks []*eth.SignedBeaconBlock
		select {
		case <-ctx.Done():
			t.Error(ctx.Err())
		case resp := <-fetcher.requestResponses():
			if resp.err != nil {
				t.Error(resp.err)
			} else {
				blocks = resp.blocks
			}
		}
		if uint64(len(blocks)) != blockBatchLimit {
			t.Errorf("incorrect number of blocks returned, expected: %v, got: %v", blockBatchLimit, len(blocks))
		}

		var receivedBlockSlots []uint64
		for _, blk := range blocks {
			receivedBlockSlots = append(receivedBlockSlots, blk.Block.Slot)
		}
		missing := sliceutil.NotUint64(
			sliceutil.IntersectionUint64(chainConfig.expectedBlockSlots, receivedBlockSlots),
			chainConfig.expectedBlockSlots)
		if len(missing) > 0 {
			t.Errorf("Missing blocks at slots %v", missing)
		}
	})
}

func TestBlocksFetcher_requestBeaconBlocksByRange(t *testing.T) {
	blockBatchLimit := uint64(flags.Get().BlockBatchLimit)
	chainConfig := struct {
		expectedBlockSlots []uint64
		peers              []*peerData
	}{
		expectedBlockSlots: makeSequence(1, 320),
		peers: []*peerData{
			{
				blocks:         makeSequence(1, 320),
				finalizedEpoch: 8,
				headSlot:       320,
			},
			{
				blocks:         makeSequence(1, 320),
				finalizedEpoch: 8,
				headSlot:       320,
			},
		},
	}

	hook := logTest.NewGlobal()
	mc, p2p, _ := initializeTestServices(t, chainConfig.expectedBlockSlots, chainConfig.peers)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fetcher := newBlocksFetcher(
		ctx,
		&blocksFetcherConfig{
			headFetcher: mc,
			p2p:         p2p,
		})

	root, _, peers := p2p.Peers().BestFinalized(params.BeaconConfig().MaxPeersToSync, helpers.SlotToEpoch(mc.HeadSlot()))

	blocks, err := fetcher.requestBeaconBlocksByRange(context.Background(), peers[0], root, 1, 1, blockBatchLimit)
	if err != nil {
		t.Errorf("error: %v", err)
	}
	if uint64(len(blocks)) != blockBatchLimit {
		t.Errorf("incorrect number of blocks returned, expected: %v, got: %v", blockBatchLimit, len(blocks))
	}

	if !featureconfig.Get().EnableInitSyncWeightedRoundRobin {
		// Test request fail over (success).
		err = fetcher.p2p.Disconnect(peers[0])
		if err != nil {
			t.Error(err)
		}
		blocks, err = fetcher.requestBeaconBlocksByRange(context.Background(), peers[0], root, 1, 1, blockBatchLimit)
		if err != nil {
			t.Errorf("error: %v", err)
		}

		// Test request fail over (error).
		err = fetcher.p2p.Disconnect(peers[1])
		ctx, cancel = context.WithTimeout(context.Background(), time.Second*1)
		defer cancel()
		blocks, err = fetcher.requestBeaconBlocksByRange(ctx, peers[1], root, 1, 1, blockBatchLimit)
		testutil.AssertLogsContain(t, hook, "Request failed, trying to forward request to another peer")
		if err == nil || err.Error() != "context deadline exceeded" {
			t.Errorf("expected context closed error, got: %v", err)
		}
	}

	// Test context cancellation.
	ctx, cancel = context.WithCancel(context.Background())
	cancel()
	blocks, err = fetcher.requestBeaconBlocksByRange(ctx, peers[0], root, 1, 1, blockBatchLimit)
	if err == nil || err.Error() != "context canceled" {
		t.Errorf("expected context closed error, got: %v", err)
	}
}

func TestBlocksFetcher_selectFailOverPeer(t *testing.T) {
	type args struct {
		excludedPID peer.ID
		peers       []peer.ID
	}
	fetcher := newBlocksFetcher(context.Background(), &blocksFetcherConfig{})
	tests := []struct {
		name    string
		args    args
		want    peer.ID
		wantErr error
	}{
		{
			name: "No peers provided",
			args: args{
				excludedPID: "abc",
				peers:       []peer.ID{},
			},
			want:    "",
			wantErr: errNoPeersAvailable,
		},
		{
			name: "Single peer which needs to be excluded",
			args: args{
				excludedPID: "abc",
				peers: []peer.ID{
					"abc",
				},
			},
			want:    "",
			wantErr: errNoPeersAvailable,
		},
		{
			name: "Single peer available",
			args: args{
				excludedPID: "abc",
				peers: []peer.ID{
					"cde",
				},
			},
			want:    "cde",
			wantErr: nil,
		},
		{
			name: "Two peers available",
			args: args{
				excludedPID: "abc",
				peers: []peer.ID{
					"abc", "cde",
				},
			},
			want:    "cde",
			wantErr: nil,
		},
		{
			name: "Multiple peers available",
			args: args{
				excludedPID: "abc",
				peers: []peer.ID{
					"abc", "cde", "cde", "cde",
				},
			},
			want:    "cde",
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := fetcher.selectFailOverPeer(tt.args.excludedPID, tt.args.peers)
			if err != nil && err != tt.wantErr {
				t.Errorf("selectFailOverPeer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("selectFailOverPeer() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBlocksFetcher_nonSkippedSlotAfter(t *testing.T) {
	peersGen := func(size int) []*peerData {
		blocks := append(makeSequence(1, 64), makeSequence(500, 640)...)
		blocks = append(blocks, makeSequence(51200, 51264)...)
		blocks = append(blocks, 55000)
		blocks = append(blocks, makeSequence(57000, 57256)...)
		var peers []*peerData
		for i := 0; i < size; i++ {
			peers = append(peers, &peerData{
				blocks:         blocks,
				finalizedEpoch: 1800,
				headSlot:       57000,
			})
		}
		return peers
	}
	chainConfig := struct {
		peers []*peerData
	}{
		peers: peersGen(5),
	}

	mc, p2p, _ := initializeTestServices(t, []uint64{}, chainConfig.peers)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fetcher := newBlocksFetcher(
		ctx,
		&blocksFetcherConfig{
			headFetcher: mc,
			p2p:         p2p,
		},
	)
	fetcher.rateLimiter = leakybucket.NewCollector(6400, 6400, false)
	seekSlots := map[uint64]uint64{
		0:     1,
		10:    11,
		31:    32,
		32:    33,
		63:    64,
		64:    500,
		160:   500,
		352:   500,
		480:   500,
		512:   513,
		639:   640,
		640:   51200,
		6640:  51200,
		51200: 51201,
	}
	for seekSlot, expectedSlot := range seekSlots {
		t.Run(fmt.Sprintf("range: %d (%d-%d)", expectedSlot-seekSlot, seekSlot, expectedSlot), func(t *testing.T) {
			slot, err := fetcher.nonSkippedSlotAfter(ctx, seekSlot)
			if err != nil {
				t.Error(err)
			}
			if slot != expectedSlot {
				t.Errorf("unexpected slot, want: %v, got: %v", expectedSlot, slot)
			}
		})
	}

	t.Run("test isolated non-skipped slot", func(t *testing.T) {
		seekSlot := uint64(51264)
		expectedSlot := uint64(55000)
		found := false
		var i int
		for i = 0; i < 100; i++ {
			slot, err := fetcher.nonSkippedSlotAfter(ctx, seekSlot)
			if err != nil {
				t.Error(err)
			}
			if slot == expectedSlot {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Isolated non-skipped slot not found in %d iterations: %v", i, expectedSlot)
		} else {
			t.Logf("Isolated non-skipped slot found in %d iterations", i)
		}
	})
}

func TestBlocksFetcher_filterPeers(t *testing.T) {
	if !featureconfig.Get().EnableInitSyncWeightedRoundRobin {
		t.Skip("Test is run only when EnableInitSyncWeightedRoundRobin = true")
	}
	type weightedPeer struct {
		peer.ID
		usedCapacity int64
	}
	type args struct {
		peers           []weightedPeer
		peersPercentage float64
	}
	fetcher := newBlocksFetcher(context.Background(), &blocksFetcherConfig{})
	tests := []struct {
		name string
		args args
		want []peer.ID
	}{
		{
			name: "no peers available",
			args: args{
				peers:           []weightedPeer{},
				peersPercentage: 1.0,
			},
			want: []peer.ID{},
		},
		{
			name: "single peer",
			args: args{
				peers: []weightedPeer{
					{"abc", 10},
				},
				peersPercentage: 1.0,
			},
			want: []peer.ID{"abc"},
		},
		{
			name: "multiple peers same capacity",
			args: args{
				peers: []weightedPeer{
					{"abc", 10},
					{"def", 10},
					{"xyz", 10},
				},
				peersPercentage: 1.0,
			},
			want: []peer.ID{"abc", "def", "xyz"},
		},
		{
			name: "multiple peers different capacity",
			args: args{
				peers: []weightedPeer{
					{"abc", 20},
					{"def", 15},
					{"ghi", 10},
					{"jkl", 90},
					{"xyz", 20},
				},
				peersPercentage: 1.0,
			},
			want: []peer.ID{"ghi", "def", "abc", "xyz", "jkl"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Non-leaking bucket, with initial capacity of 100.
			fetcher.rateLimiter = leakybucket.NewCollector(0.000001, 100, false)
			pids := make([]peer.ID, 0)
			for _, pid := range tt.args.peers {
				pids = append(pids, pid.ID)
				fetcher.rateLimiter.Add(pid.ID.String(), pid.usedCapacity)
			}
			got, err := fetcher.filterPeers(pids, tt.args.peersPercentage)
			if err != nil {
				t.Fatal(err)
			}
			// Re-arrange peers with the same remaining capacity, deterministically .
			// They are deliberately shuffled - so that on the same capacity any of
			// such peers can be selected. That's why they are sorted here.
			sort.SliceStable(got, func(i, j int) bool {
				cap1 := fetcher.rateLimiter.Remaining(pids[i].String())
				cap2 := fetcher.rateLimiter.Remaining(pids[j].String())
				if cap1 == cap2 {
					return pids[i].String() < pids[j].String()
				}
				return i < j
			})
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("filterPeers() got = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestBlocksFetcher_RequestBlocksRateLimitingLocks(t *testing.T) {
	p1 := p2pt.NewTestP2P(t)
	p2 := p2pt.NewTestP2P(t)
	p3 := p2pt.NewTestP2P(t)
	p1.Connect(p2)
	p1.Connect(p3)
	if len(p1.BHost.Network().Peers()) != 2 {
		t.Fatal("Expected peers to be connected")
	}
	req := &p2ppb.BeaconBlocksByRangeRequest{
		StartSlot: 100,
		Step:      1,
		Count:     64,
	}

	topic := p2pm.RPCBlocksByRangeTopic
	protocol := core.ProtocolID(topic + p2.Encoding().ProtocolSuffix())
	streamHandlerFn := func(stream network.Stream) {
		if err := stream.Close(); err != nil {
			t.Error(err)
		}
	}
	p2.BHost.SetStreamHandler(protocol, streamHandlerFn)
	p3.BHost.SetStreamHandler(protocol, streamHandlerFn)

	burstFactor := uint64(flags.Get().BlockBatchLimitBurstFactor)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	fetcher := newBlocksFetcher(ctx, &blocksFetcherConfig{p2p: p1})
	fetcher.rateLimiter = leakybucket.NewCollector(float64(req.Count), int64(req.Count*burstFactor), false)

	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func() {
		// Exhaust available rate for p2, so that rate limiting is triggered.
		for i := uint64(0); i <= burstFactor; i++ {
			if i == burstFactor {
				// The next request will trigger rate limiting for p2. Now, allow concurrent
				// p3 data request (p3 shouldn't be rate limited).
				time.AfterFunc(1*time.Second, func() {
					wg.Done()
				})
			}
			_, err := fetcher.requestBlocks(ctx, req, p2.PeerID())
			if err != nil && err != errFetcherCtxIsDone {
				t.Error(err)
			}
		}
	}()

	// Wait until p2 exhausts its rate and is spinning on rate limiting timer.
	wg.Wait()

	// The next request should NOT trigger rate limiting as rate is exhausted for p2, not p3.
	ch := make(chan struct{}, 1)
	go func() {
		_, err := fetcher.requestBlocks(ctx, req, p3.PeerID())
		if err != nil {
			t.Error(err)
		}
		ch <- struct{}{}
	}()
	timer := time.NewTimer(2 * time.Second)
	select {
	case <-timer.C:
		t.Error("p3 takes too long to respond: lock contention")
	case <-ch:
		// p3 responded w/o waiting for rate limiter's lock (on which p2 spins).
	}
}

func TestBlocksFetcher_removeStalePeerLocks(t *testing.T) {
	type peerData struct {
		peerID   peer.ID
		accessed time.Time
	}
	tests := []struct {
		name     string
		age      time.Duration
		peersIn  []peerData
		peersOut []peerData
	}{
		{
			name:     "empty map",
			age:      peerLockMaxAge,
			peersIn:  []peerData{},
			peersOut: []peerData{},
		},
		{
			name: "no stale peer locks",
			age:  peerLockMaxAge,
			peersIn: []peerData{
				{
					peerID:   "abc",
					accessed: roughtime.Now(),
				},
				{
					peerID:   "def",
					accessed: roughtime.Now(),
				},
				{
					peerID:   "ghi",
					accessed: roughtime.Now(),
				},
			},
			peersOut: []peerData{
				{
					peerID:   "abc",
					accessed: roughtime.Now(),
				},
				{
					peerID:   "def",
					accessed: roughtime.Now(),
				},
				{
					peerID:   "ghi",
					accessed: roughtime.Now(),
				},
			},
		},
		{
			name: "one stale peer lock",
			age:  peerLockMaxAge,
			peersIn: []peerData{
				{
					peerID:   "abc",
					accessed: roughtime.Now(),
				},
				{
					peerID:   "def",
					accessed: roughtime.Now().Add(-peerLockMaxAge),
				},
				{
					peerID:   "ghi",
					accessed: roughtime.Now(),
				},
			},
			peersOut: []peerData{
				{
					peerID:   "abc",
					accessed: roughtime.Now(),
				},
				{
					peerID:   "ghi",
					accessed: roughtime.Now(),
				},
			},
		},
		{
			name: "all peer locks are stale",
			age:  peerLockMaxAge,
			peersIn: []peerData{
				{
					peerID:   "abc",
					accessed: roughtime.Now().Add(-peerLockMaxAge),
				},
				{
					peerID:   "def",
					accessed: roughtime.Now().Add(-peerLockMaxAge),
				},
				{
					peerID:   "ghi",
					accessed: roughtime.Now().Add(-peerLockMaxAge),
				},
			},
			peersOut: []peerData{},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	fetcher := newBlocksFetcher(ctx, &blocksFetcherConfig{})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fetcher.peerLocks = make(map[peer.ID]*peerLock, len(tt.peersIn))
			for _, data := range tt.peersIn {
				fetcher.peerLocks[data.peerID] = &peerLock{
					Mutex:    sync.Mutex{},
					accessed: data.accessed,
				}
			}

			fetcher.removeStalePeerLocks(tt.age)

			var peersOut1, peersOut2 []peer.ID
			for _, data := range tt.peersOut {
				peersOut1 = append(peersOut1, data.peerID)
			}
			for peerID := range fetcher.peerLocks {
				peersOut2 = append(peersOut2, peerID)
			}
			sort.SliceStable(peersOut1, func(i, j int) bool {
				return peersOut1[i].String() < peersOut1[j].String()
			})
			sort.SliceStable(peersOut2, func(i, j int) bool {
				return peersOut2[i].String() < peersOut2[j].String()
			})
			if !reflect.DeepEqual(peersOut1, peersOut2) {
				t.Errorf("unexpected peers map, want: %#v, got: %#v", peersOut1, peersOut2)
			}
		})
	}
}
