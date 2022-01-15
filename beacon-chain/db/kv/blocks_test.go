package kv

import (
	"context"
	"sort"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
	"google.golang.org/protobuf/proto"
)

func TestStore_SaveBlock_NoDuplicates(t *testing.T) {
	slot := types.Slot(20)
	ctx := context.Background()

	setup := func(t testing.TB) (*Store, func()) {
		BlockCacheSize = 1
		return setupDB(t), func() {
			// We reset the block cache size.
			BlockCacheSize = 256
		}
	}

	eval := func(t testing.TB, db *Store, slot types.Slot, blk interface{}) {
		// Even with a full cache, saving new blocks should not cause
		// duplicated blocks in the DB.
		for i := 0; i < 100; i++ {
			wsb, err := wrapper.WrappedSignedBeaconBlock(blk)
			require.NoError(t, err)
			require.NoError(t, db.SaveBlock(ctx, wsb))
		}

		f := filters.NewFilter().SetStartSlot(slot).SetEndSlot(slot)
		retrieved, _, err := db.Blocks(ctx, f)
		require.NoError(t, err)
		assert.Equal(t, 1, len(retrieved))
	}

	t.Run("phase0", func(t *testing.T) {
		db, teardown := setup(t)
		defer teardown()

		// First we save a previous block to ensure the cache max size is reached.
		prevBlock := util.NewBeaconBlock()
		prevBlock.Block.Slot = slot - 1
		prevBlock.Block.ParentRoot = bytesutil.PadTo([]byte{1, 2, 3}, 32)
		require.NoError(t, db.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(prevBlock)))

		blk := util.NewBeaconBlock()
		blk.Block.Slot = slot
		blk.Block.ParentRoot = bytesutil.PadTo([]byte{1, 2, 3}, 32)
		eval(t, db, slot, blk)
	})

	t.Run("altair", func(t *testing.T) {
		db, teardown := setup(t)
		defer teardown()

		// First we save a previous block to ensure the cache max size is reached.
		prevBlock := util.NewBeaconBlockAltair()
		prevBlock.Block.Slot = slot - 1
		prevBlock.Block.ParentRoot = bytesutil.PadTo([]byte{1, 2, 3}, 32)
		wsb, err := wrapper.WrappedAltairSignedBeaconBlock(prevBlock)
		require.NoError(t, err)
		require.NoError(t, db.SaveBlock(ctx, wsb))

		blk := util.NewBeaconBlockAltair()
		blk.Block.Slot = slot
		blk.Block.ParentRoot = bytesutil.PadTo([]byte{1, 2, 3}, 32)

		eval(t, db, slot, blk)
	})

	t.Run("bellatrix", func(t *testing.T) {
		db, teardown := setup(t)
		defer teardown()

		// First we save a previous block to ensure the cache max size is reached.
		prevBlock := util.NewBeaconBlockMerge()
		prevBlock.Block.Slot = slot - 1
		prevBlock.Block.ParentRoot = bytesutil.PadTo([]byte{1, 2, 3}, 32)
		wsb, err := wrapper.WrappedMergeSignedBeaconBlock(prevBlock)
		require.NoError(t, err)
		require.NoError(t, db.SaveBlock(ctx, wsb))

		blk := util.NewBeaconBlockMerge()
		blk.Block.Slot = slot
		blk.Block.ParentRoot = bytesutil.PadTo([]byte{1, 2, 3}, 32)

		eval(t, db, slot, blk)
	})
}

