package kv

import (
	"context"
	"sort"
	"testing"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestStore_SaveBlock_NoDuplicates(t *testing.T) {
	BlockCacheSize = 1
	db := setupDB(t)
	slot := uint64(20)
	ctx := context.Background()
	// First we save a previous block to ensure the cache max size is reached.
	prevBlock := testutil.NewBeaconBlock()
	prevBlock.Block.Slot = slot - 1
	prevBlock.Block.ParentRoot = bytesutil.PadTo([]byte{1, 2, 3}, 32)
	require.NoError(t, db.SaveBlock(ctx, prevBlock))

	block := testutil.NewBeaconBlock()
	block.Block.Slot = slot
	block.Block.ParentRoot = bytesutil.PadTo([]byte{1, 2, 3}, 32)
	// Even with a full cache, saving new blocks should not cause
	// duplicated blocks in the DB.
	for i := 0; i < 100; i++ {
		require.NoError(t, db.SaveBlock(ctx, block))
	}
	f := filters.NewFilter().SetStartSlot(slot).SetEndSlot(slot)
	retrieved, err := db.Blocks(ctx, f)
	require.NoError(t, err)
	assert.Equal(t, 1, len(retrieved))
	// We reset the block cache size.
	BlockCacheSize = 256
}

func TestStore_BlocksCRUD(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	block := testutil.NewBeaconBlock()
	block.Block.Slot = 20
	block.Block.ParentRoot = bytesutil.PadTo([]byte{1, 2, 3}, 32)

	blockRoot, err := stateutil.BlockRoot(block.Block)
	require.NoError(t, err)
	retrievedBlock, err := db.Block(ctx, blockRoot)
	require.NoError(t, err)
	assert.Equal(t, (*ethpb.SignedBeaconBlock)(nil), retrievedBlock, "Expected nil block")
	require.NoError(t, db.SaveBlock(ctx, block))
	assert.Equal(t, true, db.HasBlock(ctx, blockRoot), "Expected block to exist in the db")
	retrievedBlock, err = db.Block(ctx, blockRoot)
	require.NoError(t, err)
	assert.Equal(t, true, proto.Equal(block, retrievedBlock), "Wanted: %v, received: %v", block, retrievedBlock)
	require.NoError(t, db.deleteBlock(ctx, blockRoot))
	assert.Equal(t, false, db.HasBlock(ctx, blockRoot), "Expected block to have been deleted from the db")
}

func TestStore_BlocksBatchDelete(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	numBlocks := 10
	totalBlocks := make([]*ethpb.SignedBeaconBlock, numBlocks)
	blockRoots := make([][32]byte, 0)
	oddBlocks := make([]*ethpb.SignedBeaconBlock, 0)
	for i := 0; i < len(totalBlocks); i++ {
		b := testutil.NewBeaconBlock()
		b.Block.Slot = uint64(i)
		b.Block.ParentRoot = bytesutil.PadTo([]byte("parent"), 32)
		totalBlocks[i] = b
		if i%2 == 0 {
			r, err := stateutil.BlockRoot(totalBlocks[i].Block)
			require.NoError(t, err)
			blockRoots = append(blockRoots, r)
		} else {
			oddBlocks = append(oddBlocks, totalBlocks[i])
		}
	}
	require.NoError(t, db.SaveBlocks(ctx, totalBlocks))
	retrieved, err := db.Blocks(ctx, filters.NewFilter().SetParentRoot(bytesutil.PadTo([]byte("parent"), 32)))
	require.NoError(t, err)
	assert.Equal(t, numBlocks, len(retrieved), "Unexpected number of blocks received")
	// We delete all even indexed blocks.
	require.NoError(t, db.deleteBlocks(ctx, blockRoots))
	// When we retrieve the data, only the odd indexed blocks should remain.
	retrieved, err = db.Blocks(ctx, filters.NewFilter().SetParentRoot(bytesutil.PadTo([]byte("parent"), 32)))
	require.NoError(t, err)
	sort.Slice(retrieved, func(i, j int) bool {
		return retrieved[i].Block.Slot < retrieved[j].Block.Slot
	})
	for i, block := range retrieved {
		assert.Equal(t, true, proto.Equal(block, oddBlocks[i]), "Wanted: %v, received: %v", block, oddBlocks[i])
	}
}

func TestStore_GenesisBlock(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	genesisBlock := testutil.NewBeaconBlock()
	genesisBlock.Block.ParentRoot = bytesutil.PadTo([]byte{1, 2, 3}, 32)
	blockRoot, err := stateutil.BlockRoot(genesisBlock.Block)
	require.NoError(t, err)
	require.NoError(t, db.SaveGenesisBlockRoot(ctx, blockRoot))
	require.NoError(t, db.SaveBlock(ctx, genesisBlock))
	retrievedBlock, err := db.GenesisBlock(ctx)
	require.NoError(t, err)
	assert.Equal(t, true, proto.Equal(genesisBlock, retrievedBlock), "Wanted: %v, received: %v", genesisBlock, retrievedBlock)
}

func TestStore_BlocksCRUD_NoCache(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	block := testutil.NewBeaconBlock()
	block.Block.Slot = 20
	block.Block.ParentRoot = bytesutil.PadTo([]byte{1, 2, 3}, 32)
	blockRoot, err := stateutil.BlockRoot(block.Block)
	require.NoError(t, err)
	retrievedBlock, err := db.Block(ctx, blockRoot)
	require.NoError(t, err)
	require.Equal(t, (*ethpb.SignedBeaconBlock)(nil), retrievedBlock, "Expected nil block")
	require.NoError(t, db.SaveBlock(ctx, block))
	db.blockCache.Del(string(blockRoot[:]))
	assert.Equal(t, true, db.HasBlock(ctx, blockRoot), "Expected block to exist in the db")
	retrievedBlock, err = db.Block(ctx, blockRoot)
	require.NoError(t, err)
	assert.Equal(t, true, proto.Equal(block, retrievedBlock), "Wanted: %v, received: %v", block, retrievedBlock)
	require.NoError(t, db.deleteBlock(ctx, blockRoot))
	assert.Equal(t, false, db.HasBlock(ctx, blockRoot), "Expected block to have been deleted from the db")
}

func TestStore_Blocks_FiltersCorrectly(t *testing.T) {
	db := setupDB(t)
	b4 := testutil.NewBeaconBlock()
	b4.Block.Slot = 4
	b4.Block.ParentRoot = bytesutil.PadTo([]byte("parent"), 32)
	b5 := testutil.NewBeaconBlock()
	b5.Block.Slot = 5
	b5.Block.ParentRoot = bytesutil.PadTo([]byte("parent2"), 32)
	b6 := testutil.NewBeaconBlock()
	b6.Block.Slot = 6
	b6.Block.ParentRoot = bytesutil.PadTo([]byte("parent2"), 32)
	b7 := testutil.NewBeaconBlock()
	b7.Block.Slot = 7
	b7.Block.ParentRoot = bytesutil.PadTo([]byte("parent3"), 32)
	b8 := testutil.NewBeaconBlock()
	b8.Block.Slot = 8
	b8.Block.ParentRoot = bytesutil.PadTo([]byte("parent4"), 32)
	blocks := []*ethpb.SignedBeaconBlock{b4, b5, b6, b7, b8}
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
			filter:            filters.NewFilter().SetStartSlot(5),
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
		retrievedBlocks, err := db.Blocks(ctx, tt.filter)
		require.NoError(t, err)
		assert.Equal(t, tt.expectedNumBlocks, len(retrievedBlocks), "Unexpected number of blocks")
	}
}

