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
)

func TestStore_SaveBlock_NoDuplicates(t *testing.T) {
	BlockCacheSize = 1
	db := setupDB(t)
	slot := uint64(20)
	ctx := context.Background()
	// First we save a previous block to ensure the cache max size is reached.
	prevBlock := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			Slot:       slot - 1,
			ParentRoot: bytesutil.PadTo([]byte{1, 2, 3}, 32),
		},
	}
	if err := db.SaveBlock(ctx, prevBlock); err != nil {
		t.Fatal(err)
	}
	block := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			Slot:       slot,
			ParentRoot: bytesutil.PadTo([]byte{1, 2, 3}, 32),
		},
	}
	// Even with a full cache, saving new blocks should not cause
	// duplicated blocks in the DB.
	for i := 0; i < 100; i++ {
		if err := db.SaveBlock(ctx, block); err != nil {
			t.Fatal(err)
		}
	}
	f := filters.NewFilter().SetStartSlot(slot).SetEndSlot(slot)
	retrieved, err := db.Blocks(ctx, f)
	if err != nil {
		t.Fatal(err)
	}
	if len(retrieved) != 1 {
		t.Errorf("Expected 1, received %d: %v", len(retrieved), retrieved)
	}
	// We reset the block cache size.
	BlockCacheSize = 256
}

func TestStore_BlocksCRUD(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	block := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			Slot:       20,
			ParentRoot: bytesutil.PadTo([]byte{1, 2, 3}, 32),
		},
	}
	blockRoot, err := stateutil.BlockRoot(block.Block)
	if err != nil {
		t.Fatal(err)
	}
	retrievedBlock, err := db.Block(ctx, blockRoot)
	if err != nil {
		t.Fatal(err)
	}
	if retrievedBlock != nil {
		t.Errorf("Expected nil block, received %v", retrievedBlock)
	}
	if err := db.SaveBlock(ctx, block); err != nil {
		t.Fatal(err)
	}
	if !db.HasBlock(ctx, blockRoot) {
		t.Error("Expected block to exist in the db")
	}
	retrievedBlock, err = db.Block(ctx, blockRoot)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(block, retrievedBlock) {
		t.Errorf("Wanted %v, received %v", block, retrievedBlock)
	}
	if err := db.deleteBlock(ctx, blockRoot); err != nil {
		t.Fatal(err)
	}
	if db.HasBlock(ctx, blockRoot) {
		t.Error("Expected block to have been deleted from the db")
	}
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
			if err != nil {
				t.Fatal(err)
			}
			blockRoots = append(blockRoots, r)
		} else {
			oddBlocks = append(oddBlocks, totalBlocks[i])
		}
	}
	if err := db.SaveBlocks(ctx, totalBlocks); err != nil {
		t.Fatal(err)
	}
	retrieved, err := db.Blocks(ctx, filters.NewFilter().SetParentRoot(bytesutil.PadTo([]byte("parent"), 32)))
	if err != nil {
		t.Fatal(err)
	}
	if len(retrieved) != numBlocks {
		t.Errorf("Received %d blocks, wanted 1000", len(retrieved))
	}
	// We delete all even indexed blocks.
	if err := db.deleteBlocks(ctx, blockRoots); err != nil {
		t.Fatal(err)
	}
	// When we retrieve the data, only the odd indexed blocks should remain.
	retrieved, err = db.Blocks(ctx, filters.NewFilter().SetParentRoot(bytesutil.PadTo([]byte("parent"), 32)))
	if err != nil {
		t.Fatal(err)
	}
	sort.Slice(retrieved, func(i, j int) bool {
		return retrieved[i].Block.Slot < retrieved[j].Block.Slot
	})
	for i, block := range retrieved {
		if !proto.Equal(block, oddBlocks[i]) {
			t.Errorf("Wanted %v, received %v", oddBlocks[i], block)
		}
	}
}

