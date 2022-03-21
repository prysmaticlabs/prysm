package kv

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
	"google.golang.org/protobuf/proto"
)

var blockTests = []struct {
	name     string
	newBlock func(types.Slot, []byte) (block.SignedBeaconBlock, error)
}{
	{
		name: "phase0",
		newBlock: func(slot types.Slot, root []byte) (block.SignedBeaconBlock, error) {
			b := util.NewBeaconBlock()
			b.Block.Slot = slot
			if root != nil {
				b.Block.ParentRoot = root
			}
			return wrapper.WrappedPhase0SignedBeaconBlock(b), nil
		},
	},
	{
		name: "altair",
		newBlock: func(slot types.Slot, root []byte) (block.SignedBeaconBlock, error) {
			b := util.NewBeaconBlockAltair()
			b.Block.Slot = slot
			if root != nil {
				b.Block.ParentRoot = root
			}
			return wrapper.WrappedAltairSignedBeaconBlock(b)
		},
	},
	{
		name: "bellatrix",
		newBlock: func(slot types.Slot, root []byte) (block.SignedBeaconBlock, error) {
			b := util.NewBeaconBlockBellatrix()
			b.Block.Slot = slot
			if root != nil {
				b.Block.ParentRoot = root
			}
			return wrapper.WrappedBellatrixSignedBeaconBlock(b)
		},
	},
}

func TestStore_SaveBlock_NoDuplicates(t *testing.T) {
	BlockCacheSize = 1
	slot := types.Slot(20)
	ctx := context.Background()

	for _, tt := range blockTests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupDB(t)

			// First we save a previous block to ensure the cache max size is reached.
			prevBlock, err := tt.newBlock(slot-1, bytesutil.PadTo([]byte{1, 2, 3}, 32))
			require.NoError(t, err)
			require.NoError(t, db.SaveBlock(ctx, prevBlock))

			blk, err := tt.newBlock(slot, bytesutil.PadTo([]byte{1, 2, 3}, 32))
			require.NoError(t, err)

			// Even with a full cache, saving new blocks should not cause
			// duplicated blocks in the DB.
			for i := 0; i < 100; i++ {
				require.NoError(t, db.SaveBlock(ctx, blk))
			}

			f := filters.NewFilter().SetStartSlot(slot).SetEndSlot(slot)
			retrieved, _, err := db.Blocks(ctx, f)
			require.NoError(t, err)
			assert.Equal(t, 1, len(retrieved))
		})
	}
	// We reset the block cache size.
	BlockCacheSize = 256
}

func TestStore_BlocksCRUD(t *testing.T) {
	ctx := context.Background()

	for _, tt := range blockTests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupDB(t)

			blk, err := tt.newBlock(types.Slot(20), bytesutil.PadTo([]byte{1, 2, 3}, 32))
			require.NoError(t, err)
			blockRoot, err := blk.Block().HashTreeRoot()
			require.NoError(t, err)

			retrievedBlock, err := db.Block(ctx, blockRoot)
			require.NoError(t, err)
			assert.DeepEqual(t, nil, retrievedBlock, "Expected nil block")
			require.NoError(t, db.SaveBlock(ctx, blk))
			assert.Equal(t, true, db.HasBlock(ctx, blockRoot), "Expected block to exist in the db")
			retrievedBlock, err = db.Block(ctx, blockRoot)
			require.NoError(t, err)
			assert.Equal(t, true, proto.Equal(blk.Proto(), retrievedBlock.Proto()), "Wanted: %v, received: %v", blk, retrievedBlock)
		})
	}
}

func TestStore_BlocksHandleZeroCase(t *testing.T) {
	for _, tt := range blockTests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupDB(t)
			ctx := context.Background()
			numBlocks := 10
			totalBlocks := make([]block.SignedBeaconBlock, numBlocks)
			for i := 0; i < len(totalBlocks); i++ {
				b, err := tt.newBlock(types.Slot(i), bytesutil.PadTo([]byte("parent"), 32))
				require.NoError(t, err)
				totalBlocks[i] = b
				_, err = totalBlocks[i].Block().HashTreeRoot()
				require.NoError(t, err)
			}
			require.NoError(t, db.SaveBlocks(ctx, totalBlocks))
			zeroFilter := filters.NewFilter().SetStartSlot(0).SetEndSlot(0)
			retrieved, _, err := db.Blocks(ctx, zeroFilter)
			require.NoError(t, err)
			assert.Equal(t, 1, len(retrieved), "Unexpected number of blocks received, expected one")
		})
	}
}

