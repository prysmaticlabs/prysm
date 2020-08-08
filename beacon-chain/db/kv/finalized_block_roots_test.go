package kv

import (
	"context"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

var genesisBlockRoot = bytesutil.ToBytes32([]byte{'G', 'E', 'N', 'E', 'S', 'I', 'S'})

func TestStore_IsFinalizedBlock(t *testing.T) {
	slotsPerEpoch := params.BeaconConfig().SlotsPerEpoch
	db := setupDB(t)
	ctx := context.Background()

	require.NoError(t, db.SaveGenesisBlockRoot(ctx, genesisBlockRoot))

	blks := makeBlocks(t, 0, slotsPerEpoch*3, genesisBlockRoot)
	require.NoError(t, db.SaveBlocks(ctx, blks))

	root, err := stateutil.BlockRoot(blks[slotsPerEpoch].Block)
	require.NoError(t, err)

	cp := &ethpb.Checkpoint{
		Epoch: 1,
		Root:  root[:],
	}

	st := testutil.NewBeaconState()
	// a state is required to save checkpoint
	require.NoError(t, db.SaveState(ctx, st, root))
	require.NoError(t, db.SaveFinalizedCheckpoint(ctx, cp))

	// All blocks up to slotsPerEpoch*2 should be in the finalized index.
	for i := uint64(0); i < slotsPerEpoch*2; i++ {
		root, err := stateutil.BlockRoot(blks[i].Block)
		require.NoError(t, err)
		assert.Equal(t, true, db.IsFinalizedBlock(ctx, root), "Block at index %d was not considered finalized in the index", i)
	}
	for i := slotsPerEpoch * 3; i < uint64(len(blks)); i++ {
		root, err := stateutil.BlockRoot(blks[i].Block)
		require.NoError(t, err)
		assert.Equal(t, false, db.IsFinalizedBlock(ctx, root), "Block at index %d was considered finalized in the index, but should not have", i)
	}
}

func TestStore_IsFinalizedBlockGenesis(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	blk := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 0}}
	root, err := stateutil.BlockRoot(blk.Block)
	require.NoError(t, err)
	require.NoError(t, db.SaveBlock(ctx, blk))
	require.NoError(t, db.SaveGenesisBlockRoot(ctx, root))
	assert.Equal(t, true, db.IsFinalizedBlock(ctx, root), "Finalized genesis block doesn't exist in db")
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
	slotsPerEpoch := params.BeaconConfig().SlotsPerEpoch
	blocks0 := makeBlocks(t, slotsPerEpoch*0, slotsPerEpoch, genesisBlockRoot)
	blocks1 := append(
		makeBlocks(t, slotsPerEpoch*1, 1, bytesutil.ToBytes32(sszRootOrDie(t, blocks0[len(blocks0)-1]))), // No block builds off of the first block in epoch.
		makeBlocks(t, slotsPerEpoch*1+1, slotsPerEpoch-1, bytesutil.ToBytes32(sszRootOrDie(t, blocks0[len(blocks0)-1])))...,
	)
	blocks2 := makeBlocks(t, slotsPerEpoch*2, slotsPerEpoch, bytesutil.ToBytes32(sszRootOrDie(t, blocks1[len(blocks1)-1])))

	db := setupDB(t)
	ctx := context.Background()

	require.NoError(t, db.SaveGenesisBlockRoot(ctx, genesisBlockRoot))
	require.NoError(t, db.SaveBlocks(ctx, blocks0))
	require.NoError(t, db.SaveBlocks(ctx, blocks1))
	require.NoError(t, db.SaveBlocks(ctx, blocks2))

	// First checkpoint
	checkpoint1 := &ethpb.Checkpoint{
		Root:  sszRootOrDie(t, blocks1[0]),
		Epoch: 1,
	}

	st := testutil.NewBeaconState()
	// A state is required to save checkpoint
	require.NoError(t, db.SaveState(ctx, st, bytesutil.ToBytes32(checkpoint1.Root)))
	require.NoError(t, db.SaveFinalizedCheckpoint(ctx, checkpoint1))
	// All blocks in blocks0 and blocks1 should be finalized and canonical.
	for i, block := range append(blocks0, blocks1...) {
		root := sszRootOrDie(t, block)
		assert.Equal(t, true, db.IsFinalizedBlock(ctx, bytesutil.ToBytes32(root)), "%d - Expected block %#x to be finalized", i, root)
	}

	// Second checkpoint
	checkpoint2 := &ethpb.Checkpoint{
		Root:  sszRootOrDie(t, blocks2[0]),
		Epoch: 2,
	}
	// A state is required to save checkpoint
	require.NoError(t, db.SaveState(ctx, st, bytesutil.ToBytes32(checkpoint2.Root)))
	require.NoError(t, db.SaveFinalizedCheckpoint(ctx, checkpoint2))
	// All blocks in blocks0 and blocks2 should be finalized and canonical.
	for i, block := range append(blocks0, blocks2...) {
		root := sszRootOrDie(t, block)
		assert.Equal(t, true, db.IsFinalizedBlock(ctx, bytesutil.ToBytes32(root)), "%d - Expected block %#x to be finalized", i, root)
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
	root, err := stateutil.BlockRoot(block.Block)
	require.NoError(t, err)
	return root[:]
}

func makeBlocks(t *testing.T, i, n uint64, previousRoot [32]byte) []*ethpb.SignedBeaconBlock {
	blocks := make([]*ethpb.SignedBeaconBlock, n)
	for j := i; j < n+i; j++ {
		parentRoot := make([]byte, 32)
		copy(parentRoot, previousRoot[:])
		blocks[j-i] = &ethpb.SignedBeaconBlock{
			Block: &ethpb.BeaconBlock{
				Slot:       j + 1,
				ParentRoot: parentRoot,
			},
		}
		var err error
		previousRoot, err = stateutil.BlockRoot(blocks[j-i].Block)
		require.NoError(t, err)
	}
	return blocks
}