func TestStore_Blocks_Retrieve_SlotRange(t *testing.T) {
	db := setupDB(t)
	totalBlocks := make([]*ethpb.SignedBeaconBlock, 500)
	for i := 0; i < 500; i++ {
		b := testutil.NewBeaconBlock()
		b.Block.Slot = uint64(i)
		b.Block.ParentRoot = bytesutil.PadTo([]byte("parent"), 32)
		totalBlocks[i] = b
	}
	ctx := context.Background()
	require.NoError(t, db.SaveBlocks(ctx, totalBlocks))
	retrieved, err := db.Blocks(ctx, filters.NewFilter().SetStartSlot(100).SetEndSlot(399))
	require.NoError(t, err)
	assert.Equal(t, 300, len(retrieved))
}

func TestStore_Blocks_Retrieve_Epoch(t *testing.T) {
	db := setupDB(t)
	slots := params.BeaconConfig().SlotsPerEpoch * 7
	totalBlocks := make([]*ethpb.SignedBeaconBlock, slots)
	for i := uint64(0); i < slots; i++ {
		b := testutil.NewBeaconBlock()
		b.Block.Slot = i
		b.Block.ParentRoot = bytesutil.PadTo([]byte("parent"), 32)
		totalBlocks[i] = b
	}
	ctx := context.Background()
	require.NoError(t, db.SaveBlocks(ctx, totalBlocks))
	retrieved, err := db.Blocks(ctx, filters.NewFilter().SetStartEpoch(5).SetEndEpoch(6))
	require.NoError(t, err)
	want := params.BeaconConfig().SlotsPerEpoch * 2
	assert.Equal(t, want, uint64(len(retrieved)))
	retrieved, err = db.Blocks(ctx, filters.NewFilter().SetStartEpoch(0).SetEndEpoch(0))
	require.NoError(t, err)
	want = params.BeaconConfig().SlotsPerEpoch
	assert.Equal(t, want, uint64(len(retrieved)))
}

