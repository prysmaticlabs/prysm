package node

import (
	"context"
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/prysmaticlabs/prysm/cmd/validator/flags"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/validator/accounts"
	"github.com/prysmaticlabs/prysm/validator/accounts/wallet"
	"github.com/prysmaticlabs/prysm/validator/keymanager"
	logTest "github.com/sirupsen/logrus/hooks/test"
	"github.com/urfave/cli/v2"
)

// Test that the sharding node can build with default flag values.
func TestNode_Builds(t *testing.T) {
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	set.String("datadir", t.TempDir()+"/datadir", "the node data directory")
	dir := t.TempDir() + "/walletpath"
	passwordDir := t.TempDir() + "/password"
	require.NoError(t, os.MkdirAll(passwordDir, os.ModePerm))
	passwordFile := filepath.Join(passwordDir, "password.txt")
	walletPassword := "$$Passw0rdz2$$"
	require.NoError(t, ioutil.WriteFile(
		passwordFile,
		[]byte(walletPassword),
		os.ModePerm,
	))
	set.String("wallet-dir", dir, "path to wallet")
	set.String("wallet-password-file", passwordFile, "path to wallet password")
	set.String("keymanager-kind", "imported", "keymanager kind")
	set.String("verbosity", "debug", "log verbosity")
	require.NoError(t, set.Set(flags.WalletPasswordFileFlag.Name, passwordFile))
	context := cli.NewContext(&app, set, nil)
	_, err := accounts.CreateWalletWithKeymanager(context.Context, &accounts.CreateWalletConfig{
		WalletCfg: &wallet.Config{
			WalletDir:      dir,
			KeymanagerKind: keymanager.Local,
			WalletPassword: walletPassword,
		},
	})
	require.NoError(t, err)

	valClient, err := NewValidatorClient(context)
	require.NoError(t, err, "Failed to create ValidatorClient")
	err = valClient.db.Close()
	require.NoError(t, err)
}

// TestClearDB tests clearing the database
func TestClearDB(t *testing.T) {
	hook := logTest.NewGlobal()
	tmp := filepath.Join(t.TempDir(), "datadirtest")
	require.NoError(t, clearDB(context.Background(), tmp, true))
	require.LogsContain(t, hook, "Removing database")
}
