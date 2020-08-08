package kv

import (
	"context"
	"io/ioutil"
	"path"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestStore_Backup(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	head := testutil.NewBeaconBlock()
	head.Block.Slot = 5000

	require.NoError(t, db.SaveBlock(ctx, head))
	root, err := stateutil.BlockRoot(head.Block)
	require.NoError(t, err)
	st := testutil.NewBeaconState()
	require.NoError(t, db.SaveState(ctx, st, root))
	require.NoError(t, db.SaveHeadBlockRoot(ctx, root))

	require.NoError(t, db.Backup(ctx))

	files, err := ioutil.ReadDir(path.Join(db.databasePath, backupsDirectoryName))
	require.NoError(t, err)
	require.NotEqual(t, 0, len(files), "No backups created")
}