func TestStore_BlocksHandleInvalidEndSlot(t *testing.T) {
	for _, tt := range blockTests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupDB(t)
			ctx := context.Background()
			numBlocks := 10
			totalBlocks := make([]block.SignedBeaconBlock, numBlocks)
			// Save blocks from slot 1 onwards.
			for i := 0; i < len(totalBlocks); i++ {
				b, err := tt.newBlock(types.Slot(i+1), bytesutil.PadTo([]byte("parent"), 32))
				require.NoError(t, err)
				totalBlocks[i] = b
				_, err = totalBlocks[i].Block().HashTreeRoot()
				require.NoError(t, err)
			}
			require.NoError(t, db.SaveBlocks(ctx, totalBlocks))
			badFilter := filters.NewFilter().SetStartSlot(5).SetEndSlot(1)
			_, _, err := db.Blocks(ctx, badFilter)
			require.ErrorContains(t, errInvalidSlotRange.Error(), err)

			goodFilter := filters.NewFilter().SetStartSlot(0).SetEndSlot(1)
			requested, _, err := db.Blocks(ctx, goodFilter)
			require.NoError(t, err)
			assert.Equal(t, 1, len(requested), "Unexpected number of blocks received, only expected two")
		})
	}
}

func TestStore_DeleteBlock(t *testing.T) {
	slotsPerEpoch := uint64(params.BeaconConfig().SlotsPerEpoch)
	db := setupDB(t)
	ctx := context.Background()

	require.NoError(t, db.SaveGenesisBlockRoot(ctx, genesisBlockRoot))
	blks := makeBlocks(t, 0, slotsPerEpoch*4, genesisBlockRoot)
	require.NoError(t, db.SaveBlocks(ctx, blks))

	root, err := blks[slotsPerEpoch].Block().HashTreeRoot()
	require.NoError(t, err)
	cp := &ethpb.Checkpoint{
		Epoch: 1,
		Root:  root[:],
	}
	st, err := util.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, db.SaveState(ctx, st, root))
	require.NoError(t, db.SaveFinalizedCheckpoint(ctx, cp))

	root2, err := blks[4*slotsPerEpoch-2].Block().HashTreeRoot()
	require.NoError(t, err)
	b, err := db.Block(ctx, root2)
	require.NoError(t, err)
	require.NotNil(t, b)
	require.NoError(t, db.DeleteBlock(ctx, root2))
	st, err = db.State(ctx, root2)
	require.NoError(t, err)
	require.Equal(t, st, nil)

	b, err = db.Block(ctx, root2)
	require.NoError(t, err)
	require.Equal(t, b, nil)

	require.ErrorIs(t, db.DeleteBlock(ctx, root), ErrDeleteJustifiedAndFinalized)

}

func TestStore_GenesisBlock(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	genesisBlock := util.NewBeaconBlock()
	genesisBlock.Block.ParentRoot = bytesutil.PadTo([]byte{1, 2, 3}, 32)
	blockRoot, err := genesisBlock.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, db.SaveGenesisBlockRoot(ctx, blockRoot))
	require.NoError(t, db.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(genesisBlock)))
	retrievedBlock, err := db.GenesisBlock(ctx)
	require.NoError(t, err)
	assert.Equal(t, true, proto.Equal(genesisBlock, retrievedBlock.Proto()), "Wanted: %v, received: %v", genesisBlock, retrievedBlock)
}

func TestStore_BlocksCRUD_NoCache(t *testing.T) {
	for _, tt := range blockTests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupDB(t)
			ctx := context.Background()
			blk, err := tt.newBlock(types.Slot(20), bytesutil.PadTo([]byte{1, 2, 3}, 32))
			require.NoError(t, err)
			blockRoot, err := blk.Block().HashTreeRoot()
			require.NoError(t, err)
			retrievedBlock, err := db.Block(ctx, blockRoot)
			require.NoError(t, err)
			require.DeepEqual(t, nil, retrievedBlock, "Expected nil block")
			require.NoError(t, db.SaveBlock(ctx, blk))
			db.blockCache.Del(string(blockRoot[:]))
			assert.Equal(t, true, db.HasBlock(ctx, blockRoot), "Expected block to exist in the db")
			retrievedBlock, err = db.Block(ctx, blockRoot)
			require.NoError(t, err)
			assert.Equal(t, true, proto.Equal(blk.Proto(), retrievedBlock.Proto()), "Wanted: %v, received: %v", blk, retrievedBlock)
		})
	}
}

