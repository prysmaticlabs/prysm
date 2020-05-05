package initialsync

import (
	"context"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	dbtest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	p2pt "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	p2ppb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/sliceutil"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/sirupsen/logrus"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestBlocksFetcherInitStartStop(t *testing.T) {
	mc, p2p, _ := initializeTestServices(t, []uint64{}, []*peerData{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	fetcher := newBlocksFetcher(
		ctx,
		&blocksFetcherConfig{
			headFetcher: mc,
			p2p:         p2p,
		})

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

func TestBlocksFetcherRoundRobin(t *testing.T) {
	tests := []struct {
		name               string
		expectedBlockSlots []uint64
		peers              []*peerData
		requests           []*fetchRequestParams
	}{
		{
			name:               "Single peer with all blocks",
			expectedBlockSlots: makeSequence(1, 3*blockBatchSize),
			peers: []*peerData{
				{
					blocks:         makeSequence(1, 131),
					finalizedEpoch: 3,
					headSlot:       131,
				},
			},
			requests: []*fetchRequestParams{
				{
					start: 1,
					count: blockBatchSize,
				},
				{
					start: blockBatchSize + 1,
					count: blockBatchSize,
				},
				{
					start: 2*blockBatchSize + 1,
					count: blockBatchSize,
				},
			},
		},
		{
			name:               "Single peer with all blocks (many small requests)",
			expectedBlockSlots: makeSequence(1, 80),
			peers: []*peerData{
				{
					blocks:         makeSequence(1, 131),
					finalizedEpoch: 3,
					headSlot:       131,
				},
			},
			requests: []*fetchRequestParams{
				{
					start: 1,
					count: blockBatchSize / 2,
				},
				{
					start: blockBatchSize/2 + 1,
					count: blockBatchSize / 2,
				},
				{
					start: 2*blockBatchSize/2 + 1,
					count: blockBatchSize / 2,
				},
				{
					start: 3*blockBatchSize/2 + 1,
					count: blockBatchSize / 2,
				},
				{
					start: 4*blockBatchSize/2 + 1,
					count: blockBatchSize / 2,
				},
			},
		},
		{
			name:               "Multiple peers with all blocks",
			expectedBlockSlots: makeSequence(1, 96), // up to 4th epoch
			peers: []*peerData{
				{
					blocks:         makeSequence(1, 131),
					finalizedEpoch: 3,
					headSlot:       131,
				},
				{
					blocks:         makeSequence(1, 131),
					finalizedEpoch: 3,
					headSlot:       131,
				},
				{
					blocks:         makeSequence(1, 131),
					finalizedEpoch: 3,
					headSlot:       131,
				},
				{
					blocks:         makeSequence(1, 131),
					finalizedEpoch: 3,
					headSlot:       131,
				},
				{
					blocks:         makeSequence(1, 131),
					finalizedEpoch: 3,
					headSlot:       131,
				},
			},
			requests: []*fetchRequestParams{
				{
					start: 1,
					count: blockBatchSize,
				},
				{
					start: blockBatchSize + 1,
					count: blockBatchSize,
				},
				{
					start: 2*blockBatchSize + 1,
					count: blockBatchSize,
				},
			},
		},
		{
			name: "Multiple peers with skipped slots",
			// finalizedEpoch(18).slot = 608
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
					count: blockBatchSize,
				},
				{
					start: blockBatchSize + 1,
					count: blockBatchSize,
				},
				{
					start: 2*blockBatchSize + 1,
					count: blockBatchSize,
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
			expectedBlockSlots: makeSequence(1, 2*blockBatchSize),
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
					count: blockBatchSize,
				},
				{
					start: blockBatchSize + 1,
					count: blockBatchSize,
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
			if missing := sliceutil.NotUint64(sliceutil.IntersectionUint64(tt.expectedBlockSlots, receivedBlockSlots), tt.expectedBlockSlots); len(missing) > 0 {
				t.Errorf("Missing blocks at slots %v", missing)
			}
		})
	}
}

func TestBlocksFetcherScheduleRequest(t *testing.T) {
	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		fetcher := newBlocksFetcher(ctx, &blocksFetcherConfig{
			headFetcher: nil,
			p2p:         nil,
		})
		cancel()
		if err := fetcher.scheduleRequest(ctx, 1, blockBatchSize); err == nil {
			t.Errorf("expected error: %v", errFetcherCtxIsDone)
		}
	})
}

