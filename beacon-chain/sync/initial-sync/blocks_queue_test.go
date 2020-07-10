package initialsync

import (
	"context"
	"fmt"
	"testing"

	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/flags"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/sliceutil"
)

func TestBlocksQueueInitStartStop(t *testing.T) {
	blockBatchLimit := uint64(flags.Get().BlockBatchLimit)
	mc, p2p, _ := initializeTestServices(t, []uint64{}, []*peerData{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	fetcher := newBlocksFetcher(ctx, &blocksFetcherConfig{
		headFetcher: mc,
		p2p:         p2p,
	})

	t.Run("stop without start", func(t *testing.T) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		queue := newBlocksQueue(ctx, &blocksQueueConfig{
			headFetcher:         mc,
			highestExpectedSlot: blockBatchLimit,
		})

		if err := queue.stop(); err == nil {
			t.Errorf("expected error: %v", errQueueTakesTooLongToStop)
		}
	})

	t.Run("use default fetcher", func(t *testing.T) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		queue := newBlocksQueue(ctx, &blocksQueueConfig{
			headFetcher:         mc,
			highestExpectedSlot: blockBatchLimit,
		})
		if err := queue.start(); err != nil {
			t.Error(err)
		}
	})

	t.Run("stop timeout", func(t *testing.T) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		queue := newBlocksQueue(ctx, &blocksQueueConfig{
			headFetcher:         mc,
			highestExpectedSlot: blockBatchLimit,
		})
		if err := queue.start(); err != nil {
			t.Error(err)
		}
		if err := queue.stop(); err == nil {
			t.Errorf("expected error: %v", errQueueTakesTooLongToStop)
		}
	})

	t.Run("check for leaked goroutines", func(t *testing.T) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		queue := newBlocksQueue(ctx, &blocksQueueConfig{
			blocksFetcher:       fetcher,
			headFetcher:         mc,
			highestExpectedSlot: blockBatchLimit,
		})

		if err := queue.start(); err != nil {
			t.Error(err)
		}
		// Blocks up until all resources are reclaimed (or timeout is called)
		if err := queue.stop(); err != nil {
			t.Error(err)
		}
		select {
		case <-queue.fetchedBlocks:
		default:
			t.Error("queue.fetchedBlocks channel is leaked")
		}
		select {
		case <-fetcher.fetchResponses:
		default:
			t.Error("fetcher.fetchResponses channel is leaked")
		}
	})

	t.Run("re-starting of stopped queue", func(t *testing.T) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		queue := newBlocksQueue(ctx, &blocksQueueConfig{
			blocksFetcher:       fetcher,
			headFetcher:         mc,
			highestExpectedSlot: blockBatchLimit,
		})
		if err := queue.start(); err != nil {
			t.Error(err)
		}
		if err := queue.stop(); err != nil {
			t.Error(err)
		}
		if err := queue.start(); err == nil {
			t.Errorf("expected error not returned: %v", errQueueCtxIsDone)
		}
	})

	t.Run("multiple stopping attempts", func(t *testing.T) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		queue := newBlocksQueue(ctx, &blocksQueueConfig{
			blocksFetcher:       fetcher,
			headFetcher:         mc,
			highestExpectedSlot: blockBatchLimit,
		})
		if err := queue.start(); err != nil {
			t.Error(err)
		}

		if err := queue.stop(); err != nil {
			t.Error(err)
		}
		if err := queue.stop(); err != nil {
			t.Error(err)
		}
	})

	t.Run("cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		queue := newBlocksQueue(ctx, &blocksQueueConfig{
			blocksFetcher:       fetcher,
			headFetcher:         mc,
			highestExpectedSlot: blockBatchLimit,
		})
		if err := queue.start(); err != nil {
			t.Error(err)
		}

		cancel()
		if err := queue.stop(); err != nil {
			t.Error(err)
		}
	})
}

func TestBlocksQueueLoop(t *testing.T) {
	tests := []struct {
		name                string
		highestExpectedSlot uint64
		expectedBlockSlots  []uint64
		peers               []*peerData
	}{
		{
			name:                "Single peer with all blocks",
			highestExpectedSlot: 251, // will be auto-fixed to 256 (to 8th epoch), by queue
			expectedBlockSlots:  makeSequence(1, 256),
			peers: []*peerData{
				{
					blocks:         makeSequence(1, 320),
					finalizedEpoch: 8,
					headSlot:       320,
				},
			},
		},
		{
			name:                "Multiple peers with all blocks",
			highestExpectedSlot: 256,
			expectedBlockSlots:  makeSequence(1, 256),
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
				{
					blocks:         makeSequence(1, 320),
					finalizedEpoch: 8,
					headSlot:       320,
				},
			},
		},
		{
			name:                "Multiple peers with skipped slots",
			highestExpectedSlot: 576,
			expectedBlockSlots:  append(makeSequence(1, 64), makeSequence(500, 576)...), // up to 18th epoch
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
		},
		{
			name:                "Multiple peers with failures",
			highestExpectedSlot: 128,
			expectedBlockSlots:  makeSequence(1, 256),
			peers: []*peerData{
				{
					blocks:         makeSequence(1, 320),
					finalizedEpoch: 8,
					headSlot:       320,
					failureSlots:   makeSequence(32*3+1, 32*3+32),
				},
				{
					blocks:         makeSequence(1, 320),
					finalizedEpoch: 8,
					headSlot:       320,
					failureSlots:   makeSequence(1, 32*3),
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
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc, p2p, beaconDB := initializeTestServices(t, tt.expectedBlockSlots, tt.peers)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			fetcher := newBlocksFetcher(ctx, &blocksFetcherConfig{
				headFetcher: mc,
				p2p:         p2p,
			})
			queue := newBlocksQueue(ctx, &blocksQueueConfig{
				blocksFetcher:       fetcher,
				headFetcher:         mc,
				highestExpectedSlot: tt.highestExpectedSlot,
			})
			if err := queue.start(); err != nil {
				t.Error(err)
			}
			processBlock := func(block *eth.SignedBeaconBlock) error {
				if !beaconDB.HasBlock(ctx, bytesutil.ToBytes32(block.Block.ParentRoot)) {
					return fmt.Errorf("beacon node doesn't have a block in db with root %#x", block.Block.ParentRoot)
				}
				root, err := stateutil.BlockRoot(block.Block)
				if err != nil {
					return err
				}
				if err := mc.ReceiveBlock(ctx, block, root); err != nil {
					return err
				}

				return nil
			}

			var blocks []*eth.SignedBeaconBlock
			for fetchedBlocks := range queue.fetchedBlocks {
				for _, block := range fetchedBlocks {
					if err := processBlock(block); err != nil {
						continue
					}
					blocks = append(blocks, block)
				}
			}

			if err := queue.stop(); err != nil {
				t.Error(err)
			}

			if queue.headFetcher.HeadSlot() < tt.highestExpectedSlot {
				t.Errorf("Not enough slots synced, want: %v, got: %v",
					len(tt.expectedBlockSlots), queue.headFetcher.HeadSlot())
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