func TestStore_BlocksCRUD(t *testing.T) {
	ctx := context.Background()

	eval := func(t testing.TB, db *Store, blockRoot [32]byte, iBlk interface{}) {
		blk, err := wrapper.WrappedSignedBeaconBlock(iBlk)
		require.NoError(t, err)

		retrievedBlock, err := db.Block(ctx, blockRoot)
		require.NoError(t, err)
		assert.DeepEqual(t, nil, retrievedBlock, "Expected nil block")
		require.NoError(t, db.SaveBlock(ctx, blk))
		assert.Equal(t, true, db.HasBlock(ctx, blockRoot), "Expected block to exist in the db")
		retrievedBlock, err = db.Block(ctx, blockRoot)
		require.NoError(t, err)
		assert.Equal(t, true, proto.Equal(blk.Proto(), retrievedBlock.Proto()), "Wanted: %v, received: %v", blk, retrievedBlock)
		require.NoError(t, db.deleteBlock(ctx, blockRoot))
		assert.Equal(t, false, db.HasBlock(ctx, blockRoot), "Expected block to have been deleted from the db")
	}

	t.Run("phase0", func(t *testing.T) {
		db := setupDB(t)

		blk := util.NewBeaconBlock()
		blk.Block.Slot = 20
		blk.Block.ParentRoot = bytesutil.PadTo([]byte{1, 2, 3}, 32)
		blockRoot, err := blk.Block.HashTreeRoot()
		require.NoError(t, err)

		eval(t, db, blockRoot, blk)
	})

	t.Run("altair", func(t *testing.T) {
		db := setupDB(t)

		blk := util.NewBeaconBlockAltair()
		blk.Block.Slot = 20
		blk.Block.ParentRoot = bytesutil.PadTo([]byte{1, 2, 3}, 32)
		blockRoot, err := blk.Block.HashTreeRoot()
		require.NoError(t, err)

		eval(t, db, blockRoot, blk)
	})

	t.Run("bellatrix", func(t *testing.T) {
		db := setupDB(t)

		blk := util.NewBeaconBlockMerge()
		blk.Block.Slot = 20
		blk.Block.ParentRoot = bytesutil.PadTo([]byte{1, 2, 3}, 32)
		blockRoot, err := blk.Block.HashTreeRoot()
		require.NoError(t, err)

		eval(t, db, blockRoot, blk)
	})
}

func TestStore_BlocksBatchDelete(t *testing.T) {
	tests := []struct {
		name     string
		newBlock func(types.Slot, []byte) (block.SignedBeaconBlock, error)
	}{
		{
			name: "phase0",
			newBlock: func(slot types.Slot, root []byte) (block.SignedBeaconBlock, error) {
				b := util.NewBeaconBlock()
				b.Block.Slot = slot
				b.Block.ParentRoot = root
				return wrapper.WrappedPhase0SignedBeaconBlock(b), nil
			},
		},
		{
			name: "altair",
			newBlock: func(slot types.Slot, root []byte) (block.SignedBeaconBlock, error) {
				b := util.NewBeaconBlockAltair()
				b.Block.Slot = slot
				b.Block.ParentRoot = root
				return wrapper.WrappedAltairSignedBeaconBlock(b)
			},
		},
		{
			name: "bellatrix",
			newBlock: func(slot types.Slot, root []byte) (block.SignedBeaconBlock, error) {
				b := util.NewBeaconBlockMerge()
				b.Block.Slot = slot
				b.Block.ParentRoot = root
				return wrapper.WrappedMergeSignedBeaconBlock(b)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupDB(t)
			ctx := context.Background()
			numBlocks := 10
			totalBlocks := make([]block.SignedBeaconBlock, numBlocks)
			blockRoots := make([][32]byte, 0)
			oddBlocks := make([]block.SignedBeaconBlock, 0)
			for i := 0; i < len(totalBlocks); i++ {
				b, err := tt.newBlock(types.Slot(i), bytesutil.PadTo([]byte("parent"), 32))
				require.NoError(t, err)
				totalBlocks[i] = b
				if i%2 == 0 {
					r, err := totalBlocks[i].Block().HashTreeRoot()
					require.NoError(t, err)
					blockRoots = append(blockRoots, r)
				} else {
					oddBlocks = append(oddBlocks, totalBlocks[i])
				}
			}
			require.NoError(t, db.SaveBlocks(ctx, totalBlocks))
			retrieved, _, err := db.Blocks(ctx, filters.NewFilter().SetParentRoot(bytesutil.PadTo([]byte("parent"), 32)))
			require.NoError(t, err)
			assert.Equal(t, numBlocks, len(retrieved), "Unexpected number of blocks received")
			// We delete all even indexed blocks.
			require.NoError(t, db.deleteBlocks(ctx, blockRoots))
			// When we retrieve the data, only the odd indexed blocks should remain.
			retrieved, _, err = db.Blocks(ctx, filters.NewFilter().SetParentRoot(bytesutil.PadTo([]byte("parent"), 32)))
			require.NoError(t, err)
			sort.Slice(retrieved, func(i, j int) bool {
				return retrieved[i].Block().Slot() < retrieved[j].Block().Slot()
			})
			for i, blk := range retrieved {
				assert.Equal(t, true, proto.Equal(blk.Proto(), oddBlocks[i].Proto()), "Wanted: %v, received: %v", blk, oddBlocks[i])
			}
		})
	}
}

