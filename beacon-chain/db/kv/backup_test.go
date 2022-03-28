package kv

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
)

func TestStore_Backup(t *testing.T) {
	db, err := NewKVStore(context.Background(), t.TempDir(), &Config{})
	require.NoError(t, err, "Failed to instantiate DB")
	ctx := context.Background()

	head := util.NewBeaconBlock()
	head.Block.Slot = 5000

	wsb, err := wrapper.WrappedSignedBeaconBlock(head)
	require.NoError(t, err)
	require.NoError(t, db.SaveBlock(ctx, wsb))
	root, err := head.Block.HashTreeRoot()
	require.NoError(t, err)
	st, err := util.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, db.SaveState(ctx, st, root))
	require.NoError(t, db.SaveHeadBlockRoot(ctx, root))

	require.NoError(t, db.Backup(ctx, "", false))

	backupsPath := filepath.Join(db.databasePath, backupsDirectoryName)
	files, err := ioutil.ReadDir(backupsPath)
	require.NoError(t, err)
	require.NotEqual(t, 0, len(files), "No backups created")
	require.NoError(t, db.Close(), "Failed to close database")

	oldFilePath := filepath.Join(backupsPath, files[0].Name())
	newFilePath := filepath.Join(backupsPath, DatabaseFileName)
	// We rename the file to match the database file name
	// our NewKVStore function expects when opening a database.
	require.NoError(t, os.Rename(oldFilePath, newFilePath))

	backedDB, err := NewKVStore(ctx, backupsPath, &Config{})
	require.NoError(t, err, "Failed to instantiate DB")
	t.Cleanup(func() {
		require.NoError(t, backedDB.Close(), "Failed to close database")
	})
	require.Equal(t, true, backedDB.HasState(ctx, root))
}

func TestStore_BackupMultipleBuckets(t *testing.T) {
	db, err := NewKVStore(context.Background(), t.TempDir(), &Config{})
	require.NoError(t, err, "Failed to instantiate DB")
	ctx := context.Background()

	startSlot := types.Slot(5000)

	for i := startSlot; i < 5200; i++ {
		head := util.NewBeaconBlock()
		head.Block.Slot = i
		wsb, err := wrapper.WrappedSignedBeaconBlock(head)
		require.NoError(t, err)
		require.NoError(t, db.SaveBlock(ctx, wsb))
		root, err := head.Block.HashTreeRoot()
		require.NoError(t, err)
		st, err := util.NewBeaconState()
		require.NoError(t, st.SetSlot(i))
		require.NoError(t, err)
		require.NoError(t, db.SaveState(ctx, st, root))
		require.NoError(t, db.SaveHeadBlockRoot(ctx, root))
	}

	require.NoError(t, db.Backup(ctx, "", false))

	backupsPath := filepath.Join(db.databasePath, backupsDirectoryName)
	files, err := ioutil.ReadDir(backupsPath)
	require.NoError(t, err)
	require.NotEqual(t, 0, len(files), "No backups created")
	require.NoError(t, db.Close(), "Failed to close database")

	oldFilePath := filepath.Join(backupsPath, files[0].Name())
	newFilePath := filepath.Join(backupsPath, DatabaseFileName)
	// We rename the file to match the database file name
	// our NewKVStore function expects when opening a database.
	require.NoError(t, os.Rename(oldFilePath, newFilePath))

	backedDB, err := NewKVStore(ctx, backupsPath, &Config{})
	require.NoError(t, err, "Failed to instantiate DB")
	t.Cleanup(func() {
		require.NoError(t, backedDB.Close(), "Failed to close database")
	})
	for i := startSlot; i < 5200; i++ {
		head := util.NewBeaconBlock()
		head.Block.Slot = i
		root, err := head.Block.HashTreeRoot()
		require.NoError(t, err)
		nBlock, err := backedDB.Block(ctx, root)
		require.NoError(t, err)
		require.NotNil(t, nBlock)
		require.Equal(t, nBlock.Block().Slot(), i)
		nState, err := backedDB.State(ctx, root)
		require.NoError(t, err)
		require.NotNil(t, nState)
		require.Equal(t, nState.Slot(), i)
	}
}
