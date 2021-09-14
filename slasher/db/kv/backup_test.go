package kv

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestStore_Backup(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	pubKey := []byte("hello")
	require.NoError(t, db.SavePubKey(ctx, types.ValidatorIndex(1), pubKey))
	require.NoError(t, db.Backup(ctx, "", false))

	backupsPath := filepath.Join(db.databasePath, backupsDirectoryName)
	files, err := ioutil.ReadDir(backupsPath)
	require.NoError(t, err)
	require.NotEqual(t, 0, len(files), "No backups created")

	oldFilePath := filepath.Join(backupsPath, files[0].Name())
	newFilePath := filepath.Join(backupsPath, DatabaseFileName)
	// We rename the file to match the database file name
	// our NewKVStore function expects when opening a database.
	require.NoError(t, os.Rename(oldFilePath, newFilePath))

	backedDB, err := NewKVStore(backupsPath, &Config{})
	require.NoError(t, err, "Failed to instantiate DB")
	t.Cleanup(func() {
		require.NoError(t, backedDB.Close(), "Failed to close database")
	})
	received, err := backedDB.ValidatorPubKey(ctx, types.ValidatorIndex(1))
	require.NoError(t, err)
	require.DeepEqual(t, pubKey, received)
}