func TestStore_Blocks_FiltersCorrectly(t *testing.T) {
	for _, tt := range blockTests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupDB(t)
			b4, err := tt.newBlock(types.Slot(4), bytesutil.PadTo([]byte("parent"), 32))
			require.NoError(t, err)
			b5, err := tt.newBlock(types.Slot(5), bytesutil.PadTo([]byte("parent2"), 32))
			require.NoError(t, err)
			b6, err := tt.newBlock(types.Slot(6), bytesutil.PadTo([]byte("parent2"), 32))
			require.NoError(t, err)
			b7, err := tt.newBlock(types.Slot(7), bytesutil.PadTo([]byte("parent3"), 32))
			require.NoError(t, err)
			b8, err := tt.newBlock(types.Slot(8), bytesutil.PadTo([]byte("parent4"), 32))
			require.NoError(t, err)
			blocks := []block.SignedBeaconBlock{
				b4,
				b5,
				b6,
				b7,
				b8,
			}
			ctx := context.Background()
			require.NoError(t, db.SaveBlocks(ctx, blocks))

			tests := []struct {
				filter            *filters.QueryFilter
				expectedNumBlocks int
			}{
				{
					filter:            filters.NewFilter().SetParentRoot(bytesutil.PadTo([]byte("parent2"), 32)),
					expectedNumBlocks: 2,
				},
				{
					// No block meets the criteria below.
					filter:            filters.NewFilter().SetParentRoot(bytesutil.PadTo([]byte{3, 4, 5}, 32)),
					expectedNumBlocks: 0,
				},
				{
					// Block slot range filter criteria.
					filter:            filters.NewFilter().SetStartSlot(5).SetEndSlot(7),
					expectedNumBlocks: 3,
				},
				{
					filter:            filters.NewFilter().SetStartSlot(7).SetEndSlot(7),
					expectedNumBlocks: 1,
				},
				{
					filter:            filters.NewFilter().SetStartSlot(4).SetEndSlot(8),
					expectedNumBlocks: 5,
				},
				{
					filter:            filters.NewFilter().SetStartSlot(4).SetEndSlot(5),
					expectedNumBlocks: 2,
				},
				{
					filter:            filters.NewFilter().SetStartSlot(5).SetEndSlot(9),
					expectedNumBlocks: 4,
				},
				{
					filter:            filters.NewFilter().SetEndSlot(7),
					expectedNumBlocks: 4,
				},
				{
					filter:            filters.NewFilter().SetEndSlot(8),
					expectedNumBlocks: 5,
				},
				{
					filter:            filters.NewFilter().SetStartSlot(5).SetEndSlot(10),
					expectedNumBlocks: 4,
				},
				{
					// Composite filter criteria.
					filter: filters.NewFilter().
						SetParentRoot(bytesutil.PadTo([]byte("parent2"), 32)).
						SetStartSlot(6).
						SetEndSlot(8),
					expectedNumBlocks: 1,
				},
			}
			for _, tt2 := range tests {
				retrievedBlocks, _, err := db.Blocks(ctx, tt2.filter)
				require.NoError(t, err)
				assert.Equal(t, tt2.expectedNumBlocks, len(retrievedBlocks), "Unexpected number of blocks")
			}
		})
	}
}

func TestStore_Blocks_VerifyBlockRoots(t *testing.T) {
	for _, tt := range blockTests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			db := setupDB(t)
			b1, err := tt.newBlock(types.Slot(1), nil)
			require.NoError(t, err)
			r1, err := b1.Block().HashTreeRoot()
			require.NoError(t, err)
			b2, err := tt.newBlock(types.Slot(2), nil)
			require.NoError(t, err)
			r2, err := b2.Block().HashTreeRoot()
			require.NoError(t, err)

			require.NoError(t, db.SaveBlock(ctx, b1))
			require.NoError(t, db.SaveBlock(ctx, b2))

			filter := filters.NewFilter().SetStartSlot(b1.Block().Slot()).SetEndSlot(b2.Block().Slot())
			roots, err := db.BlockRoots(ctx, filter)
			require.NoError(t, err)

			assert.DeepEqual(t, [][32]byte{r1, r2}, roots)
		})
	}
}

