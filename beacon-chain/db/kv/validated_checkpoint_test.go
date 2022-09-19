package kv

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/v3/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/testing/util"
	"google.golang.org/protobuf/proto"
)

func TestStore_LastValidatedCheckpoint_CanSaveRetrieve(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	root := bytesutil.ToBytes32([]byte{'A'})
	cp := &ethpb.Checkpoint{
		Epoch: 10,
		Root:  root[:],
	}
	st, err := util.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, st.SetSlot(1))
	require.NoError(t, db.SaveState(ctx, st, root))
	require.NoError(t, db.SaveLastValidatedCheckpoint(ctx, cp))

	retrieved, err := db.LastValidatedCheckpoint(ctx)
	require.NoError(t, err)
	assert.Equal(t, true, proto.Equal(cp, retrieved), "Wanted %v, received %v", cp, retrieved)
}

func TestStore_LastValidatedCheckpoint_Recover(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	blk := util.HydrateSignedBeaconBlock(&ethpb.SignedBeaconBlock{})
	r, err := blk.Block.HashTreeRoot()
	require.NoError(t, err)
	cp := &ethpb.Checkpoint{
		Epoch: 2,
		Root:  r[:],
	}
	wb, err := blocks.NewSignedBeaconBlock(blk)
	require.NoError(t, err)
	require.NoError(t, db.SaveBlock(ctx, wb))
	require.NoError(t, db.SaveLastValidatedCheckpoint(ctx, cp))
	retrieved, err := db.LastValidatedCheckpoint(ctx)
	require.NoError(t, err)
	assert.Equal(t, true, proto.Equal(cp, retrieved), "Wanted %v, received %v", cp, retrieved)
}

func TestStore_LastValidatedCheckpoint_DefaultIsFinalized(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	genesis := bytesutil.ToBytes32([]byte{'G', 'E', 'N', 'E', 'S', 'I', 'S'})
	require.NoError(t, db.SaveGenesisBlockRoot(ctx, genesis))

	blk := util.NewBeaconBlock()
	blk.Block.ParentRoot = genesis[:]
	blk.Block.Slot = 40

	root, err := blk.Block.HashTreeRoot()
	require.NoError(t, err)

	cp := &ethpb.Checkpoint{
		Epoch: 5,
		Root:  root[:],
	}

	// a valid chain is required to save finalized checkpoint.
	wsb, err := blocks.NewSignedBeaconBlock(blk)
	require.NoError(t, err)
	require.NoError(t, db.SaveBlock(ctx, wsb))
	st, err := util.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, st.SetSlot(1))
	// a state is required to save checkpoint
	require.NoError(t, db.SaveState(ctx, st, root))

	require.NoError(t, db.SaveFinalizedCheckpoint(ctx, cp))

	retrieved, err := db.LastValidatedCheckpoint(ctx)
	require.NoError(t, err)
	assert.Equal(t, true, proto.Equal(cp, retrieved), "Wanted %v, received %v", cp, retrieved)
}