func TestStore_BlocksHandleZeroCase(t *testing.T) {
	tests := []struct {
		name     string
		newBlock func(types.Slot, []byte) (block.SignedBeaconBlock, error)
	}{
		{
			name: "phase0",
			newBlock: func(slot types.Slot, root []byte) (block.SignedBeaconBlock, error) {
				b := util.NewBeaconBlock()
				b.Block.Slot = slot
				b.Block.ParentRoot = root
				return wrapper.WrappedPhase0SignedBeaconBlock(b), nil
			},
		},
		{
			name: "altair",
			newBlock: func(slot types.Slot, root []byte) (block.SignedBeaconBlock, error) {
				b := util.NewBeaconBlockAltair()
				b.Block.Slot = slot
				b.Block.ParentRoot = root
				return wrapper.WrappedAltairSignedBeaconBlock(b)
			},
		},
		{
			name: "bellatrix",
			newBlock: func(slot types.Slot, root []byte) (block.SignedBeaconBlock, error) {
				b := util.NewBeaconBlockMerge()
				b.Block.Slot = slot
				b.Block.ParentRoot = root
				return wrapper.WrappedMergeSignedBeaconBlock(b)
			},
		},
	}

	for _, tt := range tests {
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
	tests := []struct {
		name     string
		newBlock func(types.Slot, []byte) (block.SignedBeaconBlock, error)
	}{
		{
			name: "phase0",
			newBlock: func(slot types.Slot, root []byte) (block.SignedBeaconBlock, error) {
				b := util.NewBeaconBlock()
				b.Block.Slot = slot
				b.Block.ParentRoot = root
				return wrapper.WrappedPhase0SignedBeaconBlock(b), nil
			},
		},
		{
			name: "altair",
			newBlock: func(slot types.Slot, root []byte) (block.SignedBeaconBlock, error) {
				b := util.NewBeaconBlockAltair()
				b.Block.Slot = slot
				b.Block.ParentRoot = root
				return wrapper.WrappedAltairSignedBeaconBlock(b)
			},
		},
		{
			name: "bellatrix",
			newBlock: func(slot types.Slot, root []byte) (block.SignedBeaconBlock, error) {
				b := util.NewBeaconBlockMerge()
				b.Block.Slot = slot
				b.Block.ParentRoot = root
				return wrapper.WrappedMergeSignedBeaconBlock(b)
			},
		},
	}

	for _, tt := range tests {
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
	tests := []struct {
		name     string
		newBlock func(types.Slot, []byte) (block.SignedBeaconBlock, error)
	}{
		{
			name: "phase0",
			newBlock: func(slot types.Slot, root []byte) (block.SignedBeaconBlock, error) {
				b := util.NewBeaconBlock()
				b.Block.Slot = slot
				b.Block.ParentRoot = root
				return wrapper.WrappedPhase0SignedBeaconBlock(b), nil
			},
		},
		{
			name: "altair",
			newBlock: func(slot types.Slot, root []byte) (block.SignedBeaconBlock, error) {
				b := util.NewBeaconBlockAltair()
				b.Block.Slot = slot
				b.Block.ParentRoot = root
				return wrapper.WrappedAltairSignedBeaconBlock(b)
			},
		},
		{
			name: "bellatrix",
			newBlock: func(slot types.Slot, root []byte) (block.SignedBeaconBlock, error) {
				b := util.NewBeaconBlockMerge()
				b.Block.Slot = slot
				b.Block.ParentRoot = root
				return wrapper.WrappedMergeSignedBeaconBlock(b)
			},
		},
	}

	for _, tt := range tests {
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
			require.NoError(t, db.deleteBlock(ctx, blockRoot))
			assert.Equal(t, false, db.HasBlock(ctx, blockRoot), "Expected block to have been deleted from the db")
		})
	}
}