func TestStore_Blocks_Retrieve_SlotRange(t *testing.T) {
	for _, tt := range blockTests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupDB(t)
			totalBlocks := make([]block.SignedBeaconBlock, 500)
			for i := 0; i < 500; i++ {
				b, err := tt.newBlock(types.Slot(i), bytesutil.PadTo([]byte("parent"), 32))
				require.NoError(t, err)
				totalBlocks[i] = b
			}
			ctx := context.Background()
			require.NoError(t, db.SaveBlocks(ctx, totalBlocks))
			retrieved, _, err := db.Blocks(ctx, filters.NewFilter().SetStartSlot(100).SetEndSlot(399))
			require.NoError(t, err)
			assert.Equal(t, 300, len(retrieved))
		})
	}
}

func TestStore_Blocks_Retrieve_Epoch(t *testing.T) {
	for _, tt := range blockTests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupDB(t)
			slots := params.BeaconConfig().SlotsPerEpoch.Mul(7)
			totalBlocks := make([]block.SignedBeaconBlock, slots)
			for i := types.Slot(0); i < slots; i++ {
				b, err := tt.newBlock(i, bytesutil.PadTo([]byte("parent"), 32))
				require.NoError(t, err)
				totalBlocks[i] = b
			}
			ctx := context.Background()
			require.NoError(t, db.SaveBlocks(ctx, totalBlocks))
			retrieved, _, err := db.Blocks(ctx, filters.NewFilter().SetStartEpoch(5).SetEndEpoch(6))
			require.NoError(t, err)
			want := params.BeaconConfig().SlotsPerEpoch.Mul(2)
			assert.Equal(t, uint64(want), uint64(len(retrieved)))
			retrieved, _, err = db.Blocks(ctx, filters.NewFilter().SetStartEpoch(0).SetEndEpoch(0))
			require.NoError(t, err)
			want = params.BeaconConfig().SlotsPerEpoch
			assert.Equal(t, uint64(want), uint64(len(retrieved)))
		})
	}
}

func TestStore_Blocks_Retrieve_SlotRangeWithStep(t *testing.T) {
	for _, tt := range blockTests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupDB(t)
			totalBlocks := make([]block.SignedBeaconBlock, 500)
			for i := 0; i < 500; i++ {
				b, err := tt.newBlock(types.Slot(i), bytesutil.PadTo([]byte("parent"), 32))
				require.NoError(t, err)
				totalBlocks[i] = b
			}
			const step = 2
			ctx := context.Background()
			require.NoError(t, db.SaveBlocks(ctx, totalBlocks))
			retrieved, _, err := db.Blocks(ctx, filters.NewFilter().SetStartSlot(100).SetEndSlot(399).SetSlotStep(step))
			require.NoError(t, err)
			assert.Equal(t, 150, len(retrieved))
			for _, b := range retrieved {
				assert.Equal(t, types.Slot(0), (b.Block().Slot()-100)%step, "Unexpect block slot %d", b.Block().Slot())
			}
		})
	}
}

func TestStore_SaveBlock_CanGetHighestAt(t *testing.T) {
	for _, tt := range blockTests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupDB(t)
			ctx := context.Background()

			block1, err := tt.newBlock(types.Slot(1), nil)
			require.NoError(t, err)
			block2, err := tt.newBlock(types.Slot(10), nil)
			require.NoError(t, err)
			block3, err := tt.newBlock(types.Slot(100), nil)
			require.NoError(t, err)

			require.NoError(t, db.SaveBlock(ctx, block1))
			require.NoError(t, db.SaveBlock(ctx, block2))
			require.NoError(t, db.SaveBlock(ctx, block3))

			highestAt, err := db.HighestSlotBlocksBelow(ctx, 2)
			require.NoError(t, err)
			assert.Equal(t, false, len(highestAt) <= 0, "Got empty highest at slice")
			assert.Equal(t, true, proto.Equal(block1.Proto(), highestAt[0].Proto()), "Wanted: %v, received: %v", block1, highestAt[0])
			highestAt, err = db.HighestSlotBlocksBelow(ctx, 11)
			require.NoError(t, err)
			assert.Equal(t, false, len(highestAt) <= 0, "Got empty highest at slice")
			assert.Equal(t, true, proto.Equal(block2.Proto(), highestAt[0].Proto()), "Wanted: %v, received: %v", block2, highestAt[0])
			highestAt, err = db.HighestSlotBlocksBelow(ctx, 101)
			require.NoError(t, err)
			assert.Equal(t, false, len(highestAt) <= 0, "Got empty highest at slice")
			assert.Equal(t, true, proto.Equal(block3.Proto(), highestAt[0].Proto()), "Wanted: %v, received: %v", block3, highestAt[0])
		})
	}
}

