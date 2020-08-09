package initialsync

import (
	"context"
	"fmt"
	"testing"

	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"

	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	dbtest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/flags"
	p2pt "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/sliceutil"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestConstants(t *testing.T) {
	if params.BeaconConfig().MaxPeersToSync*flags.Get().BlockBatchLimit > 1000 {
		t.Fatal("rpc rejects requests over 1000 range slots")
	}
}

func TestService_roundRobinSync(t *testing.T) {
	tests := []struct {
		name               string
		currentSlot        uint64
		expectedBlockSlots []uint64
		peers              []*peerData
	}{
		{
			name:               "Single peer with no finalized blocks",
			currentSlot:        2,
			expectedBlockSlots: makeSequence(1, 2),
			peers: []*peerData{
				{
					blocks:         makeSequence(1, 2),
					finalizedEpoch: 0,
					headSlot:       2,
				},
			},
		},
		{
			name:               "Multiple peers with no finalized blocks",
			currentSlot:        2,
			expectedBlockSlots: makeSequence(1, 2),
			peers: []*peerData{
				{
					blocks:         makeSequence(1, 2),
					finalizedEpoch: 0,
					headSlot:       2,
				},
				{
					blocks:         makeSequence(1, 2),
					finalizedEpoch: 0,
					headSlot:       2,
				},
				{
					blocks:         makeSequence(1, 2),
					finalizedEpoch: 0,
					headSlot:       2,
				},
			},
		},
		{
			name:               "Single peer with all blocks",
			currentSlot:        131,
			expectedBlockSlots: makeSequence(1, 131),
			peers: []*peerData{
				{
					blocks:         makeSequence(1, 131),
					finalizedEpoch: 1,
					headSlot:       131,
				},
			},
		},
		{
			name:               "Multiple peers with all blocks",
			currentSlot:        131,
			expectedBlockSlots: makeSequence(1, 131),
			peers: []*peerData{
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
		},
		{
			name:               "Multiple peers with failures",
			currentSlot:        320, // 10 epochs
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
		},
		{
			name:               "Multiple peers with many skipped slots",
			currentSlot:        1280,
			expectedBlockSlots: append(makeSequence(1, 64), makeSequence(1000, 1280)...),
			peers: []*peerData{
				{
					blocks:         append(makeSequence(1, 64), makeSequence(1000, 1280)...),
					finalizedEpoch: 36,
					headSlot:       1280,
				},
				{
					blocks:         append(makeSequence(1, 64), makeSequence(1000, 1280)...),
					finalizedEpoch: 36,
					headSlot:       1280,
				},
				{
					blocks:         append(makeSequence(1, 64), makeSequence(1000, 1280)...),
					finalizedEpoch: 36,
					headSlot:       1280,
				},
			},
		},
		{
			name:               "Multiple peers with multiple failures",
			currentSlot:        320, // 10 epochs
			expectedBlockSlots: makeSequence(1, 320),
			peers: []*peerData{
				{
					blocks:         makeSequence(1, 320),
					finalizedEpoch: 9,
					headSlot:       320,
				},
				{
					blocks:         makeSequence(1, 320),
					finalizedEpoch: 9,
					headSlot:       320,
					failureSlots:   makeSequence(1, 320),
				},
				{
					blocks:         makeSequence(1, 320),
					finalizedEpoch: 9,
					headSlot:       320,
					failureSlots:   makeSequence(1, 320),
				},
				{
					blocks:         makeSequence(1, 320),
					finalizedEpoch: 9,
					headSlot:       320,
					failureSlots:   makeSequence(1, 320),
				},
			},
		},
		{
			name:               "Multiple peers with different finalized epoch",
			currentSlot:        320, // 10 epochs
			expectedBlockSlots: makeSequence(1, 320),
			peers: []*peerData{
				{
					blocks:         makeSequence(1, 320),
					finalizedEpoch: 4,
					headSlot:       320,
				},
				{
					blocks:         makeSequence(1, 256),
					finalizedEpoch: 3,
					headSlot:       256,
				},
				{
					blocks:         makeSequence(1, 256),
					finalizedEpoch: 3,
					headSlot:       256,
				},
				{
					blocks:         makeSequence(1, 192),
					finalizedEpoch: 2,
					headSlot:       192,
				},
			},
		},
		{
			name:               "Multiple peers with missing parent blocks",
			currentSlot:        160, // 5 epochs
			expectedBlockSlots: makeSequence(1, 160),
			peers: []*peerData{
				{
					blocks:         makeSequence(1, 160),
					finalizedEpoch: 4,
					headSlot:       160,
				},
				{
					blocks:         append(makeSequence(1, 6), makeSequence(161, 165)...),
					finalizedEpoch: 4,
					headSlot:       160,
					forkedPeer:     true,
				},
				{
					blocks:         makeSequence(1, 160),
					finalizedEpoch: 4,
					headSlot:       160,
				},
				{
					blocks:         makeSequence(1, 160),
					finalizedEpoch: 4,
					headSlot:       160,
				},
				{
					blocks:         makeSequence(1, 160),
					finalizedEpoch: 4,
					headSlot:       160,
				},
				{
					blocks:         makeSequence(1, 160),
					finalizedEpoch: 4,
					headSlot:       160,
				},
				{
					blocks:         makeSequence(1, 160),
					finalizedEpoch: 4,
					headSlot:       160,
				},
				{
					blocks:         makeSequence(1, 160),
					finalizedEpoch: 4,
					headSlot:       160,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache.initializeRootCache(tt.expectedBlockSlots, t)

			p := p2pt.NewTestP2P(t)
			beaconDB, _ := dbtest.SetupDB(t)

			connectPeers(t, p, tt.peers, p.Peers())
			cache.RLock()
			genesisRoot := cache.rootCache[0]
			cache.RUnlock()

			err := beaconDB.SaveBlock(context.Background(), testutil.NewBeaconBlock())
			require.NoError(t, err)

			st := testutil.NewBeaconState()
			mc := &mock.ChainService{
				State: st,
				Root:  genesisRoot[:],
				DB:    beaconDB,
			} // no-op mock
			s := &Service{
				chain:        mc,
				p2p:          p,
				db:           beaconDB,
				synced:       false,
				chainStarted: true,
			}
			assert.NoError(t, s.roundRobinSync(makeGenesisTime(tt.currentSlot)))
			if s.chain.HeadSlot() != tt.currentSlot {
				t.Errorf("Head slot (%d) is not currentSlot (%d)", s.chain.HeadSlot(), tt.currentSlot)
			}
			assert.Equal(t, len(tt.expectedBlockSlots), len(mc.BlocksReceived), "Processes wrong number of blocks")
			var receivedBlockSlots []uint64
			for _, blk := range mc.BlocksReceived {
				receivedBlockSlots = append(receivedBlockSlots, blk.Block.Slot)
			}
			missing := sliceutil.NotUint64(sliceutil.IntersectionUint64(tt.expectedBlockSlots, receivedBlockSlots), tt.expectedBlockSlots)
			if len(missing) > 0 {
				t.Errorf("Missing blocks at slots %v", missing)
			}
		})
	}
}

func TestService_processBlock(t *testing.T) {
	beaconDB, _ := dbtest.SetupDB(t)
	genesisBlk := testutil.NewBeaconBlock()
	genesisBlkRoot, err := stateutil.BlockRoot(genesisBlk.Block)
	require.NoError(t, err)
	err = beaconDB.SaveBlock(context.Background(), genesisBlk)
	require.NoError(t, err)
	st := testutil.NewBeaconState()
	s := NewInitialSync(&Config{
		P2P: p2pt.NewTestP2P(t),
		DB:  beaconDB,
		Chain: &mock.ChainService{
			State: st,
			Root:  genesisBlkRoot[:],
			DB:    beaconDB,
		},
	})
	ctx := context.Background()
	genesis := makeGenesisTime(32)

	t.Run("process duplicate block", func(t *testing.T) {
		blk1 := testutil.NewBeaconBlock()
		blk1.Block.Slot = 1
		blk1.Block.ParentRoot = genesisBlkRoot[:]
		blk1Root, err := stateutil.BlockRoot(blk1.Block)
		require.NoError(t, err)
		blk2 := testutil.NewBeaconBlock()
		blk2.Block.Slot = 2
		blk2.Block.ParentRoot = blk1Root[:]

		// Process block normally.
		err = s.processBlock(ctx, genesis, blk1, func(
			ctx context.Context, block *eth.SignedBeaconBlock, blockRoot [32]byte) error {
			assert.NoError(t, s.chain.ReceiveBlock(ctx, block, blockRoot))
			return nil
		})
		assert.NoError(t, err)

		// Duplicate processing should trigger error.
		err = s.processBlock(ctx, genesis, blk1, func(
			ctx context.Context, block *eth.SignedBeaconBlock, blockRoot [32]byte) error {
			return nil
		})
		expectedErr := fmt.Errorf("slot %d already processed", blk1.Block.Slot)
		assert.ErrorContains(t, expectedErr.Error(), err)

		// Continue normal processing, should proceed w/o errors.
		err = s.processBlock(ctx, genesis, blk2, func(
			ctx context.Context, block *eth.SignedBeaconBlock, blockRoot [32]byte) error {
			assert.NoError(t, s.chain.ReceiveBlock(ctx, block, blockRoot))
			return nil
		})
		assert.NoError(t, err)
		assert.Equal(t, uint64(2), s.chain.HeadSlot(), "Unexpected head slot")
	})
}

func TestService_processBlockBatch(t *testing.T) {
	beaconDB, _ := dbtest.SetupDB(t)
	genesisBlk := testutil.NewBeaconBlock()
	genesisBlkRoot, err := stateutil.BlockRoot(genesisBlk.Block)
	require.NoError(t, err)
	err = beaconDB.SaveBlock(context.Background(), genesisBlk)
	require.NoError(t, err)
	st := testutil.NewBeaconState()
	s := NewInitialSync(&Config{
		P2P: p2pt.NewTestP2P(t),
		DB:  beaconDB,
		Chain: &mock.ChainService{
			State: st,
			Root:  genesisBlkRoot[:],
			DB:    beaconDB,
		},
	})
	ctx := context.Background()
	genesis := makeGenesisTime(32)

	t.Run("process non-linear batch", func(t *testing.T) {
		batch := []*eth.SignedBeaconBlock{}
		currBlockRoot := genesisBlkRoot
		for i := 1; i < 10; i++ {
			parentRoot := currBlockRoot
			blk1 := testutil.NewBeaconBlock()
			blk1.Block.Slot = uint64(i)
			blk1.Block.ParentRoot = parentRoot[:]
			blk1Root, err := stateutil.BlockRoot(blk1.Block)
			require.NoError(t, err)
			err = beaconDB.SaveBlock(context.Background(), blk1)
			require.NoError(t, err)
			batch = append(batch, blk1)
			currBlockRoot = blk1Root
		}

		batch2 := []*eth.SignedBeaconBlock{}
		for i := 10; i < 20; i++ {
			parentRoot := currBlockRoot
			blk1 := testutil.NewBeaconBlock()
			blk1.Block.Slot = uint64(i)
			blk1.Block.ParentRoot = parentRoot[:]
			blk1Root, err := stateutil.BlockRoot(blk1.Block)
			require.NoError(t, err)
			err = beaconDB.SaveBlock(context.Background(), blk1)
			require.NoError(t, err)
			batch2 = append(batch2, blk1)
			currBlockRoot = blk1Root
		}

		// Process block normally.
		err = s.processBatchedBlocks(ctx, genesis, batch, func(
			ctx context.Context, blks []*eth.SignedBeaconBlock, blockRoots [][32]byte) error {
			assert.NoError(t, s.chain.ReceiveBlockBatch(ctx, blks, blockRoots))
			return nil
		})
		assert.NoError(t, err)

		// Duplicate processing should trigger error.
		err = s.processBatchedBlocks(ctx, genesis, batch, func(
			ctx context.Context, blocks []*eth.SignedBeaconBlock, blockRoots [][32]byte) error {
			return nil
		})
		expectedErr := fmt.Sprintf("no good blocks in batch")
		assert.ErrorContains(t, expectedErr, err)

		var badBatch2 []*eth.SignedBeaconBlock
		for i, b := range batch2 {
			// create a non-linear batch
			if i%3 == 0 && i != 0 {
				continue
			}
			badBatch2 = append(badBatch2, b)
		}

		// Bad batch should fail because it is non linear
		err = s.processBatchedBlocks(ctx, genesis, badBatch2, func(
			ctx context.Context, blks []*eth.SignedBeaconBlock, blockRoots [][32]byte) error {
			return nil
		})
		expectedSubErr := "expected linear block list"
		assert.ErrorContains(t, expectedSubErr, err)

		// Continue normal processing, should proceed w/o errors.
		err = s.processBatchedBlocks(ctx, genesis, batch2, func(
			ctx context.Context, blks []*eth.SignedBeaconBlock, blockRoots [][32]byte) error {
			assert.NoError(t, s.chain.ReceiveBlockBatch(ctx, blks, blockRoots))
			return nil
		})
		assert.NoError(t, err)
		assert.Equal(t, uint64(19), s.chain.HeadSlot(), "Unexpected head slot")
	})
}
