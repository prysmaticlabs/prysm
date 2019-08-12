package db

import (
	"context"
	"reflect"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/go-ssz"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
)

func TestNilDB_OK(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	block := &ethpb.BeaconBlock{}
	h, _ := ssz.SigningRoot(block)

	hasBlock := db.HasBlock(h)
	if hasBlock {
		t.Fatal("HashBlock should return false")
	}

	bPrime, err := db.Block(h)
	if err != nil {
		t.Fatalf("failed to get block: %v", err)
	}
	if bPrime != nil {
		t.Fatalf("get should return nil for a non existent key")
	}
}

func TestSaveBlock_OK(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	block1 := &ethpb.BeaconBlock{}
	h1, _ := ssz.SigningRoot(block1)

	err := db.SaveBlock(block1)
	if err != nil {
		t.Fatalf("save block failed: %v", err)
	}

	b1Prime, err := db.Block(h1)
	if err != nil {
		t.Fatalf("failed to get block: %v", err)
	}
	h1Prime, _ := ssz.SigningRoot(b1Prime)

	if b1Prime == nil || h1 != h1Prime {
		t.Fatalf("get should return b1: %x", h1)
	}

	block2 := &ethpb.BeaconBlock{
		Slot: 0,
	}
	h2, _ := ssz.SigningRoot(block2)

	err = db.SaveBlock(block2)
	if err != nil {
		t.Fatalf("save block failed: %v", err)
	}

	b2Prime, err := db.Block(h2)
	if err != nil {
		t.Fatalf("failed to get block: %v", err)
	}
	h2Prime, _ := ssz.SigningRoot(b2Prime)
	if b2Prime == nil || h2 != h2Prime {
		t.Fatalf("get should return b2: %x", h2)
	}
}

func TestSaveBlock_NilBlkInCache(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	block := &ethpb.BeaconBlock{Slot: 999}
	h1, _ := ssz.SigningRoot(block)

	// Save a nil block to with block root.
	db.blocks[h1] = nil

	if err := db.SaveBlock(block); err != nil {
		t.Fatalf("save block failed: %v", err)
	}

	savedBlock, err := db.Block(h1)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(block, savedBlock) {
		t.Error("Could not save block in DB")
	}

	// Verify we have the correct cached block
	if !proto.Equal(db.blocks[h1], savedBlock) {
		t.Error("Could not save block in cache")
	}
}

func TestSaveBlockInCache_OK(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	block := &ethpb.BeaconBlock{Slot: 999}
	h, _ := ssz.SigningRoot(block)

	err := db.SaveBlock(block)
	if err != nil {
		t.Fatalf("save block failed: %v", err)
	}

	if !proto.Equal(block, db.blocks[h]) {
		t.Error("Could not save block in cache")
	}

	savedBlock, err := db.Block(h)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(block, savedBlock) {
		t.Error("Could not save block in cache")
	}
}

func TestDeleteBlock_OK(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	block := &ethpb.BeaconBlock{Slot: 0}
	h, _ := ssz.SigningRoot(block)

	err := db.SaveBlock(block)
	if err != nil {
		t.Fatalf("save block failed: %v", err)
	}

	savedBlock, err := db.Block(h)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(block, savedBlock) {
		t.Fatal(err)
	}
	if err := db.DeleteBlock(block); err != nil {
		t.Fatal(err)
	}
	savedBlock, err = db.Block(h)
	if err != nil {
		t.Fatal(err)
	}
	if savedBlock != nil {
		t.Errorf("Expected block to have been deleted, received: %v", savedBlock)
	}
}

func TestDeleteBlockInCache_OK(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	block := &ethpb.BeaconBlock{Slot: 0}
	h, _ := ssz.SigningRoot(block)

	err := db.SaveBlock(block)
	if err != nil {
		t.Fatalf("save block failed: %v", err)
	}

	if err := db.DeleteBlock(block); err != nil {
		t.Fatal(err)
	}

	if _, exists := db.blocks[h]; exists {
		t.Error("Expected block to have been deleted")
	}
}

func TestBlocksBySlotEmptyChain_OK(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)
	ctx := context.Background()

	blocks, _ := db.BlocksBySlot(ctx, 0)
	if len(blocks) > 0 {
		t.Error("BlockBySlot should return nil for an empty chain")
	}
}

func TestBlocksBySlot_MultipleBlocks(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)
	ctx := context.Background()

	slotNum := uint64(3)
	b1 := &ethpb.BeaconBlock{
		Slot:       slotNum,
		ParentRoot: []byte("A"),
		Body: &ethpb.BeaconBlockBody{
			RandaoReveal: []byte("A"),
		},
	}
	b2 := &ethpb.BeaconBlock{
		Slot:       slotNum,
		ParentRoot: []byte("B"),
		Body: &ethpb.BeaconBlockBody{
			RandaoReveal: []byte("B"),
		}}
	b3 := &ethpb.BeaconBlock{
		Slot:       slotNum,
		ParentRoot: []byte("C"),
		Body: &ethpb.BeaconBlockBody{
			RandaoReveal: []byte("C"),
		}}
	if err := db.SaveBlock(b1); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveBlock(b2); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveBlock(b3); err != nil {
		t.Fatal(err)
	}

	blocks, _ := db.BlocksBySlot(ctx, 3)
	if len(blocks) != 3 {
		t.Errorf("Wanted %d blocks, received %d", 3, len(blocks))
	}
}

func TestHasBlock_returnsTrue(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	block := &ethpb.BeaconBlock{
		Slot: uint64(44),
	}

	root, err := ssz.SigningRoot(block)
	if err != nil {
		t.Fatal(err)
	}

	if err := db.SaveBlock(block); err != nil {
		t.Fatal(err)
	}

	if !db.HasBlock(root) {
		t.Fatal("db.HasBlock returned false for block just saved")
	}
}

func TestHighestBlockSlot_UpdatedOnSaveBlock(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	block := &ethpb.BeaconBlock{
		Slot: 23,
	}

	if err := db.SaveBlock(block); err != nil {
		t.Fatal(err)
	}

	if db.HighestBlockSlot() != block.Slot {
		t.Errorf("Unexpected highest slot %d, wanted %d", db.HighestBlockSlot(), block.Slot)
	}

	block.Slot = 55
	if err := db.SaveBlock(block); err != nil {
		t.Fatal(err)
	}

	if db.HighestBlockSlot() != block.Slot {
		t.Errorf("Unexpected highest slot %d, wanted %d", db.HighestBlockSlot(), block.Slot)
	}
}

func TestClearBlockCache_OK(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	block := &ethpb.BeaconBlock{Slot: 0}

	err := db.SaveBlock(block)
	if err != nil {
		t.Fatalf("save block failed: %v", err)
	}
	if len(db.blocks) != 1 {
		t.Error("incorrect block cache length")
	}
	db.ClearBlockCache()
	if len(db.blocks) != 0 {
		t.Error("incorrect block cache length")
	}
}

func TestSaveHeadBlockRoot_OK(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	s := &pb.BeaconState{Slot: 1}
	headRoot := []byte{'A'}

	if err := db.SaveForkChoiceState(context.Background(), s, headRoot); err != nil {
		t.Fatal(err)
	}

	if err := db.SaveHeadBlockRoot(headRoot); err != nil {
		t.Fatal(err)
	}

	savedS, err := db.HeadState(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(s, savedS) {
		t.Error("incorrect saved head state")
	}
}
