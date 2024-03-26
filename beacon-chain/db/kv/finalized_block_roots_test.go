package kv

import (
	"bytes"
	"context"
	"testing"

	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	consensusblocks "github.com/prysmaticlabs/prysm/v5/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/testing/assert"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	"github.com/prysmaticlabs/prysm/v5/testing/util"
	bolt "go.etcd.io/bbolt"
)

var genesisBlockRoot = bytesutil.ToBytes32([]byte{'G', 'E', 'N', 'E', 'S', 'I', 'S'})

func TestStore_IsFinalizedBlock(t *testing.T) {
	slotsPerEpoch := uint64(params.BeaconConfig().SlotsPerEpoch)
	db := setupDB(t)
	ctx := context.Background()

	require.NoError(t, db.SaveGenesisBlockRoot(ctx, genesisBlockRoot))
	blks := makeBlocks(t, 0, slotsPerEpoch*2, genesisBlockRoot)
	require.NoError(t, db.SaveBlocks(ctx, blks))

	root, err := blks[slotsPerEpoch].Block().HashTreeRoot()
	require.NoError(t, err)
	cp := &ethpb.Checkpoint{
		Epoch: 1,
		Root:  root[:],
	}
	require.NoError(t, db.SaveFinalizedCheckpoint(ctx, cp))

	for i := uint64(0); i <= slotsPerEpoch; i++ {
		root, err = blks[i].Block().HashTreeRoot()
		require.NoError(t, err)
		assert.Equal(t, true, db.IsFinalizedBlock(ctx, root), "Block at index %d was not considered finalized", i)
	}
	for i := slotsPerEpoch + 1; i < uint64(len(blks)); i++ {
		root, err = blks[i].Block().HashTreeRoot()
		require.NoError(t, err)
		assert.Equal(t, false, db.IsFinalizedBlock(ctx, root), "Block at index %d was considered finalized, but should not have", i)
	}
}

func TestStore_IsFinalizedGenesisBlock(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	blk := util.NewBeaconBlock()
	blk.Block.Slot = 0
	root, err := blk.Block.HashTreeRoot()
	require.NoError(t, err)
	wsb, err := consensusblocks.NewSignedBeaconBlock(blk)
	require.NoError(t, err)
	require.NoError(t, db.SaveBlock(ctx, wsb))
	require.NoError(t, db.SaveGenesisBlockRoot(ctx, root))
	assert.Equal(t, true, db.IsFinalizedBlock(ctx, root))
}

func TestStore_IsFinalizedChildBlock(t *testing.T) {
	slotsPerEpoch := uint64(params.BeaconConfig().SlotsPerEpoch)
	ctx := context.Background()
	db := setupDB(t)
	require.NoError(t, db.SaveGenesisBlockRoot(ctx, genesisBlockRoot))

	blks := makeBlocks(t, 0, slotsPerEpoch*2, genesisBlockRoot)
	require.NoError(t, db.SaveBlocks(ctx, blks))
	root, err := blks[slotsPerEpoch].Block().HashTreeRoot()
	require.NoError(t, err)
	cp := &ethpb.Checkpoint{
		Epoch: 1,
		Root:  root[:],
	}
	require.NoError(t, db.SaveFinalizedCheckpoint(ctx, cp))

	for i := uint64(0); i < slotsPerEpoch; i++ {
		root, err = blks[i].Block().HashTreeRoot()
		require.NoError(t, err)
		assert.Equal(t, true, db.IsFinalizedBlock(ctx, root), "Block at index %d was not considered finalized", i)
		blk, err := db.FinalizedChildBlock(ctx, root)
		assert.NoError(t, err)
		assert.Equal(t, false, blk == nil, "Child block at index %d was not considered finalized", i)
	}
}