func TestStore_Blocks_FiltersCorrectly(t *testing.T) {
	db := setupDB(t)
	b4 := util.NewBeaconBlock()
	b4.Block.Slot = 4
	b4.Block.ParentRoot = bytesutil.PadTo([]byte("parent"), 32)
	b5 := util.NewBeaconBlock()
	b5.Block.Slot = 5
	b5.Block.ParentRoot = bytesutil.PadTo([]byte("parent2"), 32)
	b6 := util.NewBeaconBlock()
	b6.Block.Slot = 6
	b6.Block.ParentRoot = bytesutil.PadTo([]byte("parent2"), 32)
	b7 := util.NewBeaconBlock()
	b7.Block.Slot = 7
	b7.Block.ParentRoot = bytesutil.PadTo([]byte("parent3"), 32)
	b8 := util.NewBeaconBlock()
	b8.Block.Slot = 8
	b8.Block.ParentRoot = bytesutil.PadTo([]byte("parent4"), 32)
	blocks := []block.SignedBeaconBlock{
		wrapper.WrappedPhase0SignedBeaconBlock(b4),
		wrapper.WrappedPhase0SignedBeaconBlock(b5),
		wrapper.WrappedPhase0SignedBeaconBlock(b6),
		wrapper.WrappedPhase0SignedBeaconBlock(b7),
		wrapper.WrappedPhase0SignedBeaconBlock(b8),
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
	for _, tt := range tests {
		retrievedBlocks, _, err := db.Blocks(ctx, tt.filter)
		require.NoError(t, err)
		assert.Equal(t, tt.expectedNumBlocks, len(retrievedBlocks), "Unexpected number of blocks")
	}
}

func TestStore_Blocks_VerifyBlockRoots(t *testing.T) {
	ctx := context.Background()
	db := setupDB(t)
	b1 := util.NewBeaconBlock()
	b1.Block.Slot = 1
	r1, err := b1.Block.HashTreeRoot()
	require.NoError(t, err)
	b2 := util.NewBeaconBlock()
	b2.Block.Slot = 2
	r2, err := b2.Block.HashTreeRoot()
	require.NoError(t, err)

	require.NoError(t, db.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(b1)))
	require.NoError(t, db.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(b2)))

	filter := filters.NewFilter().SetStartSlot(b1.Block.Slot).SetEndSlot(b2.Block.Slot)
	roots, err := db.BlockRoots(ctx, filter)
	require.NoError(t, err)

	assert.DeepEqual(t, [][32]byte{r1, r2}, roots)
}

func TestStore_Blocks_Retrieve_SlotRange(t *testing.T) {
	db := setupDB(t)
	totalBlocks := make([]block.SignedBeaconBlock, 500)
	for i := 0; i < 500; i++ {
		b := util.NewBeaconBlock()
		b.Block.Slot = types.Slot(i)
		b.Block.ParentRoot = bytesutil.PadTo([]byte("parent"), 32)
		totalBlocks[i] = wrapper.WrappedPhase0SignedBeaconBlock(b)
	}
	ctx := context.Background()
	require.NoError(t, db.SaveBlocks(ctx, totalBlocks))
	retrieved, _, err := db.Blocks(ctx, filters.NewFilter().SetStartSlot(100).SetEndSlot(399))
	require.NoError(t, err)
	assert.Equal(t, 300, len(retrieved))
}

