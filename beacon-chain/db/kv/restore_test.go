package kv

import (
	"context"
	"io/ioutil"
	"path"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestRestore(t *testing.T) {
	backupDb := setupDB(t)
	ctx := context.Background()

	head := testutil.NewBeaconBlock()
	head.Block.Slot = 5000
	require.NoError(t, backupDb.SaveBlock(ctx, head))
	root, err := head.Block.HashTreeRoot()
	require.NoError(t, err)
	st := testutil.NewBeaconState()
	require.NoError(t, backupDb.SaveState(ctx, st, root))
	require.NoError(t, backupDb.SaveHeadBlockRoot(ctx, root))
	require.NoError(t, err)
	require.NoError(t, backupDb.Close())

	t.Run("works OK", func(t *testing.T) {
		restoreDir := t.TempDir()
		assert.NoError(t, Restore(context.Background(), path.Join(backupDb.databasePath, databaseFileName), restoreDir))
		files, err := ioutil.ReadDir(path.Join(restoreDir, BeaconNodeDbDirName))
		require.NoError(t, err)
		assert.Equal(t, 1, len(files))
		assert.Equal(t, databaseFileName, files[0].Name())

		restoredDb, err := NewKVStore(path.Join(restoreDir, BeaconNodeDbDirName), nil)
		defer func() {
			require.NoError(t, restoredDb.Close())
		}()
		require.NoError(t, err)
		headBlock, err := restoredDb.HeadBlock(ctx)
		require.NoError(t, err)
		assert.Equal(t, uint64(5000), headBlock.Block.Slot, "Restored database has incorrect data")
	})

	t.Run("database file already exists", func(t *testing.T) {
		restoreDir := t.TempDir()
		backedDb, err := NewKVStore(path.Join(restoreDir, BeaconNodeDbDirName), nil)
		defer func() {
			require.NoError(t, backedDb.Close())
		}()
		require.NoError(t, err)
		err = Restore(context.Background(), path.Join(backupDb.databasePath, databaseFileName), restoreDir)
		assert.ErrorContains(t, "database file already exists", err)
	})
}