func TestStore_GenesisBlock(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	genesisBlock := testutil.NewBeaconBlock()
	genesisBlock.Block.ParentRoot = bytesutil.PadTo([]byte{1, 2, 3}, 32)
	blockRoot, err := stateutil.BlockRoot(genesisBlock.Block)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SaveGenesisBlockRoot(ctx, blockRoot); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveBlock(ctx, genesisBlock); err != nil {
		t.Fatal(err)
	}
	retrievedBlock, err := db.GenesisBlock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(genesisBlock, retrievedBlock) {
		t.Errorf("Wanted %v, received %v", genesisBlock, retrievedBlock)
	}
}

func TestStore_BlocksCRUD_NoCache(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	block := testutil.NewBeaconBlock()
	block.Block.Slot = 20
	block.Block.ParentRoot = bytesutil.PadTo([]byte{1, 2, 3}, 32)
	blockRoot, err := stateutil.BlockRoot(block.Block)
	if err != nil {
		t.Fatal(err)
	}
	retrievedBlock, err := db.Block(ctx, blockRoot)
	if err != nil {
		t.Fatal(err)
	}
	if retrievedBlock != nil {
		t.Errorf("Expected nil block, received %v", retrievedBlock)
	}
	if err := db.SaveBlock(ctx, block); err != nil {
		t.Fatal(err)
	}
	db.blockCache.Del(string(blockRoot[:]))
	if !db.HasBlock(ctx, blockRoot) {
		t.Error("Expected block to exist in the db")
	}
	retrievedBlock, err = db.Block(ctx, blockRoot)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(block, retrievedBlock) {
		t.Errorf("Wanted %v, received %v", block, retrievedBlock)
	}
	if err := db.deleteBlock(ctx, blockRoot); err != nil {
		t.Fatal(err)
	}
	if db.HasBlock(ctx, blockRoot) {
		t.Error("Expected block to have been deleted from the db")
	}
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
	if err := db.SaveBlocks(ctx, blocks); err != nil {
		t.Fatal(err)
	}

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
		if err != nil {
			t.Fatal(err)
		}
		if len(retrievedBlocks) != tt.expectedNumBlocks {
			t.Errorf("Expected %d blocks, received %d", tt.expectedNumBlocks, len(retrievedBlocks))
		}
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
	if err := db.SaveBlocks(ctx, totalBlocks); err != nil {
		t.Fatal(err)
	}
	retrieved, err := db.Blocks(ctx, filters.NewFilter().SetStartSlot(100).SetEndSlot(399))
	if err != nil {
		t.Fatal(err)
	}
	want := 300
	if len(retrieved) != want {
		t.Errorf("Wanted %d, received %d", want, len(retrieved))
	}
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
	if err := db.SaveBlocks(ctx, totalBlocks); err != nil {
		t.Fatal(err)
	}
	retrieved, err := db.Blocks(ctx, filters.NewFilter().SetStartEpoch(5).SetEndEpoch(6))
	if err != nil {
		t.Fatal(err)
	}
	want := params.BeaconConfig().SlotsPerEpoch * 2
	if uint64(len(retrieved)) != want {
		t.Errorf("Wanted %d, received %d", want, len(retrieved))
	}
	retrieved, err = db.Blocks(ctx, filters.NewFilter().SetStartEpoch(0).SetEndEpoch(0))
	if err != nil {
		t.Fatal(err)
	}
	want = params.BeaconConfig().SlotsPerEpoch
	if uint64(len(retrieved)) != want {
		t.Errorf("Wanted %d, received %d", want, len(retrieved))
	}
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
	if err := db.SaveBlocks(ctx, totalBlocks); err != nil {
		t.Fatal(err)
	}
	retrieved, err := db.Blocks(ctx, filters.NewFilter().SetStartSlot(100).SetEndSlot(399).SetSlotStep(step))
	if err != nil {
		t.Fatal(err)
	}
	want := 150
	if len(retrieved) != want {
		t.Errorf("Wanted %d, received %d", want, len(retrieved))
	}
	for _, b := range retrieved {
		if (b.Block.Slot-100)%step != 0 {
			t.Errorf("Unexpect block slot %d", b.Block.Slot)
		}
	}
}

