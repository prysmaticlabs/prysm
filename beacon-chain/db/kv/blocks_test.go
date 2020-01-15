package kv

import (
	"context"
	"reflect"
	"sort"
	"testing"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestStore_SaveBlock_NoDuplicates(t *testing.T) {
	BlockCacheSize = 1
	db := setupDB(t)
	defer teardownDB(t, db)
	slot := uint64(20)
	ctx := context.Background()
	// First we save a previous block to ensure the cache max size is reached.
	prevBlock := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			Slot:       slot - 1,
			ParentRoot: []byte{1, 2, 3},
		},
	}
	if err := db.SaveBlock(ctx, prevBlock); err != nil {
		t.Fatal(err)
	}
	block := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			Slot:       slot,
			ParentRoot: []byte{1, 2, 3},
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
	defer teardownDB(t, db)
	ctx := context.Background()
	block := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			Slot:       20,
			ParentRoot: []byte{1, 2, 3},
		},
	}
	blockRoot, err := ssz.HashTreeRoot(block.Block)
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
	if err := db.DeleteBlock(ctx, blockRoot); err != nil {
		t.Fatal(err)
	}
	if db.HasBlock(ctx, blockRoot) {
		t.Error("Expected block to have been deleted from the db")
	}
}

func TestStore_BlocksBatchDelete(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)
	ctx := context.Background()
	numBlocks := 1000
	totalBlocks := make([]*ethpb.SignedBeaconBlock, numBlocks)
	blockRoots := make([][32]byte, 0)
	oddBlocks := make([]*ethpb.SignedBeaconBlock, 0)
	for i := 0; i < len(totalBlocks); i++ {
		totalBlocks[i] = &ethpb.SignedBeaconBlock{
			Block: &ethpb.BeaconBlock{
				Slot:       uint64(i),
				ParentRoot: []byte("parent"),
			},
		}

		if i%2 == 0 {
			r, err := ssz.HashTreeRoot(totalBlocks[i].Block)
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
	retrieved, err := db.Blocks(ctx, filters.NewFilter().SetParentRoot([]byte("parent")))
	if err != nil {
		t.Fatal(err)
	}
	if len(retrieved) != numBlocks {
		t.Errorf("Received %d blocks, wanted 1000", len(retrieved))
	}
	// We delete all even indexed blocks.
	if err := db.DeleteBlocks(ctx, blockRoots); err != nil {
		t.Fatal(err)
	}
	// When we retrieve the data, only the odd indexed blocks should remain.
	retrieved, err = db.Blocks(ctx, filters.NewFilter().SetParentRoot([]byte("parent")))
	if err != nil {
		t.Fatal(err)
	}
	sort.Slice(retrieved, func(i, j int) bool {
		return retrieved[i].Block.Slot < retrieved[j].Block.Slot
	})
	if !reflect.DeepEqual(retrieved, oddBlocks) {
		t.Errorf("Wanted %v, received %v", oddBlocks, retrieved)
	}
}

func TestStore_GenesisBlock(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)
	ctx := context.Background()
	genesisBlock := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			Slot:       0,
			ParentRoot: []byte{1, 2, 3},
		},
	}
	blockRoot, err := ssz.HashTreeRoot(genesisBlock.Block)
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
	defer teardownDB(t, db)
	ctx := context.Background()
	block := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			Slot:       20,
			ParentRoot: []byte{1, 2, 3},
		},
	}
	blockRoot, err := ssz.HashTreeRoot(block.Block)
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
	if err := db.DeleteBlock(ctx, blockRoot); err != nil {
		t.Fatal(err)
	}
	if db.HasBlock(ctx, blockRoot) {
		t.Error("Expected block to have been deleted from the db")
	}
}

func TestStore_Blocks_FiltersCorrectly(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)
	blocks := []*ethpb.SignedBeaconBlock{
		{
			Block: &ethpb.BeaconBlock{
				Slot:       4,
				ParentRoot: []byte("parent"),
			},
		},
		{
			Block: &ethpb.BeaconBlock{
				Slot:       5,
				ParentRoot: []byte("parent2"),
			},
		},
		{
			Block: &ethpb.BeaconBlock{
				Slot:       6,
				ParentRoot: []byte("parent2"),
			},
		},
		{
			Block: &ethpb.BeaconBlock{
				Slot:       7,
				ParentRoot: []byte("parent3"),
			},
		},
		{
			Block: &ethpb.BeaconBlock{
				Slot:       8,
				ParentRoot: []byte("parent4"),
			},
		},
	}
	ctx := context.Background()
	if err := db.SaveBlocks(ctx, blocks); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		filter            *filters.QueryFilter
		expectedNumBlocks int
	}{
		{
			filter:            filters.NewFilter().SetParentRoot([]byte("parent2")),
			expectedNumBlocks: 2,
		},
		{
			// No block meets the criteria below.
			filter:            filters.NewFilter().SetParentRoot([]byte{3, 4, 5}),
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
				SetParentRoot([]byte("parent2")).
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
	defer teardownDB(t, db)
	b := make([]*ethpb.SignedBeaconBlock, 500)
	for i := 0; i < 500; i++ {
		b[i] = &ethpb.SignedBeaconBlock{
			Block: &ethpb.BeaconBlock{
				ParentRoot: []byte("parent"),
				Slot:       uint64(i),
			},
		}
	}
	ctx := context.Background()
	if err := db.SaveBlocks(ctx, b); err != nil {
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
	defer teardownDB(t, db)
	slots := params.BeaconConfig().SlotsPerEpoch * 7
	b := make([]*ethpb.SignedBeaconBlock, slots)
	for i := uint64(0); i < slots; i++ {
		b[i] = &ethpb.SignedBeaconBlock{
			Block: &ethpb.BeaconBlock{
				ParentRoot: []byte("parent"),
				Slot:       i,
			},
		}
	}
	ctx := context.Background()
	if err := db.SaveBlocks(ctx, b); err != nil {
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
	defer teardownDB(t, db)
	b := make([]*ethpb.SignedBeaconBlock, 500)
	for i := 0; i < 500; i++ {
		b[i] = &ethpb.SignedBeaconBlock{
			Block: &ethpb.BeaconBlock{
				ParentRoot: []byte("parent"),
				Slot:       uint64(i),
			},
		}
	}
	const step = 2
	ctx := context.Background()
	if err := db.SaveBlocks(ctx, b); err != nil {
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
