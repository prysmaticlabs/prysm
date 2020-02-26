package initialsync

import (
	"context"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/kevinms/leakybucket-go"
	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	dbtest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	p2pt "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	p2ppb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/sliceutil"
	"github.com/sirupsen/logrus"
)

func TestBlocksFetcher(t *testing.T) {
	tests := []struct {
		name               string
		expectedBlockSlots []uint64
		peers              []*peerData
		requests           []*fetchRequestParams
	}{
		{
			name:               "Single peer with all blocks",
			expectedBlockSlots: makeSequence(1, 128), // up to 4th epoch
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
			expectedBlockSlots: makeSequence(1, 128), // up to 4th epoch
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
			expectedBlockSlots: makeSequence(1, 128), // up to 4th epoch
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
					start: 400,
					count: 150,
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
			initializeRootCache(tt.expectedBlockSlots, t)

			beaconDB := dbtest.SetupDB(t)

			p := p2pt.NewTestP2P(t)
			connectPeers(t, p, tt.peers, p.Peers())
			genesisRoot := rootCache[0]

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
			fetcher := newBlocksFetcher(&blocksFetcherConfig{
				ctx:         ctx,
				headFetcher: mc,
				p2p:         p,
				rateLimiter: leakybucket.NewCollector(allowedBlocksPerSecond, allowedBlocksPerSecond, false /* deleteEmptyBuckets */),
			})

			fetcher.start()

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
					case <-time.After(5 * time.Second): // TODO(4815): Temporary safeguard, remove
						t.Fatal("timeout")
					case resp, ok := <-fetcher.iter():
						if !ok { // channel closed, aggregate
							return unionRespBlocks, nil
						}

						if resp.err != nil {
							log.WithError(resp.err).Debug("Block fetcher returned error")
						} else {
							unionRespBlocks = append(unionRespBlocks, resp.blocks...)
							if len(resp.blocks) == 0 {
								log.WithFields(logrus.Fields{
									"params": resp.params,
								}).Debug("Received empty slot")
							}
						}

						wg.Done()
					}
				}
			}

			maxExpectedBlocks := uint64(0)
			for _, requestParams := range tt.requests {
				fetcher.scheduleRequest(requestParams)
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

			dbtest.TeardownDB(t, beaconDB)
		})
	}
}
