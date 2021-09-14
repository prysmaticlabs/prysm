package db

import (
	"context"
	"flag"
	"io/ioutil"
	"os"
	"path"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/slasher/db/kv"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
	logTest "github.com/sirupsen/logrus/hooks/test"
	"github.com/urfave/cli/v2"
)

func TestRestore(t *testing.T) {
	logHook := logTest.NewGlobal()
	ctx := context.Background()

	backupDb, err := kv.NewKVStore(t.TempDir(), &kv.Config{})
	defer func() {
		require.NoError(t, backupDb.Close())
	}()
	require.NoError(t, err)
	pubKey := []byte("hello")
	require.NoError(t, backupDb.SavePubKey(ctx, types.ValidatorIndex(1), pubKey))
	require.NoError(t, backupDb.Close())
	// We rename the backup file so that we can later verify
	// whether the restored db has been renamed correctly.
	require.NoError(t, os.Rename(
		path.Join(backupDb.DatabasePath(), kv.DatabaseFileName),
		path.Join(backupDb.DatabasePath(), "backup.db")))

	restoreDir := t.TempDir()
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	set.String(cmd.RestoreSourceFileFlag.Name, "", "")
	set.String(cmd.RestoreTargetDirFlag.Name, "", "")
	require.NoError(t, set.Set(cmd.RestoreSourceFileFlag.Name, path.Join(backupDb.DatabasePath(), "backup.db")))
	require.NoError(t, set.Set(cmd.RestoreTargetDirFlag.Name, restoreDir))
	cliCtx := cli.NewContext(&app, set, nil)

	assert.NoError(t, restore(cliCtx))

	files, err := ioutil.ReadDir(path.Join(restoreDir, kv.SlasherDbDirName))
	require.NoError(t, err)
	assert.Equal(t, 1, len(files))
	assert.Equal(t, kv.DatabaseFileName, files[0].Name())
	restoredDb, err := kv.NewKVStore(path.Join(restoreDir, kv.SlasherDbDirName), &kv.Config{})
	defer func() {
		require.NoError(t, restoredDb.Close())
	}()
	require.NoError(t, err)
	received, err := restoredDb.ValidatorPubKey(ctx, types.ValidatorIndex(1))
	require.NoError(t, err)
	require.DeepEqual(t, pubKey, received, "Restored database has incorrect data")
	assert.LogsContain(t, logHook, "Restore completed successfully")
}
