package kv

import (
	"context"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
)

var genesisBlockRoot = bytesutil.ToBytes32([]byte{'G', 'E', 'N', 'E', 'S', 'I', 'S'})

// In this test we consider the following forked chain
//
// 0 -- ... -- 62 -- 64 -- 65 -- ... -- 127
//              \
//                -- 63
// The last slot of Epoch 1 is orphaned.
//
// This tests the following
// 1- We finalize Epoch 1, all blocks 0 -- 63 should be final
// 2- Check that blocks 64 -- 127 are not final
// 3- Finalize Epoch 2, all blocks 64 -- 95 are final
// 4- Block 63 should not be final
// 5- Blocks 96-- 127 should not be final
// 6- Blocks 0 -- 62 should be final
func TestStore_DanglingNonCanonicalBlock(t *testing.T) {
	slotsPerEpoch := uint64(params.BeaconConfig().SlotsPerEpoch)
	db := setupDB(t)
	ctx := context.Background()

	require.NoError(t, db.SaveGenesisBlockRoot(ctx, genesisBlockRoot))

	blks := makeBlocks(t, 0, slotsPerEpoch*2, genesisBlockRoot)
	require.NoError(t, db.SaveBlocks(ctx, blks))
	orphanParentRoot, err := blks[slotsPerEpoch*2-2].Block().HashTreeRoot()
	require.NoError(t, err)
	blks2 := makeBlocks(t, slotsPerEpoch*2, slotsPerEpoch*2, orphanParentRoot)
	require.NoError(t, db.SaveBlocks(ctx, blks2))

	require.NoError(t, db.SaveBlocks(ctx, blks))

	root, err := blks[slotsPerEpoch].Block().HashTreeRoot()
	require.NoError(t, err)

	cp := &ethpb.Checkpoint{
		Epoch: 1,
		Root:  root[:],
	}

	st, err := util.NewBeaconState()
	require.NoError(t, err)
	// a state is required to save checkpoint
	require.NoError(t, db.SaveState(ctx, st, root))
	require.NoError(t, db.SaveFinalizedCheckpoint(ctx, cp))

	// We finalize Epoch 1 that contains the dangling node
	// All blocks up to slotsPerEpoch*2 should be in the finalized index.
	for i := uint64(0); i < slotsPerEpoch*2; i++ {
		root, err := blks[i].Block().HashTreeRoot()
		require.NoError(t, err)
		assert.Equal(t, true, db.IsFinalizedBlock(ctx, root), "Block at index %d was not considered finalized in the index", i)
	}
	// All blocks in Epochs 2 and 3 should not be Final
	for i := 0; i < len(blks2); i++ {
		root, err := blks2[i].Block().HashTreeRoot()
		require.NoError(t, err)
		assert.Equal(t, false, db.IsFinalizedBlock(ctx, root), "Block at index %d was considered finalized in the index, but should not have", i)
	}

	// We now finalize epoch 2, the first block of which orphans the
	// dangling node
	root, err = blks2[0].Block().HashTreeRoot()
	require.NoError(t, err)
	cp.Root = root[:]
	cp.Epoch = 2
	require.NoError(t, db.SaveState(ctx, st, root))
	require.NoError(t, db.SaveFinalizedCheckpoint(ctx, cp))
	// All blocks up to slotsPerEpoch*3 should be in the finalized index.
	for i := uint64(0); i < slotsPerEpoch; i++ {
		root, err := blks2[i].Block().HashTreeRoot()
		require.NoError(t, err)
		assert.Equal(t, true, db.IsFinalizedBlock(ctx, root), "Block at index %d was not considered finalized in the index", i)
	}
	// The Block at slot 63 should not be considered final
	root, err = blks[2*slotsPerEpoch-1].Block().HashTreeRoot()
	require.NoError(t, err)
	assert.Equal(t, false, db.IsFinalizedBlock(ctx, root), "Block at slot 63 was considered finalized in the index")
	// All blocks up to slotsPerEpoch*3 should be in the finalized index.
	for i := slotsPerEpoch; i < 2*slotsPerEpoch; i++ {
		root, err := blks2[i].Block().HashTreeRoot()
		require.NoError(t, err)
		assert.Equal(t, false, db.IsFinalizedBlock(ctx, root), "Block at slot %d was considered finalized in the index and should not be", i+2*slotsPerEpoch)
	}
	// All blocks up to slotsPerEpoch*2-2 should be in the finalized index.
	for i := uint64(0); i < slotsPerEpoch*2-2; i++ {
		root, err := blks[i].Block().HashTreeRoot()
		require.NoError(t, err)
		assert.Equal(t, true, db.IsFinalizedBlock(ctx, root), "Block at index %d was not considered finalized in the index", i)
	}

}

