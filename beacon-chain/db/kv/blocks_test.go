package kv

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
)

func TestStore_BlocksCRUD(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)
	ctx := context.Background()
	block := &ethpb.BeaconBlock{
		Slot:       20,
		ParentRoot: []byte{1, 2, 3},
	}
	blockRoot, err := ssz.SigningRoot(block)
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

func TestStore_BlocksCRUD_NoCache(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)
	ctx := context.Background()
	block := &ethpb.BeaconBlock{
		Slot:       20,
		ParentRoot: []byte{1, 2, 3},
	}
	blockRoot, err := ssz.SigningRoot(block)
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
	db.blockCache.Delete(string(blockRoot[:]))
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
	blocks := []*ethpb.BeaconBlock{
		{
			Slot:       4,
			ParentRoot: []byte("parent"),
		},
		{
			Slot:       5,
			ParentRoot: []byte("parent2"),
		},
		{
			Slot:       6,
			ParentRoot: []byte("parent2"),
		},
		{
			Slot:       7,
			ParentRoot: []byte("parent3"),
		},
		{
			Slot:       8,
			ParentRoot: []byte("parent4"),
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
			// No specified filter should return all blocks.
			filter:            nil,
			expectedNumBlocks: 5,
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

func TestStore_Blocks_RetrieveRange(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)
	b := make([]*ethpb.BeaconBlock, 500)
	for i := 0; i < 500; i++ {
		b[i] = &ethpb.BeaconBlock{
			ParentRoot: []byte("parent"),
			Slot:       uint64(i),
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