func TestStore_Blocks_Retrieve_SlotRangeWithStep(t *testing.T) {
	db := setupDB(t)
	totalBlocks := make([]*ethpb.SignedBeaconBlock, 500)
	for i := 0; i < 500; i++ {
		b := testutil.NewBeaconBlock()
		b.Block.Slot = uint64(i)
		b.Block.ParentRoot = bytesutil.PadTo([]byte("parent"), 32)
		totalBlocks[i] = b
	}
	const step = 2
	ctx := context.Background()
	require.NoError(t, db.SaveBlocks(ctx, totalBlocks))
	retrieved, err := db.Blocks(ctx, filters.NewFilter().SetStartSlot(100).SetEndSlot(399).SetSlotStep(step))
	require.NoError(t, err)
	assert.Equal(t, 150, len(retrieved))
	for _, b := range retrieved {
		assert.Equal(t, uint64(0), (b.Block.Slot-100)%step, "Unexpect block slot %d", b.Block.Slot)
	}
}

func TestStore_SaveBlock_CanGetHighestAt(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	block1 := testutil.NewBeaconBlock()
	block1.Block.Slot = 1
	require.NoError(t, db.SaveBlock(ctx, block1))
	block2 := testutil.NewBeaconBlock()
	block2.Block.Slot = 10
	require.NoError(t, db.SaveBlock(ctx, block2))
	block3 := testutil.NewBeaconBlock()
	block3.Block.Slot = 100
	require.NoError(t, db.SaveBlock(ctx, block3))

	highestAt, err := db.HighestSlotBlocksBelow(ctx, 2)
	require.NoError(t, err)
	assert.Equal(t, false, len(highestAt) <= 0, "Got empty highest at slice")
	assert.Equal(t, true, proto.Equal(block1, highestAt[0]), "Wanted: %v, received: %v", block1, highestAt[0])
	highestAt, err = db.HighestSlotBlocksBelow(ctx, 11)
	require.NoError(t, err)
	assert.Equal(t, false, len(highestAt) <= 0, "Got empty highest at slice")
	assert.Equal(t, true, proto.Equal(block2, highestAt[0]), "Wanted: %v, received: %v", block2, highestAt[0])
	highestAt, err = db.HighestSlotBlocksBelow(ctx, 101)
	require.NoError(t, err)
	assert.Equal(t, false, len(highestAt) <= 0, "Got empty highest at slice")
	assert.Equal(t, true, proto.Equal(block3, highestAt[0]), "Wanted: %v, received: %v", block3, highestAt[0])

	r3, err := stateutil.BlockRoot(block3.Block)
	require.NoError(t, err)
	require.NoError(t, db.deleteBlock(ctx, r3))

	highestAt, err = db.HighestSlotBlocksBelow(ctx, 101)
	require.NoError(t, err)
	assert.Equal(t, true, proto.Equal(block2, highestAt[0]), "Wanted: %v, received: %v", block2, highestAt[0])
}

func TestStore_GenesisBlock_CanGetHighestAt(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	genesisBlock := testutil.NewBeaconBlock()
	genesisRoot, err := stateutil.BlockRoot(genesisBlock.Block)
	require.NoError(t, err)
	require.NoError(t, db.SaveGenesisBlockRoot(ctx, genesisRoot))
	require.NoError(t, db.SaveBlock(ctx, genesisBlock))
	block1 := testutil.NewBeaconBlock()
	block1.Block.Slot = 1
	require.NoError(t, db.SaveBlock(ctx, block1))

	highestAt, err := db.HighestSlotBlocksBelow(ctx, 2)
	require.NoError(t, err)
	assert.Equal(t, true, proto.Equal(block1, highestAt[0]), "Wanted: %v, received: %v", block1, highestAt[0])
	highestAt, err = db.HighestSlotBlocksBelow(ctx, 1)
	require.NoError(t, err)
	assert.Equal(t, true, proto.Equal(genesisBlock, highestAt[0]), "Wanted: %v, received: %v", genesisBlock, highestAt[0])
	highestAt, err = db.HighestSlotBlocksBelow(ctx, 0)
	require.NoError(t, err)
	assert.Equal(t, true, proto.Equal(genesisBlock, highestAt[0]), "Wanted: %v, received: %v", genesisBlock, highestAt[0])
}

func TestStore_SaveBlocks_HasCachedBlocks(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	b := make([]*ethpb.SignedBeaconBlock, 500)
	for i := 0; i < 500; i++ {
		blk := testutil.NewBeaconBlock()
		blk.Block.ParentRoot = bytesutil.PadTo([]byte("parent"), 32)
		blk.Block.Slot = uint64(i)
		b[i] = blk
	}

	require.NoError(t, db.SaveBlock(ctx, b[0]))
	require.NoError(t, db.SaveBlocks(ctx, b))
	f := filters.NewFilter().SetStartSlot(0).SetEndSlot(500)

	blks, err := db.Blocks(ctx, f)
	require.NoError(t, err)
	assert.Equal(t, 500, len(blks), "Did not get wanted blocks")
}