func TestStore_SaveBlock_CanGetHighest(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	block := testutil.NewBeaconBlock()
	block.Block.Slot = 1
	if err := db.SaveBlock(ctx, block); err != nil {
		t.Fatal(err)
	}
	highestSavedBlock, err := db.HighestSlotBlock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(block, highestSavedBlock) {
		t.Errorf("Wanted %v, received %v", block, highestSavedBlock)
	}

	block.Block.Slot = 999
	if err := db.SaveBlock(ctx, block); err != nil {
		t.Fatal(err)
	}
	highestSavedBlock, err = db.HighestSlotBlock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(block, highestSavedBlock) {
		t.Errorf("Wanted %v, received %v", block, highestSavedBlock)
	}

	block.Block.Slot = 300000000
	if err := db.SaveBlock(ctx, block); err != nil {
		t.Fatal(err)
	}
	highestSavedBlock, err = db.HighestSlotBlock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(block, highestSavedBlock) {
		t.Errorf("Wanted %v, received %v", block, highestSavedBlock)
	}
}

func TestStore_SaveBlock_CanGetHighestAt(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	block1 := testutil.NewBeaconBlock()
	block1.Block.Slot = 1
	if err := db.SaveBlock(ctx, block1); err != nil {
		t.Fatal(err)
	}
	block2 := testutil.NewBeaconBlock()
	block2.Block.Slot = 10
	if err := db.SaveBlock(ctx, block2); err != nil {
		t.Fatal(err)
	}
	block3 := testutil.NewBeaconBlock()
	block3.Block.Slot = 100
	if err := db.SaveBlock(ctx, block3); err != nil {
		t.Fatal(err)
	}

	highestAt, err := db.HighestSlotBlocksBelow(ctx, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(highestAt) <= 0 {
		t.Fatal("Got empty highest at slice")
	}
	if !proto.Equal(block1, highestAt[0]) {
		t.Errorf("Wanted %v, received %v", block1, highestAt)
	}
	highestAt, err = db.HighestSlotBlocksBelow(ctx, 11)
	if err != nil {
		t.Fatal(err)
	}
	if len(highestAt) <= 0 {
		t.Fatal("Got empty highest at slice")
	}
	if !proto.Equal(block2, highestAt[0]) {
		t.Errorf("Wanted %v, received %v", block2, highestAt)
	}
	highestAt, err = db.HighestSlotBlocksBelow(ctx, 101)
	if err != nil {
		t.Fatal(err)
	}
	if len(highestAt) <= 0 {
		t.Fatal("Got empty highest at slice")
	}
	if !proto.Equal(block3, highestAt[0]) {
		t.Errorf("Wanted %v, received %v", block3, highestAt)
	}

	r3, err := stateutil.BlockRoot(block3.Block)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.deleteBlock(ctx, r3); err != nil {
		t.Fatal(err)
	}

	highestAt, err = db.HighestSlotBlocksBelow(ctx, 101)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(block2, highestAt[0]) {
		t.Errorf("Wanted %v, received %v", block2, highestAt)
	}
}

func TestStore_GenesisBlock_CanGetHighestAt(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	genesisBlock := testutil.NewBeaconBlock()
	genesisRoot, err := stateutil.BlockRoot(genesisBlock.Block)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SaveGenesisBlockRoot(ctx, genesisRoot); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveBlock(ctx, genesisBlock); err != nil {
		t.Fatal(err)
	}
	block1 := testutil.NewBeaconBlock()
	block1.Block.Slot = 1
	if err := db.SaveBlock(ctx, block1); err != nil {
		t.Fatal(err)
	}

	highestAt, err := db.HighestSlotBlocksBelow(ctx, 2)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(block1, highestAt[0]) {
		t.Errorf("Wanted %v, received %v", block1, highestAt)
	}
	highestAt, err = db.HighestSlotBlocksBelow(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(genesisBlock, highestAt[0]) {
		t.Errorf("Wanted %v, received %v", genesisBlock, highestAt)
	}
	highestAt, err = db.HighestSlotBlocksBelow(ctx, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(genesisBlock, highestAt[0]) {
		t.Errorf("Wanted %v, received %v", genesisBlock, highestAt)
	}
}

func TestStore_SaveBlocks_CanGetHighest(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	totalBlocks := make([]*ethpb.SignedBeaconBlock, 500)
	for i := 0; i < 500; i++ {
		b := testutil.NewBeaconBlock()
		b.Block.Slot = uint64(i)
		b.Block.ParentRoot = bytesutil.PadTo([]byte("parent"), 32)
		totalBlocks[i] = b
	}

	if err := db.SaveBlocks(ctx, totalBlocks); err != nil {
		t.Fatal(err)
	}
	highestSavedBlock, err := db.HighestSlotBlock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(totalBlocks[len(totalBlocks)-1], highestSavedBlock) {
		t.Errorf("Wanted %v, received %v", totalBlocks[len(totalBlocks)-1], highestSavedBlock)
	}
}

func TestStore_SaveBlocks_HasCachedBlocks(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	b := make([]*ethpb.SignedBeaconBlock, 500)
	for i := 0; i < 500; i++ {
		b[i] = &ethpb.SignedBeaconBlock{
			Block: &ethpb.BeaconBlock{
				ParentRoot: bytesutil.PadTo([]byte("parent"), 32),
				Slot:       uint64(i),
			},
		}
	}

	if err := db.SaveBlock(ctx, b[0]); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveBlocks(ctx, b); err != nil {
		t.Fatal(err)
	}
	f := filters.NewFilter().SetStartSlot(0).SetEndSlot(500)

	blks, err := db.Blocks(ctx, f)
	if err != nil {
		t.Fatal(err)
	}
	if len(blks) != 500 {
		t.Log(len(blks))
		t.Error("Did not get wanted blocks")
	}
}

func TestStore_DeleteBlock_CanGetHighest(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	b50 := testutil.NewBeaconBlock()
	b50.Block.Slot = 50
	if err := db.SaveBlock(ctx, b50); err != nil {
		t.Fatal(err)
	}
	highestSavedBlock, err := db.HighestSlotBlock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(b50, highestSavedBlock) {
		t.Errorf("Wanted %v, received %v", b50, highestSavedBlock)
	}

	b51 := testutil.NewBeaconBlock()
	b51.Block.Slot = 51
	r51, err := stateutil.BlockRoot(b51.Block)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SaveBlock(ctx, b51); err != nil {
		t.Fatal(err)
	}

	highestSavedBlock, err = db.HighestSlotBlock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(b51, highestSavedBlock) {
		t.Errorf("Wanted %v, received %v", b51, highestSavedBlock)
	}

	if err := db.deleteBlock(ctx, r51); err != nil {
		t.Fatal(err)
	}
	highestSavedBlock, err = db.HighestSlotBlock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(b50, highestSavedBlock) {
		t.Errorf("Wanted %v, received %v", b50, highestSavedBlock)
	}
}

func TestStore_DeleteBlocks_CanGetHighest(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	var err error
	totalBlocks := make([]*ethpb.SignedBeaconBlock, 100)
	r := make([][32]byte, 100)
	for i := 0; i < 100; i++ {
		b := testutil.NewBeaconBlock()
		b.Block.Slot = uint64(i)
		b.Block.ParentRoot = bytesutil.PadTo([]byte("parent"), 32)
		totalBlocks[i] = b
		r[i], err = stateutil.BlockRoot(totalBlocks[i].Block)
		if err != nil {
			t.Error(err)
		}
	}

	if err := db.SaveBlocks(ctx, totalBlocks); err != nil {
		t.Fatal(err)
	}
	if err := db.deleteBlocks(ctx, [][32]byte{r[99], r[98], r[97]}); err != nil {
		t.Fatal(err)
	}
	highestSavedBlock, err := db.HighestSlotBlock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(totalBlocks[96], highestSavedBlock) {
		t.Errorf("Wanted %v, received %v", totalBlocks[len(totalBlocks)-1], highestSavedBlock)
	}
}
