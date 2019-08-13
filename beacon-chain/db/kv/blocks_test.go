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
	block := &ethpb.BeaconBlock{
		Slot:       20,
		ParentRoot: []byte{1, 2, 3},
	}
	ctx := context.Background()
	if err := db.SaveBlock(ctx, block); err != nil {
		t.Fatal(err)
	}
	blockRoot, err := ssz.HashTreeRoot(block)
	if err != nil {
		t.Fatal(err)
	}
	if !db.HasBlock(ctx, blockRoot) {
		t.Error("Expected block to exist in the db")
	}
	retrievedBlock, err := db.Block(ctx, blockRoot)
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
			ParentRoot: []byte("parent"),
		},
		{
			ParentRoot: []byte("parent2"),
		},
		{
			ParentRoot: []byte("parent3"),
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
			expectedNumBlocks: 1,
		},
		{
			// No specified filter should return all blocks.
			filter:            nil,
			expectedNumBlocks: 3,
		},
		{
			// No block meets the criteria below.
			filter:            filters.NewFilter().SetParentRoot([]byte{3, 4, 5}),
			expectedNumBlocks: 0,
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