func TestBlocksFetcherHandleRequest(t *testing.T) {
	chainConfig := struct {
		expectedBlockSlots []uint64
		peers              []*peerData
	}{
		expectedBlockSlots: makeSequence(1, blockBatchSize),
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
		response := fetcher.handleRequest(ctx, 1, blockBatchSize)
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

		requestCtx, _ := context.WithTimeout(context.Background(), 2*time.Second)
		go func() {
			response := fetcher.handleRequest(requestCtx, 1 /* start */, blockBatchSize /* count */)
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
		if len(blocks) != blockBatchSize {
			t.Errorf("incorrect number of blocks returned, expected: %v, got: %v", blockBatchSize, len(blocks))
		}

		var receivedBlockSlots []uint64
		for _, blk := range blocks {
			receivedBlockSlots = append(receivedBlockSlots, blk.Block.Slot)
		}
		if missing := sliceutil.NotUint64(sliceutil.IntersectionUint64(chainConfig.expectedBlockSlots, receivedBlockSlots), chainConfig.expectedBlockSlots); len(missing) > 0 {
			t.Errorf("Missing blocks at slots %v", missing)
		}
	})
}

func TestBlocksFetcherRequestBeaconBlocksByRangeRequest(t *testing.T) {
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

	blocks, err := fetcher.requestBeaconBlocksByRange(context.Background(), peers[0], root, 1, 1, blockBatchSize)
	if err != nil {
		t.Errorf("error: %v", err)
	}
	if len(blocks) != blockBatchSize {
		t.Errorf("incorrect number of blocks returned, expected: %v, got: %v", blockBatchSize, len(blocks))
	}

	// Test request fail over (success).
	err = fetcher.p2p.Disconnect(peers[0])
	if err != nil {
		t.Error(err)
	}
	blocks, err = fetcher.requestBeaconBlocksByRange(context.Background(), peers[0], root, 1, 1, blockBatchSize)
	if err != nil {
		t.Errorf("error: %v", err)
	}

	// Test request fail over (error).
	err = fetcher.p2p.Disconnect(peers[1])
	ctx, _ = context.WithTimeout(context.Background(), time.Second*1)
	blocks, err = fetcher.requestBeaconBlocksByRange(ctx, peers[1], root, 1, 1, blockBatchSize)
	testutil.AssertLogsContain(t, hook, "Request failed, trying to forward request to another peer")
	if err == nil || err.Error() != "context deadline exceeded" {
		t.Errorf("expected context closed error, got: %v", err)
	}

	// Test context cancellation.
	ctx, cancel = context.WithCancel(context.Background())
	cancel()
	blocks, err = fetcher.requestBeaconBlocksByRange(ctx, peers[0], root, 1, 1, blockBatchSize)
	if err == nil || err.Error() != "context canceled" {
		t.Errorf("expected context closed error, got: %v", err)
	}
}

func TestBlocksFetcherSelectFailOverPeer(t *testing.T) {
	type args struct {
		excludedPID peer.ID
		peers       []peer.ID
	}
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
			got, err := selectFailOverPeer(tt.args.excludedPID, tt.args.peers)
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

func TestBlocksFetcherNonSkippedSlotAfter(t *testing.T) {
	chainConfig := struct {
		expectedBlockSlots []uint64
		peers              []*peerData
	}{
		expectedBlockSlots: makeSequence(1, 320),
		peers: []*peerData{
			{
				blocks:         append(makeSequence(1, 64), makeSequence(500, 640)...),
				finalizedEpoch: 18,
				headSlot:       320,
			},
			{
				blocks:         append(makeSequence(1, 64), makeSequence(500, 640)...),
				finalizedEpoch: 18,
				headSlot:       320,
			},
		},
	}

	mc, p2p, _ := initializeTestServices(t, chainConfig.expectedBlockSlots, chainConfig.peers)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fetcher := newBlocksFetcher(
		ctx,
		&blocksFetcherConfig{
			headFetcher: mc,
			p2p:         p2p,
		})

	seekSlots := map[uint64]uint64{
		0:  1,
		10: 11,
		32: 33,
		63: 64,
		64: 500,
	}
	for seekSlot, expectedSlot := range seekSlots {
		slot, err := fetcher.nonSkippedSlotAfter(ctx, seekSlot)
		if err != nil {
			t.Error(err)
		}
		if slot != expectedSlot {
			t.Errorf("unexpected slot, want: %v, got: %v", expectedSlot, slot)
		}
	}
}

func initializeTestServices(t *testing.T, blocks []uint64, peers []*peerData) (*mock.ChainService, *p2pt.TestP2P, db.Database) {
	cache.initializeRootCache(blocks, t)
	beaconDB := dbtest.SetupDB(t)

	p := p2pt.NewTestP2P(t)
	connectPeers(t, p, peers, p.Peers())
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

	return &mock.ChainService{
		State: st,
		Root:  genesisRoot[:],
		DB:    beaconDB,
	}, p, beaconDB
}