func TestStore_GenesisBlock_CanGetHighestAt(t *testing.T) {
	for _, tt := range blockTests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupDB(t)
			ctx := context.Background()

			genesisBlock, err := tt.newBlock(types.Slot(0), nil)
			require.NoError(t, err)
			genesisRoot, err := genesisBlock.Block().HashTreeRoot()
			require.NoError(t, err)
			require.NoError(t, db.SaveGenesisBlockRoot(ctx, genesisRoot))
			require.NoError(t, db.SaveBlock(ctx, genesisBlock))
			block1, err := tt.newBlock(types.Slot(1), nil)
			require.NoError(t, err)
			require.NoError(t, db.SaveBlock(ctx, block1))

			highestAt, err := db.HighestSlotBlocksBelow(ctx, 2)
			require.NoError(t, err)
			assert.Equal(t, true, proto.Equal(block1.Proto(), highestAt[0].Proto()), "Wanted: %v, received: %v", block1, highestAt[0])
			highestAt, err = db.HighestSlotBlocksBelow(ctx, 1)
			require.NoError(t, err)
			assert.Equal(t, true, proto.Equal(genesisBlock.Proto(), highestAt[0].Proto()), "Wanted: %v, received: %v", genesisBlock, highestAt[0])
			highestAt, err = db.HighestSlotBlocksBelow(ctx, 0)
			require.NoError(t, err)
			assert.Equal(t, true, proto.Equal(genesisBlock.Proto(), highestAt[0].Proto()), "Wanted: %v, received: %v", genesisBlock, highestAt[0])
		})
	}
}

func TestStore_SaveBlocks_HasCachedBlocks(t *testing.T) {
	for _, tt := range blockTests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupDB(t)
			ctx := context.Background()

			b := make([]block.SignedBeaconBlock, 500)
			for i := 0; i < 500; i++ {
				blk, err := tt.newBlock(types.Slot(i), bytesutil.PadTo([]byte("parent"), 32))
				require.NoError(t, err)
				b[i] = blk
			}

			require.NoError(t, db.SaveBlock(ctx, b[0]))
			require.NoError(t, db.SaveBlocks(ctx, b))
			f := filters.NewFilter().SetStartSlot(0).SetEndSlot(500)

			blks, _, err := db.Blocks(ctx, f)
			require.NoError(t, err)
			assert.Equal(t, 500, len(blks), "Did not get wanted blocks")
		})
	}
}

func TestStore_SaveBlocks_HasRootsMatched(t *testing.T) {
	for _, tt := range blockTests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupDB(t)
			ctx := context.Background()

			b := make([]block.SignedBeaconBlock, 500)
			for i := 0; i < 500; i++ {
				blk, err := tt.newBlock(types.Slot(i), bytesutil.PadTo([]byte("parent"), 32))
				require.NoError(t, err)
				b[i] = blk
			}

			require.NoError(t, db.SaveBlocks(ctx, b))
			f := filters.NewFilter().SetStartSlot(0).SetEndSlot(500)

			blks, roots, err := db.Blocks(ctx, f)
			require.NoError(t, err)
			assert.Equal(t, 500, len(blks), "Did not get wanted blocks")

			for i, blk := range blks {
				rt, err := blk.Block().HashTreeRoot()
				require.NoError(t, err)
				assert.Equal(t, roots[i], rt, "mismatch of block roots")
			}
		})
	}
}

