package kv

import (
	"context"
	"io/ioutil"
	"path"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestStore_Backup(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	require.NoError(t, db.Backup(ctx, ""))

	files, err := ioutil.ReadDir(path.Join(db.databasePath, backupsDirectoryName))
	require.NoError(t, err)
	require.NotEqual(t, 0, len(files), "No backups created")
}
