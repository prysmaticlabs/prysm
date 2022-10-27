package kv

import (
	"context"
	"fmt"
	"testing"

	"github.com/prysmaticlabs/prysm/v3/config/features"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/testing/util"
	bolt "go.etcd.io/bbolt"
)

// setupDB instantiates and returns a Store instance.
func setupDB(t testing.TB) *Store {
	db, err := NewKVStore(context.Background(), t.TempDir())
	require.NoError(t, err, "Failed to instantiate DB")
	t.Cleanup(func() {
		require.NoError(t, db.Close(), "Failed to close database")
	})
	return db
}

func Test_setupBlockStorageType(t *testing.T) {
	ctx := context.Background()
	t.Run("fresh database with feature enabled to store full blocks should store full blocks", func(t *testing.T) {
		resetFn := features.InitWithReset(&features.Flags{
			SaveFullExecutionPayloads: true,
		})
		defer resetFn()
		store := setupDB(t)

		blk := util.NewBeaconBlockBellatrix()
		blk.Block.Body.ExecutionPayload.BlockNumber = 1
		wrappedBlock, err := blocks.NewSignedBeaconBlock(blk)
		require.NoError(t, err)
		root, err := wrappedBlock.Block().HashTreeRoot()
		require.NoError(t, err)
		require.NoError(t, store.SaveBlock(ctx, wrappedBlock))
		retrievedBlk, err := store.Block(ctx, root)
		require.NoError(t, err)
		require.Equal(t, false, retrievedBlk.IsBlinded())
		require.DeepEqual(t, wrappedBlock, retrievedBlk)
	})
	t.Run("fresh database with default settings should store blinded", func(t *testing.T) {
		resetFn := features.InitWithReset(&features.Flags{
			SaveFullExecutionPayloads: false,
		})
		defer resetFn()
		store := setupDB(t)

		blk := util.NewBeaconBlockBellatrix()
		blk.Block.Body.ExecutionPayload.BlockNumber = 1
		wrappedBlock, err := blocks.NewSignedBeaconBlock(blk)
		require.NoError(t, err)
		root, err := wrappedBlock.Block().HashTreeRoot()
		require.NoError(t, err)
		require.NoError(t, store.SaveBlock(ctx, wrappedBlock))
		retrievedBlk, err := store.Block(ctx, root)
		require.NoError(t, err)
		require.Equal(t, true, retrievedBlk.IsBlinded())

		wantedBlk, err := wrappedBlock.ToBlinded()
		require.NoError(t, err)
		require.DeepEqual(t, wantedBlk, retrievedBlk)
	})
	t.Run("existing database with full blocks type should continue storing full blocks", func(t *testing.T) {
		store := setupDB(t)
		require.NoError(t, store.db.Update(func(tx *bolt.Tx) error {
			return tx.Bucket(chainMetadataBucket).Delete(saveBlindedBeaconBlocksKey)
		}))

		blk := util.NewBeaconBlockBellatrix()
		blk.Block.Body.ExecutionPayload.BlockNumber = 1
		wrappedBlock, err := blocks.NewSignedBeaconBlock(blk)
		require.NoError(t, err)
		root, err := wrappedBlock.Block().HashTreeRoot()
		require.NoError(t, err)
		require.NoError(t, store.SaveBlock(ctx, wrappedBlock))
		retrievedBlk, err := store.Block(ctx, root)
		require.NoError(t, err)
		require.Equal(t, false, retrievedBlk.IsBlinded())
		require.DeepEqual(t, wrappedBlock, retrievedBlk)

		// Not a fresh database, has full blocks already and should continue being that way.
		err = store.setupBlockStorageType(false /* not a fresh database */)
		require.NoError(t, err)

		blk = util.NewBeaconBlockBellatrix()
		blk.Block.Body.ExecutionPayload.BlockNumber = 2
		wrappedBlock, err = blocks.NewSignedBeaconBlock(blk)
		require.NoError(t, err)
		root, err = wrappedBlock.Block().HashTreeRoot()
		require.NoError(t, err)
		require.NoError(t, store.SaveBlock(ctx, wrappedBlock))
		retrievedBlk, err = store.Block(ctx, root)
		require.NoError(t, err)
		require.Equal(t, false, retrievedBlk.IsBlinded())
		require.DeepEqual(t, wrappedBlock, retrievedBlk)
	})
	t.Run("existing database with blinded blocks type should error if user enables full blocks feature flag", func(t *testing.T) {
		store := setupDB(t)

		blk := util.NewBeaconBlockBellatrix()
		blk.Block.Body.ExecutionPayload.BlockNumber = 1
		wrappedBlock, err := blocks.NewSignedBeaconBlock(blk)
		require.NoError(t, err)
		root, err := wrappedBlock.Block().HashTreeRoot()
		require.NoError(t, err)
		require.NoError(t, store.SaveBlock(ctx, wrappedBlock))
		retrievedBlk, err := store.Block(ctx, root)
		require.NoError(t, err)
		require.Equal(t, true, retrievedBlk.IsBlinded())
		wantedBlk, err := wrappedBlock.ToBlinded()
		require.NoError(t, err)
		require.DeepEqual(t, wantedBlk, retrievedBlk)

		// Trying to enable full blocks with a database that is already storing blinded blocks should error.
		resetFn := features.InitWithReset(&features.Flags{
			SaveFullExecutionPayloads: true,
		})
		defer resetFn()
		err = store.setupBlockStorageType(false /* not a fresh database */)
		errMsg := "cannot use the %s flag with this existing database, as it has already been initialized"
		require.ErrorContains(t, fmt.Sprintf(errMsg, features.SaveFullExecutionPayloads.Name), err)
	})
}
