package v2

import (
	"context"
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/prysmaticlabs/prysm/validator/accounts/v2/wallet"
	"github.com/prysmaticlabs/prysm/validator/flags"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/derived"
	"github.com/urfave/cli/v2"
)

func TestRecoverDerivedWallet(t *testing.T) {
	testDir := testutil.TempDir()
	walletDir := filepath.Join(testDir, walletDirName)
	passwordsDir := filepath.Join(testDir, passwordDirName)
	exportDir := filepath.Join(testDir, exportDirName)
	defer func() {
		assert.NoError(t, os.RemoveAll(walletDir))
		assert.NoError(t, os.RemoveAll(passwordsDir))
		assert.NoError(t, os.RemoveAll(exportDir))
	}()

	passwordFilePath := filepath.Join(testDir, passwordFileName)
	require.NoError(t, ioutil.WriteFile(passwordFilePath, []byte(password), os.ModePerm))
	mnemonicFilePath := filepath.Join(testDir, mnemonicFileName)
	require.NoError(t, ioutil.WriteFile(mnemonicFilePath, []byte(mnemonic), os.ModePerm))

	numAccounts := int64(4)
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	set.String(flags.WalletDirFlag.Name, walletDir, "")
	set.String(flags.DeprecatedPasswordsDirFlag.Name, passwordsDir, "")
	set.String(flags.WalletPasswordFileFlag.Name, passwordFilePath, "")
	set.String(flags.KeymanagerKindFlag.Name, v2keymanager.Derived.String(), "")
	set.String(flags.MnemonicFileFlag.Name, mnemonicFilePath, "")
	set.Int64(flags.NumAccountsFlag.Name, numAccounts, "")
	assert.NoError(t, set.Set(flags.WalletDirFlag.Name, walletDir))
	assert.NoError(t, set.Set(flags.DeprecatedPasswordsDirFlag.Name, passwordsDir))
	assert.NoError(t, set.Set(flags.WalletPasswordFileFlag.Name, passwordFilePath))
	assert.NoError(t, set.Set(flags.KeymanagerKindFlag.Name, v2keymanager.Derived.String()))
	assert.NoError(t, set.Set(flags.MnemonicFileFlag.Name, mnemonicFilePath))
	assert.NoError(t, set.Set(flags.NumAccountsFlag.Name, strconv.Itoa(int(numAccounts))))
	cliCtx := cli.NewContext(&app, set, nil)

	require.NoError(t, RecoverWalletCli(cliCtx))

	ctx := context.Background()
	w, err := wallet.OpenWallet(cliCtx.Context, &wallet.Config{
		WalletDir:      walletDir,
		WalletPassword: password,
	})
	assert.NoError(t, err)

	encoded, err := w.ReadKeymanagerConfigFromDisk(ctx)
	assert.NoError(t, err)
	cfg, err := derived.UnmarshalOptionsFile(encoded)
	assert.NoError(t, err)

	// We assert the created configuration was as desired.
	wantCfg := derived.DefaultKeymanagerOpts()
	assert.DeepEqual(t, wantCfg, cfg)

	keymanager, err := w.InitializeKeymanager(cliCtx.Context, true)
	require.NoError(t, err)
	km, ok := keymanager.(*derived.Keymanager)
	if !ok {
		t.Fatal("not a derived keymanager")
	}
	names, err := km.ValidatingAccountNames(ctx)
	assert.NoError(t, err)
	require.Equal(t, len(names), int(numAccounts))

}

// TestRecoverDerivedWallet_OneAccount is a test for regression in cases where the number of accounts recovered is 1
func TestRecoverDerivedWallet_OneAccount(t *testing.T) {
	testDir := testutil.TempDir()
	walletDir := filepath.Join(testDir, walletDirName)
	defer func() {
		assert.NoError(t, os.RemoveAll(walletDir))
	}()

	passwordFilePath := filepath.Join(testDir, passwordFileName)
	require.NoError(t, ioutil.WriteFile(passwordFilePath, []byte(password), os.ModePerm))
	mnemonicFilePath := filepath.Join(testDir, mnemonicFileName)
	require.NoError(t, ioutil.WriteFile(mnemonicFilePath, []byte(mnemonic), os.ModePerm))

	numAccounts := int64(1)
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	set.String(flags.WalletDirFlag.Name, walletDir, "")
	set.String(flags.WalletPasswordFileFlag.Name, passwordFilePath, "")
	set.String(flags.KeymanagerKindFlag.Name, v2keymanager.Derived.String(), "")
	set.String(flags.MnemonicFileFlag.Name, mnemonicFilePath, "")
	set.Int64(flags.NumAccountsFlag.Name, numAccounts, "")
	assert.NoError(t, set.Set(flags.WalletDirFlag.Name, walletDir))
	assert.NoError(t, set.Set(flags.WalletPasswordFileFlag.Name, passwordFilePath))
	assert.NoError(t, set.Set(flags.KeymanagerKindFlag.Name, v2keymanager.Derived.String()))
	assert.NoError(t, set.Set(flags.MnemonicFileFlag.Name, mnemonicFilePath))
	assert.NoError(t, set.Set(flags.NumAccountsFlag.Name, strconv.Itoa(int(numAccounts))))
	cliCtx := cli.NewContext(&app, set, nil)

	require.NoError(t, RecoverWalletCli(cliCtx))

	_, err := wallet.OpenWallet(cliCtx.Context, &wallet.Config{
		WalletDir:      walletDir,
		WalletPassword: password,
	})
	assert.NoError(t, err)
}
