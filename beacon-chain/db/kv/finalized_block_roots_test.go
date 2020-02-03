package kv

import (
	"context"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

var genesisBlockRoot = bytesutil.ToBytes32([]byte{'G', 'E', 'N', 'E', 'S', 'I', 'S'})

func TestStore_IsFinalizedBlock(t *testing.T) {
	slotsPerEpoch := int(params.BeaconConfig().SlotsPerEpoch)
	db := setupDB(t)
	defer teardownDB(t, db)
	ctx := context.Background()

	if err := db.SaveGenesisBlockRoot(ctx, genesisBlockRoot); err != nil {
		t.Fatal(err)
	}

	blks := makeBlocks(t, 0, slotsPerEpoch*3, genesisBlockRoot)
	if err := db.SaveBlocks(ctx, blks); err != nil {
		t.Fatal(err)
	}

	root, err := ssz.HashTreeRoot(blks[slotsPerEpoch].Block)
	if err != nil {
		t.Fatal(err)
	}

	cp := &ethpb.Checkpoint{
		Epoch: 1,
		Root:  root[:],
	}

	st, err := state.InitializeFromProto(&pb.BeaconState{})
	if err != nil {
		t.Fatal(err)
	}
	// a state is required to save checkpoint
	if err := db.SaveState(ctx, st, root); err != nil {
		t.Fatal(err)
	}

	if err := db.SaveFinalizedCheckpoint(ctx, cp); err != nil {
		t.Fatal(err)
	}

	// All blocks up to slotsPerEpoch*2 should be in the finalized index.
	for i := 0; i < slotsPerEpoch*2; i++ {
		root, err := ssz.HashTreeRoot(blks[i].Block)
		if err != nil {
			t.Fatal(err)
		}
		if !db.IsFinalizedBlock(ctx, root) {
			t.Errorf("Block at index %d was not considered finalized in the index", i)
		}
	}
	for i := slotsPerEpoch * 3; i < len(blks); i++ {
		root, err := ssz.HashTreeRoot(blks[i].Block)
		if err != nil {
			t.Fatal(err)
		}
		if db.IsFinalizedBlock(ctx, root) {
			t.Errorf("Block at index %d was considered finalized in the index, but should not have", i)
		}
	}
}

// This test scenario is to test a specific edge case where the finalized block root is not part of
// the finalized and canonical chain.
//
// Example:
// 0    1  2  3   4     5   6     slot
// a <- b <-- d <- e <- f <- g    roots
//      ^- c
// Imagine that epochs are 2 slots and that epoch 1, 2, and 3 are finalized. Checkpoint roots would
// be c, e, and g. In this scenario, c was a finalized checkpoint root but no block built upon it so
// it should not be considered "final and canonical" in the view at slot 6.
func TestStore_IsFinalized_ForkEdgeCase(t *testing.T) {
	slotsPerEpoch := int(params.BeaconConfig().SlotsPerEpoch)
	blocks0 := makeBlocks(t, slotsPerEpoch*0, slotsPerEpoch, genesisBlockRoot)
	blocks1 := append(
		makeBlocks(t, slotsPerEpoch*1, 1, bytesutil.ToBytes32(sszRootOrDie(t, blocks0[len(blocks0)-1]))), // No block builds off of the first block in epoch.
		makeBlocks(t, slotsPerEpoch*1+1, slotsPerEpoch-1, bytesutil.ToBytes32(sszRootOrDie(t, blocks0[len(blocks0)-1])))...,
	)
	blocks2 := makeBlocks(t, slotsPerEpoch*2, slotsPerEpoch, bytesutil.ToBytes32(sszRootOrDie(t, blocks1[len(blocks1)-1])))

	db := setupDB(t)
	defer teardownDB(t, db)
	ctx := context.Background()

	if err := db.SaveGenesisBlockRoot(ctx, genesisBlockRoot); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveBlocks(ctx, blocks0); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveBlocks(ctx, blocks1); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveBlocks(ctx, blocks2); err != nil {
		t.Fatal(err)
	}

	// First checkpoint
	checkpoint1 := &ethpb.Checkpoint{
		Root:  sszRootOrDie(t, blocks1[0]),
		Epoch: 1,
	}

	st, err := state.InitializeFromProto(&pb.BeaconState{})
	if err != nil {
		t.Fatal(err)
	}
	// A state is required to save checkpoint
	if err := db.SaveState(ctx, st, bytesutil.ToBytes32(checkpoint1.Root)); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveFinalizedCheckpoint(ctx, checkpoint1); err != nil {
		t.Fatal(err)
	}
	// All blocks in blocks0 and blocks1 should be finalized and canonical.
	for i, block := range append(blocks0, blocks1...) {
		root := sszRootOrDie(t, block)
		if !db.IsFinalizedBlock(ctx, bytesutil.ToBytes32(root)) {
			t.Errorf("%d - Expected block %#x to be finalized", i, root)
		}
	}

	// Second checkpoint
	checkpoint2 := &ethpb.Checkpoint{
		Root:  sszRootOrDie(t, blocks2[0]),
		Epoch: 2,
	}
	// A state is required to save checkpoint
	if err := db.SaveState(ctx, st, bytesutil.ToBytes32(checkpoint2.Root)); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveFinalizedCheckpoint(ctx, checkpoint2); err != nil {
		t.Error(err)
	}
	// All blocks in blocks0 and blocks2 should be finalized and canonical.
	for i, block := range append(blocks0, blocks2...) {
		root := sszRootOrDie(t, block)
		if !db.IsFinalizedBlock(ctx, bytesutil.ToBytes32(root)) {
			t.Errorf("%d - Expected block %#x to be finalized", i, root)
		}
	}
	// All blocks in blocks1 should be finalized and canonical, except blocks1[0].
	for i, block := range blocks1 {
		root := sszRootOrDie(t, block)
		if db.IsFinalizedBlock(ctx, bytesutil.ToBytes32(root)) == (i == 0) {
			t.Errorf("Expected db.IsFinalizedBlock(ctx, blocks1[%d]) to be %v", i, i != 0)
		}
	}
}

func sszRootOrDie(t *testing.T, block *ethpb.SignedBeaconBlock) []byte {
	root, err := ssz.HashTreeRoot(block.Block)
	if err != nil {
		t.Fatal(err)
	}
	return root[:]
}

func makeBlocks(t *testing.T, i, n int, previousRoot [32]byte) []*ethpb.SignedBeaconBlock {
	blocks := make([]*ethpb.SignedBeaconBlock, n)
	for j := i; j < n+i; j++ {
		parentRoot := make([]byte, 32)
		copy(parentRoot, previousRoot[:])
		blocks[j-i] = &ethpb.SignedBeaconBlock{
			Block: &ethpb.BeaconBlock{
				Slot:       uint64(j + 1),
				ParentRoot: parentRoot,
			},
		}
		var err error
		previousRoot, err = ssz.HashTreeRoot(blocks[j-i].Block)
		if err != nil {
			t.Fatal(err)
		}
	}
	return blocks
}