func TestStore_IsFinalizedBlockGenesis(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	blk := util.NewBeaconBlock()
	blk.Block.Slot = 0
	root, err := blk.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, db.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(blk)))
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
	slotsPerEpoch := uint64(params.BeaconConfig().SlotsPerEpoch)
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

	st, err := util.NewBeaconState()
	require.NoError(t, err)
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

func TestStore_IsFinalizedChildBlock(t *testing.T) {
	slotsPerEpoch := uint64(params.BeaconConfig().SlotsPerEpoch)
	ctx := context.Background()

	eval := func(t testing.TB, ctx context.Context, db *Store, blks []block.SignedBeaconBlock) {
		require.NoError(t, db.SaveBlocks(ctx, blks))
		root, err := blks[slotsPerEpoch].Block().HashTreeRoot()
		require.NoError(t, err)

		cp := &ethpb.Checkpoint{
			Epoch: 1,
			Root:  root[:],
		}

		st, err := util.NewBeaconState()
		require.NoError(t, err)
		// a state is required to save checkpoint
		require.NoError(t, db.SaveState(ctx, st, root))
		require.NoError(t, db.SaveFinalizedCheckpoint(ctx, cp))

		// All blocks up to slotsPerEpoch should have a finalized child block.
		for i := uint64(0); i < slotsPerEpoch; i++ {
			root, err := blks[i].Block().HashTreeRoot()
			require.NoError(t, err)
			assert.Equal(t, true, db.IsFinalizedBlock(ctx, root), "Block at index %d was not considered finalized in the index", i)
			blk, err := db.FinalizedChildBlock(ctx, root)
			assert.NoError(t, err)
			if blk == nil {
				t.Error("Child block doesn't exist for valid finalized block.")
			}
		}
	}

	setup := func(t testing.TB) *Store {
		db := setupDB(t)
		require.NoError(t, db.SaveGenesisBlockRoot(ctx, genesisBlockRoot))

		return db
	}

	t.Run("phase0", func(t *testing.T) {
		db := setup(t)

		blks := makeBlocks(t, 0, slotsPerEpoch*3, genesisBlockRoot)
		eval(t, ctx, db, blks)
	})

	t.Run("altair", func(t *testing.T) {
		db := setup(t)

		blks := makeBlocksAltair(t, 0, slotsPerEpoch*3, genesisBlockRoot)
		eval(t, ctx, db, blks)
	})
}

func sszRootOrDie(t *testing.T, block block.SignedBeaconBlock) []byte {
	root, err := block.Block().HashTreeRoot()
	require.NoError(t, err)
	return root[:]
}

func makeBlocks(t *testing.T, i, n uint64, previousRoot [32]byte) []block.SignedBeaconBlock {
	blocks := make([]*ethpb.SignedBeaconBlock, n)
	ifaceBlocks := make([]block.SignedBeaconBlock, n)
	for j := i; j < n+i; j++ {
		parentRoot := make([]byte, fieldparams.RootLength)
		copy(parentRoot, previousRoot[:])
		blocks[j-i] = util.NewBeaconBlock()
		blocks[j-i].Block.Slot = types.Slot(j)
		blocks[j-i].Block.ParentRoot = parentRoot
		var err error
		previousRoot, err = blocks[j-i].Block.HashTreeRoot()
		require.NoError(t, err)
		ifaceBlocks[j-i] = wrapper.WrappedPhase0SignedBeaconBlock(blocks[j-i])
	}
	return ifaceBlocks
}

func makeBlocksAltair(t *testing.T, startIdx, num uint64, previousRoot [32]byte) []block.SignedBeaconBlock {
	blocks := make([]*ethpb.SignedBeaconBlockAltair, num)
	ifaceBlocks := make([]block.SignedBeaconBlock, num)
	for j := startIdx; j < num+startIdx; j++ {
		parentRoot := make([]byte, fieldparams.RootLength)
		copy(parentRoot, previousRoot[:])
		blocks[j-startIdx] = util.NewBeaconBlockAltair()
		blocks[j-startIdx].Block.Slot = types.Slot(j + 1)
		blocks[j-startIdx].Block.ParentRoot = parentRoot
		var err error
		previousRoot, err = blocks[j-startIdx].Block.HashTreeRoot()
		require.NoError(t, err)
		ifaceBlocks[j-startIdx], err = wrapper.WrappedAltairSignedBeaconBlock(blocks[j-startIdx])
		require.NoError(t, err)
	}
	return ifaceBlocks
}
