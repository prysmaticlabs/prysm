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
	"github.com/sirupsen/logrus"
)

func TestBlocksFetcher(t *testing.T) {
	tests := []struct {
		name               string
		currentSlot        uint64
		expectedBlockSlots []uint64
		peers              []*peerData
		requestParams      []*fetchRequestParams
	}{
		//{
		//	name:               "Single peer with all blocks",
		//	currentSlot:        131,
		//	expectedBlockSlots: makeSequence(1, 131),
		//	peers: []*peerData{
		//		{
		//			blocks:         makeSequence(1, 131),
		//			finalizedEpoch: 1,
		//			headSlot:       131,
		//		},
		//	},
		//},
		{
			name:               "Multiple peers with all blocks",
			currentSlot:        131,
			expectedBlockSlots: makeSequence(1, 131),
			peers: []*peerData{
				{
					blocks:         makeSequence(1, 131),
					finalizedEpoch: 2,
					headSlot:       131,
				},
				{
					blocks:         makeSequence(1, 131),
					finalizedEpoch: 2,
					headSlot:       131,
				},
				{
					blocks:         makeSequence(1, 131),
					finalizedEpoch: 1,
					headSlot:       131,
				},
				{
					blocks:         makeSequence(1, 131),
					finalizedEpoch: 1,
					headSlot:       131,
				},
			},
			requestParams: []*fetchRequestParams{
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
		//{
		//	name:               "Multiple peers with many skipped slots",
		//	currentSlot:        640, // 10 epochs
		//	expectedBlockSlots: append(makeSequence(1, 64), makeSequence(500, 640)...),
		//	peers: []*peerData{
		//		{
		//			blocks:         append(makeSequence(1, 64), makeSequence(500, 640)...),
		//			finalizedEpoch: 18,
		//			headSlot:       640,
		//		},
		//		{
		//			blocks:         append(makeSequence(1, 64), makeSequence(500, 640)...),
		//			finalizedEpoch: 18,
		//			headSlot:       640,
		//		},
		//		{
		//			blocks:         append(makeSequence(1, 64), makeSequence(500, 640)...),
		//			finalizedEpoch: 18,
		//			headSlot:       640,
		//		},
		//	},
		//},
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
			} // no-op mock

			ctx, cancel := context.WithCancel(context.Background())
			fetcher := newBlocksFetcher(&blocksFetcherConfig{
				ctx:         ctx,
				chain:       mc,
				p2p:         p,
				rateLimiter: leakybucket.NewCollector(allowedBlocksPerSecond, allowedBlocksPerSecond, false /* deleteEmptyBuckets */),
			})

			fetcher.start()

			var wg sync.WaitGroup
			wg.Add(len(tt.requestParams)) // how many block requests we are going to make
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
					case <-time.After(5 * time.Second): // TODO: temporary safeguard, remove
						t.Fatal("timeout")
					case resp, ok := <-fetcher.iter():
						if !ok { // channel closed, aggregate
							return unionRespBlocks, nil
						}

						if resp.err != nil {
							log.WithError(resp.err).Debug("Block fetcher returned error")
						} else {
							unionRespBlocks = append(unionRespBlocks, resp.blocks...)
						}

						wg.Done()
					}
				}
			}

			for _, requestParams := range tt.requestParams {
				fetcher.scheduleRequest(requestParams)
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

			dbtest.TeardownDB(t, beaconDB)
		})
	}
}
