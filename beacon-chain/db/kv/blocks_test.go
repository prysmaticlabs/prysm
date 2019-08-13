package kv

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/go-ssz"
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