func TestStore_ChildRootOfPrevFinalizedCheckpointIsUpdated(t *testing.T) {
	slotsPerEpoch := uint64(params.BeaconConfig().SlotsPerEpoch)
	ctx := context.Background()
	db := setupDB(t)
	require.NoError(t, db.SaveGenesisBlockRoot(ctx, genesisBlockRoot))

	blks := makeBlocks(t, 0, slotsPerEpoch*3, genesisBlockRoot)
	require.NoError(t, db.SaveBlocks(ctx, blks))
	root, err := blks[slotsPerEpoch].Block().HashTreeRoot()
	require.NoError(t, err)
	cp := &ethpb.Checkpoint{
		Epoch: 1,
		Root:  root[:],
	}
	require.NoError(t, db.SaveFinalizedCheckpoint(ctx, cp))
	root2, err := blks[slotsPerEpoch*2].Block().HashTreeRoot()
	require.NoError(t, err)
	cp = &ethpb.Checkpoint{
		Epoch: 2,
		Root:  root2[:],
	}
	require.NoError(t, db.SaveFinalizedCheckpoint(ctx, cp))

	require.NoError(t, db.db.View(func(tx *bolt.Tx) error {
		container := &ethpb.FinalizedBlockRootContainer{}
		f := tx.Bucket(finalizedBlockRootsIndexBucket).Get(root[:])
		require.NoError(t, decode(ctx, f, container))
		r, err := blks[slotsPerEpoch+1].Block().HashTreeRoot()
		require.NoError(t, err)
		assert.DeepEqual(t, r[:], container.ChildRoot)
		return nil
	}))
}

func TestStore_OrphanedBlockIsNotFinalized(t *testing.T) {
	slotsPerEpoch := uint64(params.BeaconConfig().SlotsPerEpoch)
	db := setupDB(t)
	ctx := context.Background()

	require.NoError(t, db.SaveGenesisBlockRoot(ctx, genesisBlockRoot))
	blk0 := util.NewBeaconBlock()
	blk0.Block.ParentRoot = genesisBlockRoot[:]
	blk0Root, err := blk0.Block.HashTreeRoot()
	require.NoError(t, err)
	blk1 := util.NewBeaconBlock()
	blk1.Block.Slot = 1
	blk1.Block.ParentRoot = blk0Root[:]
	blk2 := util.NewBeaconBlock()
	blk2.Block.Slot = 2
	// orphan block at index 1
	blk2.Block.ParentRoot = blk0Root[:]
	blk2Root, err := blk2.Block.HashTreeRoot()
	require.NoError(t, err)
	sBlk0, err := consensusblocks.NewSignedBeaconBlock(blk0)
	require.NoError(t, err)
	sBlk1, err := consensusblocks.NewSignedBeaconBlock(blk1)
	require.NoError(t, err)
	sBlk2, err := consensusblocks.NewSignedBeaconBlock(blk2)
	require.NoError(t, err)
	blks := append([]interfaces.ReadOnlySignedBeaconBlock{sBlk0, sBlk1, sBlk2}, makeBlocks(t, 3, slotsPerEpoch*2-3, blk2Root)...)
	require.NoError(t, db.SaveBlocks(ctx, blks))

	root, err := blks[slotsPerEpoch].Block().HashTreeRoot()
	require.NoError(t, err)
	cp := &ethpb.Checkpoint{
		Epoch: 1,
		Root:  root[:],
	}
	require.NoError(t, db.SaveFinalizedCheckpoint(ctx, cp))

	for i := uint64(0); i <= slotsPerEpoch; i++ {
		root, err = blks[i].Block().HashTreeRoot()
		require.NoError(t, err)
		if i == 1 {
			assert.Equal(t, false, db.IsFinalizedBlock(ctx, root), "Block at index 1 was considered finalized, but should not have")
		} else {
			assert.Equal(t, true, db.IsFinalizedBlock(ctx, root), "Block at index %d was not considered finalized", i)
		}
	}
}

func makeBlocks(t *testing.T, i, n uint64, previousRoot [32]byte) []interfaces.ReadOnlySignedBeaconBlock {
	blocks := make([]*ethpb.SignedBeaconBlock, n)
	ifaceBlocks := make([]interfaces.ReadOnlySignedBeaconBlock, n)
	for j := i; j < n+i; j++ {
		parentRoot := make([]byte, fieldparams.RootLength)
		copy(parentRoot, previousRoot[:])
		blocks[j-i] = util.NewBeaconBlock()
		blocks[j-i].Block.Slot = primitives.Slot(j + 1)
		blocks[j-i].Block.ParentRoot = parentRoot
		var err error
		previousRoot, err = blocks[j-i].Block.HashTreeRoot()
		require.NoError(t, err)
		ifaceBlocks[j-i], err = consensusblocks.NewSignedBeaconBlock(blocks[j-i])
		require.NoError(t, err)
	}
	return ifaceBlocks
}