func TestStore_Blocks_Retrieve_Epoch(t *testing.T) {
	db := setupDB(t)
	slots := params.BeaconConfig().SlotsPerEpoch.Mul(7)
	totalBlocks := make([]block.SignedBeaconBlock, slots)
	for i := types.Slot(0); i < slots; i++ {
		b := util.NewBeaconBlock()
		b.Block.Slot = i
		b.Block.ParentRoot = bytesutil.PadTo([]byte("parent"), 32)
		totalBlocks[i] = wrapper.WrappedPhase0SignedBeaconBlock(b)
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
}

func TestStore_Blocks_Retrieve_SlotRangeWithStep(t *testing.T) {
	db := setupDB(t)
	totalBlocks := make([]block.SignedBeaconBlock, 500)
	for i := 0; i < 500; i++ {
		b := util.NewBeaconBlock()
		b.Block.Slot = types.Slot(i)
		b.Block.ParentRoot = bytesutil.PadTo([]byte("parent"), 32)
		totalBlocks[i] = wrapper.WrappedPhase0SignedBeaconBlock(b)
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
}

func TestStore_SaveBlock_CanGetHighestAt(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	block1 := util.NewBeaconBlock()
	block1.Block.Slot = 1
	require.NoError(t, db.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(block1)))
	block2 := util.NewBeaconBlock()
	block2.Block.Slot = 10
	require.NoError(t, db.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(block2)))
	block3 := util.NewBeaconBlock()
	block3.Block.Slot = 100
	require.NoError(t, db.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(block3)))

	highestAt, err := db.HighestSlotBlocksBelow(ctx, 2)
	require.NoError(t, err)
	assert.Equal(t, false, len(highestAt) <= 0, "Got empty highest at slice")
	assert.Equal(t, true, proto.Equal(block1, highestAt[0].Proto()), "Wanted: %v, received: %v", block1, highestAt[0])
	highestAt, err = db.HighestSlotBlocksBelow(ctx, 11)
	require.NoError(t, err)
	assert.Equal(t, false, len(highestAt) <= 0, "Got empty highest at slice")
	assert.Equal(t, true, proto.Equal(block2, highestAt[0].Proto()), "Wanted: %v, received: %v", block2, highestAt[0])
	highestAt, err = db.HighestSlotBlocksBelow(ctx, 101)
	require.NoError(t, err)
	assert.Equal(t, false, len(highestAt) <= 0, "Got empty highest at slice")
	assert.Equal(t, true, proto.Equal(block3, highestAt[0].Proto()), "Wanted: %v, received: %v", block3, highestAt[0])

	r3, err := block3.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, db.deleteBlock(ctx, r3))

	highestAt, err = db.HighestSlotBlocksBelow(ctx, 101)
	require.NoError(t, err)
	assert.Equal(t, true, proto.Equal(block2, highestAt[0].Proto()), "Wanted: %v, received: %v", block2, highestAt[0])
}

func TestStore_GenesisBlock_CanGetHighestAt(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	genesisBlock := util.NewBeaconBlock()
	genesisRoot, err := genesisBlock.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, db.SaveGenesisBlockRoot(ctx, genesisRoot))
	require.NoError(t, db.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(genesisBlock)))
	block1 := util.NewBeaconBlock()
	block1.Block.Slot = 1
	require.NoError(t, db.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(block1)))

	highestAt, err := db.HighestSlotBlocksBelow(ctx, 2)
	require.NoError(t, err)
	assert.Equal(t, true, proto.Equal(block1, highestAt[0].Proto()), "Wanted: %v, received: %v", block1, highestAt[0])
	highestAt, err = db.HighestSlotBlocksBelow(ctx, 1)
	require.NoError(t, err)
	assert.Equal(t, true, proto.Equal(genesisBlock, highestAt[0].Proto()), "Wanted: %v, received: %v", genesisBlock, highestAt[0])
	highestAt, err = db.HighestSlotBlocksBelow(ctx, 0)
	require.NoError(t, err)
	assert.Equal(t, true, proto.Equal(genesisBlock, highestAt[0].Proto()), "Wanted: %v, received: %v", genesisBlock, highestAt[0])
}