func TestStore_BlocksBySlot_BlockRootsBySlot(t *testing.T) {
	for _, tt := range blockTests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupDB(t)
			ctx := context.Background()

			b1, err := tt.newBlock(types.Slot(20), nil)
			require.NoError(t, err)
			require.NoError(t, db.SaveBlock(ctx, b1))
			b2, err := tt.newBlock(types.Slot(100), bytesutil.PadTo([]byte("parent1"), 32))
			require.NoError(t, err)
			require.NoError(t, db.SaveBlock(ctx, b2))
			b3, err := tt.newBlock(types.Slot(100), bytesutil.PadTo([]byte("parent2"), 32))
			require.NoError(t, err)
			require.NoError(t, db.SaveBlock(ctx, b3))

			r1, err := b1.Block().HashTreeRoot()
			require.NoError(t, err)
			r2, err := b2.Block().HashTreeRoot()
			require.NoError(t, err)
			r3, err := b3.Block().HashTreeRoot()
			require.NoError(t, err)

			hasBlocks, retrievedBlocks, err := db.BlocksBySlot(ctx, 1)
			require.NoError(t, err)
			assert.Equal(t, 0, len(retrievedBlocks), "Unexpected number of blocks received, expected none")
			assert.Equal(t, false, hasBlocks, "Expected no blocks")
			hasBlocks, retrievedBlocks, err = db.BlocksBySlot(ctx, 20)
			require.NoError(t, err)
			assert.Equal(t, true, proto.Equal(b1.Proto(), retrievedBlocks[0].Proto()), "Wanted: %v, received: %v", b1, retrievedBlocks[0])
			assert.Equal(t, true, hasBlocks, "Expected to have blocks")
			hasBlocks, retrievedBlocks, err = db.BlocksBySlot(ctx, 100)
			require.NoError(t, err)
			if len(retrievedBlocks) != 2 {
				t.Fatalf("Expected 2 blocks, received %d blocks", len(retrievedBlocks))
			}
			assert.Equal(t, true, proto.Equal(b2.Proto(), retrievedBlocks[0].Proto()), "Wanted: %v, received: %v", b2, retrievedBlocks[0])
			assert.Equal(t, true, proto.Equal(b3.Proto(), retrievedBlocks[1].Proto()), "Wanted: %v, received: %v", b3, retrievedBlocks[1])
			assert.Equal(t, true, hasBlocks, "Expected to have blocks")

			hasBlockRoots, retrievedBlockRoots, err := db.BlockRootsBySlot(ctx, 1)
			require.NoError(t, err)
			assert.DeepEqual(t, [][32]byte{}, retrievedBlockRoots)
			assert.Equal(t, false, hasBlockRoots, "Expected no block roots")
			hasBlockRoots, retrievedBlockRoots, err = db.BlockRootsBySlot(ctx, 20)
			require.NoError(t, err)
			assert.DeepEqual(t, [][32]byte{r1}, retrievedBlockRoots)
			assert.Equal(t, true, hasBlockRoots, "Expected no block roots")
			hasBlockRoots, retrievedBlockRoots, err = db.BlockRootsBySlot(ctx, 100)
			require.NoError(t, err)
			assert.DeepEqual(t, [][32]byte{r2, r3}, retrievedBlockRoots)
			assert.Equal(t, true, hasBlockRoots, "Expected no block roots")
		})
	}
}

func TestStore_FeeRecipientByValidatorID(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	ids := []types.ValidatorIndex{0, 0, 0}
	feeRecipients := []common.Address{{}, {}, {}, {}}
	require.ErrorContains(t, "validatorIDs and feeRecipients must be the same length", db.SaveFeeRecipientsByValidatorIDs(ctx, ids, feeRecipients))

	ids = []types.ValidatorIndex{0, 1, 2}
	feeRecipients = []common.Address{{'a'}, {'b'}, {'c'}}
	require.NoError(t, db.SaveFeeRecipientsByValidatorIDs(ctx, ids, feeRecipients))
	f, err := db.FeeRecipientByValidatorID(ctx, 0)
	require.NoError(t, err)
	require.Equal(t, common.Address{'a'}, f)
	f, err = db.FeeRecipientByValidatorID(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, common.Address{'b'}, f)
	f, err = db.FeeRecipientByValidatorID(ctx, 2)
	require.NoError(t, err)
	require.Equal(t, common.Address{'c'}, f)
	_, err = db.FeeRecipientByValidatorID(ctx, 3)
	want := errors.Wrap(ErrNotFoundFeeRecipient, "validator id 3")
	require.Equal(t, want.Error(), err.Error())
}