func TestStore_BackfillFinalizedIndexSingle(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	// we're making 4 blocks so we can test an element without a valid child at the end
	blks, err := consensusblocks.NewROBlockSlice(makeBlocks(t, 0, 4, [32]byte{}))
	require.NoError(t, err)

	// existing is the child that we'll set up in the index by hand to seed the index.
	existing := blks[3]

	// toUpdate is a single item update, emulating a backfill batch size of 1. it is the parent of `existing`.
	toUpdate := blks[2]

	// set up existing finalized block
	ebpr := existing.Block().ParentRoot()
	ebr := existing.Root()
	ebf := &ethpb.FinalizedBlockRootContainer{
		ParentRoot: ebpr[:],
		ChildRoot:  make([]byte, 32), // we're bypassing validation to seed the db, so we don't need a valid child.
	}
	enc, err := encode(ctx, ebf)
	require.NoError(t, err)
	// writing this to the index outside of the validating function to seed the test.
	err = db.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(finalizedBlockRootsIndexBucket)
		return bkt.Put(ebr[:], enc)
	})
	require.NoError(t, err)

	require.NoError(t, db.BackfillFinalizedIndex(ctx, []consensusblocks.ROBlock{toUpdate}, ebr))

	// make sure that we still correctly validate descendents in the single item case.
	noChild := blks[0] // will fail to update because we don't have blks[1] in the db.
	// test wrong child param
	require.ErrorIs(t, db.BackfillFinalizedIndex(ctx, []consensusblocks.ROBlock{noChild}, ebr), errNotConnectedToFinalized)
	// test parent of child that isn't finalized
	require.ErrorIs(t, db.BackfillFinalizedIndex(ctx, []consensusblocks.ROBlock{noChild}, blks[1].Root()), errFinalizedChildNotFound)

	// now make it work by writing the missing block
	require.NoError(t, db.BackfillFinalizedIndex(ctx, []consensusblocks.ROBlock{blks[1]}, blks[2].Root()))
	// since blks[1] is now in the index, we should be able to update blks[0]
	require.NoError(t, db.BackfillFinalizedIndex(ctx, []consensusblocks.ROBlock{blks[0]}, blks[1].Root()))
}

func TestStore_BackfillFinalizedIndex(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	require.ErrorIs(t, db.BackfillFinalizedIndex(ctx, []consensusblocks.ROBlock{}, [32]byte{}), errEmptyBlockSlice)
	blks, err := consensusblocks.NewROBlockSlice(makeBlocks(t, 0, 66, [32]byte{}))
	require.NoError(t, err)

	// set up existing finalized block
	ebpr := blks[64].Block().ParentRoot()
	ebr := blks[64].Root()
	chldr := blks[65].Root()
	ebf := &ethpb.FinalizedBlockRootContainer{
		ParentRoot: ebpr[:],
		ChildRoot:  chldr[:],
	}
	enc, err := encode(ctx, ebf)
	require.NoError(t, err)
	err = db.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(finalizedBlockRootsIndexBucket)
		return bkt.Put(ebr[:], enc)
	})
	require.NoError(t, err)

	// reslice to remove the existing blocks
	blks = blks[0:64]
	// check the other error conditions with a descendent root that really doesn't exist

	disjoint := []consensusblocks.ROBlock{
		blks[0],
		blks[2],
	}
	require.ErrorIs(t, db.BackfillFinalizedIndex(ctx, disjoint, [32]byte{}), errIncorrectBlockParent)
	require.ErrorIs(t, errFinalizedChildNotFound, db.BackfillFinalizedIndex(ctx, blks, [32]byte{}))

	// use the real root so that it succeeds
	require.NoError(t, db.BackfillFinalizedIndex(ctx, blks, ebr))
	for i := range blks {
		require.NoError(t, db.db.View(func(tx *bolt.Tx) error {
			bkt := tx.Bucket(finalizedBlockRootsIndexBucket)
			encfr := bkt.Get(blks[i].RootSlice())
			require.Equal(t, true, len(encfr) > 0)
			fr := &ethpb.FinalizedBlockRootContainer{}
			require.NoError(t, decode(ctx, encfr, fr))
			require.Equal(t, 32, len(fr.ParentRoot))
			require.Equal(t, 32, len(fr.ChildRoot))
			pr := blks[i].Block().ParentRoot()
			require.Equal(t, true, bytes.Equal(fr.ParentRoot, pr[:]))
			if i > 0 {
				require.Equal(t, true, bytes.Equal(fr.ParentRoot, blks[i-1].RootSlice()))
			}
			if i < len(blks)-1 {
				require.DeepEqual(t, fr.ChildRoot, blks[i+1].RootSlice())
			}
			if i == len(blks)-1 {
				require.DeepEqual(t, fr.ChildRoot, ebr[:])
			}
			return nil
		}))
	}
}