func TestStore_SaveBlocks_HasCachedBlocks(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	b := make([]block.SignedBeaconBlock, 500)
	for i := 0; i < 500; i++ {
		blk := util.NewBeaconBlock()
		blk.Block.ParentRoot = bytesutil.PadTo([]byte("parent"), 32)
		blk.Block.Slot = types.Slot(i)
		b[i] = wrapper.WrappedPhase0SignedBeaconBlock(blk)
	}

	require.NoError(t, db.SaveBlock(ctx, b[0]))
	require.NoError(t, db.SaveBlocks(ctx, b))
	f := filters.NewFilter().SetStartSlot(0).SetEndSlot(500)

	blks, _, err := db.Blocks(ctx, f)
	require.NoError(t, err)
	assert.Equal(t, 500, len(blks), "Did not get wanted blocks")
}

func TestStore_SaveBlocks_HasRootsMatched(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	b := make([]block.SignedBeaconBlock, 500)
	for i := 0; i < 500; i++ {
		blk := util.NewBeaconBlock()
		blk.Block.ParentRoot = bytesutil.PadTo([]byte("parent"), 32)
		blk.Block.Slot = types.Slot(i)
		b[i] = wrapper.WrappedPhase0SignedBeaconBlock(blk)
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
}

func TestStore_BlocksBySlot_BlockRootsBySlot(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	b1 := util.NewBeaconBlock()
	b1.Block.Slot = 20
	require.NoError(t, db.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(b1)))
	b2 := util.NewBeaconBlock()
	b2.Block.Slot = 100
	b2.Block.ParentRoot = bytesutil.PadTo([]byte("parent1"), 32)
	require.NoError(t, db.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(b2)))
	b3 := util.NewBeaconBlock()
	b3.Block.Slot = 100
	b3.Block.ParentRoot = bytesutil.PadTo([]byte("parent2"), 32)
	require.NoError(t, db.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(b3)))

	r1, err := b1.Block.HashTreeRoot()
	require.NoError(t, err)
	r2, err := b2.Block.HashTreeRoot()
	require.NoError(t, err)
	r3, err := b3.Block.HashTreeRoot()
	require.NoError(t, err)

	hasBlocks, retrievedBlocks, err := db.BlocksBySlot(ctx, 1)
	require.NoError(t, err)
	assert.Equal(t, 0, len(retrievedBlocks), "Unexpected number of blocks received, expected none")
	assert.Equal(t, false, hasBlocks, "Expected no blocks")
	hasBlocks, retrievedBlocks, err = db.BlocksBySlot(ctx, 20)
	require.NoError(t, err)
	assert.Equal(t, true, proto.Equal(b1, retrievedBlocks[0].Proto()), "Wanted: %v, received: %v", b1, retrievedBlocks[0])
	assert.Equal(t, true, hasBlocks, "Expected to have blocks")
	hasBlocks, retrievedBlocks, err = db.BlocksBySlot(ctx, 100)
	require.NoError(t, err)
	assert.Equal(t, true, proto.Equal(b2, retrievedBlocks[0].Proto()), "Wanted: %v, received: %v", b2, retrievedBlocks[0])
	assert.Equal(t, true, proto.Equal(b3, retrievedBlocks[1].Proto()), "Wanted: %v, received: %v", b3, retrievedBlocks[1])
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
}
