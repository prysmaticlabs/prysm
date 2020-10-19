package node

import (
	"crypto/rand"
	"flag"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/prysmaticlabs/prysm/validator/accounts"
	"github.com/prysmaticlabs/prysm/validator/accounts/wallet"
	"github.com/prysmaticlabs/prysm/validator/flags"
	"github.com/prysmaticlabs/prysm/validator/keymanager"
	logTest "github.com/sirupsen/logrus/hooks/test"
	"github.com/urfave/cli/v2"
)

// Test that the sharding node can build with default flag values.
func TestNode_Builds(t *testing.T) {
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	set.String("datadir", testutil.TempDir()+"/datadir", "the node data directory")
	dir := testutil.TempDir() + "/walletpath"
	passwordDir := testutil.TempDir() + "/password"
	require.NoError(t, os.MkdirAll(passwordDir, os.ModePerm))
	passwordFile := filepath.Join(passwordDir, "password.txt")
	walletPassword := "$$Passw0rdz2$$"
	require.NoError(t, ioutil.WriteFile(
		passwordFile,
		[]byte(walletPassword),
		os.ModePerm,
	))
	defer func() {
		assert.NoError(t, os.RemoveAll(dir))
		assert.NoError(t, os.RemoveAll(passwordDir))
		assert.NoError(t, os.RemoveAll(testutil.TempDir()+"/datadir"))
	}()
	set.String("wallet-dir", dir, "path to wallet")
	set.String("wallet-password-file", passwordFile, "path to wallet password")
	set.String("keymanager-kind", "imported", "keymanager kind")
	set.String("verbosity", "debug", "log verbosity")
	require.NoError(t, set.Set(flags.WalletPasswordFileFlag.Name, passwordFile))
	context := cli.NewContext(&app, set, nil)
	w, err := accounts.CreateWalletWithKeymanager(context.Context, &accounts.CreateWalletConfig{
		WalletCfg: &wallet.Config{
			WalletDir:      dir,
			KeymanagerKind: keymanager.Imported,
			WalletPassword: walletPassword,
		},
	})
	require.NoError(t, err)
	require.NoError(t, w.SaveHashedPassword(context.Context))

	valClient, err := NewValidatorClient(context)
	require.NoError(t, err, "Failed to create ValidatorClient")
	err = valClient.db.Close()
	require.NoError(t, err)
}

// TestClearDB tests clearing the database
func TestClearDB(t *testing.T) {
	hook := logTest.NewGlobal()
	randPath, err := rand.Int(rand.Reader, big.NewInt(1000000))
	require.NoError(t, err, "Could not generate random number for file path")
	tmp := filepath.Join(testutil.TempDir(), fmt.Sprintf("datadirtest%d", randPath))
	require.NoError(t, os.RemoveAll(tmp))
	err = clearDB(tmp, true)
	require.NoError(t, err)
	require.LogsContain(t, hook, "Removing database")
}
