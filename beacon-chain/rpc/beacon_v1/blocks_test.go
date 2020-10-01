package beacon_v1

import (
	"context"
	"reflect"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/bytesutil"

	"github.com/golang/protobuf/proto"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1"
	ethpb_alpha "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	dbTest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestServer_GetBlock_GenesisRoot(t *testing.T) {
	db, _ := dbTest.SetupDB(t)
	ctx := context.Background()

	bs := &Server{
		BeaconDB: db,
	}
	// Should return the proper genesis block if it exists.
	parentRoot := [32]byte{1, 2, 3}
	blk := testutil.NewBeaconBlock()
	blk.Block.ParentRoot = parentRoot[:]
	root, err := blk.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, db.SaveBlock(ctx, blk))
	require.NoError(t, db.SaveGenesisBlockRoot(ctx, root))

	count := uint64(100)
	blks := make([]*ethpb_alpha.SignedBeaconBlock, count)
	blkContainers := make([]*ethpb_alpha.BeaconBlockContainer, count)
	for i := uint64(0); i < count; i++ {
		b := testutil.NewBeaconBlock()
		b.Block.Slot = i
		root, err := b.Block.HashTreeRoot()
		require.NoError(t, err)
		blks[i] = b
		blkContainers[i] = &ethpb_alpha.BeaconBlockContainer{Block: b, BlockRoot: root[:]}
	}
	require.NoError(t, db.SaveBlocks(ctx, blks))

	// Should throw an error if more than one blk returned.
	block, err := bs.GetBlock(ctx, &ethpb.BlockRequest{
		BlockId: root[:],
	})
	require.NoError(t, err)

	marshaledBlk, err := blk.Block.Marshal()
	require.NoError(t, err)
	v1Block := &ethpb.BeaconBlock{}
	require.NoError(t, proto.Unmarshal(marshaledBlk, v1Block))

	if !reflect.DeepEqual(block.Data.Message, v1Block) {
		t.Error("Expected blocks to equal")
	}
}

func TestServer_GetBlock_Root(t *testing.T) {
	db, _ := dbTest.SetupDB(t)
	ctx := context.Background()

	bs := &Server{
		BeaconDB: db,
	}
	// Should return the proper genesis block if it exists.
	parentRoot := [32]byte{1, 2, 3}
	blk := testutil.NewBeaconBlock()
	blk.Block.ParentRoot = parentRoot[:]
	root, err := blk.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, db.SaveBlock(ctx, blk))
	require.NoError(t, db.SaveGenesisBlockRoot(ctx, root))

	count := uint64(100)
	blks := make([]*ethpb_alpha.SignedBeaconBlock, count)
	blkContainers := make([]*ethpb_alpha.BeaconBlockContainer, count)
	for i := uint64(0); i < count; i++ {
		b := testutil.NewBeaconBlock()
		b.Block.Slot = i
		root, err := b.Block.HashTreeRoot()
		require.NoError(t, err)
		blks[i] = b
		blkContainers[i] = &ethpb_alpha.BeaconBlockContainer{Block: b, BlockRoot: root[:]}
	}
	require.NoError(t, db.SaveBlocks(ctx, blks))

	// Should throw an error if more than one blk returned.
	block, err := bs.GetBlock(ctx, &ethpb.BlockRequest{
		BlockId: blkContainers[50].BlockRoot,
	})
	require.NoError(t, err)

	marshaledBlk, err := blks[50].Block.Marshal()
	require.NoError(t, err)
	v1Block := &ethpb.BeaconBlock{}
	require.NoError(t, proto.Unmarshal(marshaledBlk, v1Block))

	if !reflect.DeepEqual(block.Data.Message, v1Block) {
		t.Error("Expected blocks to equal")
	}
}

func TestServer_GetBlock_Slot(t *testing.T) {
	db, _ := dbTest.SetupDB(t)
	ctx := context.Background()

	bs := &Server{
		BeaconDB: db,
	}
	// Should return the proper genesis block if it exists.
	parentRoot := [32]byte{1, 2, 3}
	blk := testutil.NewBeaconBlock()
	blk.Block.ParentRoot = parentRoot[:]
	root, err := blk.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, db.SaveBlock(ctx, blk))
	require.NoError(t, db.SaveGenesisBlockRoot(ctx, root))

	count := uint64(100)
	blks := make([]*ethpb_alpha.SignedBeaconBlock, count)
	blkContainers := make([]*ethpb_alpha.BeaconBlockContainer, count)
	for i := uint64(0); i < count; i++ {
		b := testutil.NewBeaconBlock()
		b.Block.Slot = i
		root, err := b.Block.HashTreeRoot()
		require.NoError(t, err)
		blks[i] = b
		blkContainers[i] = &ethpb_alpha.BeaconBlockContainer{Block: b, BlockRoot: root[:]}
	}
	require.NoError(t, db.SaveBlocks(ctx, blks))

	// Should throw an error if more than one blk returned.
	block, err := bs.GetBlock(ctx, &ethpb.BlockRequest{
		BlockId: bytesutil.ToBytes(40, 8),
	})
	require.NoError(t, err)

	marshaledBlk, err := blks[40].Block.Marshal()
	require.NoError(t, err)
	v1Block := &ethpb.BeaconBlock{}
	require.NoError(t, proto.Unmarshal(marshaledBlk, v1Block))

	if !reflect.DeepEqual(block.Data.Message, v1Block) {
		t.Error("Expected blocks to equal")
	}
}
