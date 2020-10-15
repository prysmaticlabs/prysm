package accounts

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
	"github.com/prysmaticlabs/prysm/validator/accounts/wallet"
	"github.com/prysmaticlabs/prysm/validator/flags"
	"github.com/prysmaticlabs/prysm/validator/keymanager"
	"github.com/prysmaticlabs/prysm/validator/keymanager/derived"
	"github.com/urfave/cli/v2"
)

type recoverCfgStruct struct {
	walletDir        string
	passwordFilePath string
	mnemonicFilePath string
	numAccounts      int64
}

func setupRecoverCfg(t *testing.T) *recoverCfgStruct {
	testDir := testutil.TempDir()
	walletDir := filepath.Join(testDir, walletDirName)
	passwordFilePath := filepath.Join(testDir, passwordFileName)
	require.NoError(t, ioutil.WriteFile(passwordFilePath, []byte(password), os.ModePerm))
	mnemonicFilePath := filepath.Join(testDir, mnemonicFileName)
	require.NoError(t, ioutil.WriteFile(mnemonicFilePath, []byte(mnemonic), os.ModePerm))

	t.Cleanup(func() {
		assert.NoError(t, os.RemoveAll(walletDir))
		assert.NoError(t, os.Remove(passwordFilePath))
		assert.NoError(t, os.Remove(mnemonicFilePath))
	})

	return &recoverCfgStruct{
		walletDir:        walletDir,
		passwordFilePath: passwordFilePath,
		mnemonicFilePath: mnemonicFilePath,
	}
}

func createRecoverCliCtx(t *testing.T, cfg *recoverCfgStruct) *cli.Context {
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	set.String(flags.WalletDirFlag.Name, cfg.walletDir, "")
	set.String(flags.WalletPasswordFileFlag.Name, cfg.passwordFilePath, "")
	set.String(flags.KeymanagerKindFlag.Name, keymanager.Derived.String(), "")
	set.String(flags.MnemonicFileFlag.Name, cfg.mnemonicFilePath, "")
	set.Int64(flags.NumAccountsFlag.Name, cfg.numAccounts, "")
	assert.NoError(t, set.Set(flags.WalletDirFlag.Name, cfg.walletDir))
	assert.NoError(t, set.Set(flags.WalletPasswordFileFlag.Name, cfg.passwordFilePath))
	assert.NoError(t, set.Set(flags.KeymanagerKindFlag.Name, keymanager.Derived.String()))
	assert.NoError(t, set.Set(flags.MnemonicFileFlag.Name, cfg.mnemonicFilePath))
	assert.NoError(t, set.Set(flags.NumAccountsFlag.Name, strconv.Itoa(int(cfg.numAccounts))))
	return cli.NewContext(&app, set, nil)
}

func TestRecoverDerivedWallet(t *testing.T) {
	cfg := setupRecoverCfg(t)
	cfg.numAccounts = 4
	cliCtx := createRecoverCliCtx(t, cfg)
	require.NoError(t, RecoverWalletCli(cliCtx))

	ctx := context.Background()
	w, err := wallet.OpenWallet(cliCtx.Context, &wallet.Config{
		WalletDir:      cfg.walletDir,
		WalletPassword: password,
	})
	assert.NoError(t, err)

	encoded, err := w.ReadKeymanagerConfigFromDisk(ctx)
	assert.NoError(t, err)
	walletCfg, err := derived.UnmarshalOptionsFile(encoded)
	assert.NoError(t, err)
	// We assert the created configuration was as desired.
	wantCfg := derived.DefaultKeymanagerOpts()
	assert.DeepEqual(t, wantCfg, walletCfg)

	keymanager, err := w.InitializeKeymanager(cliCtx.Context, true)
	require.NoError(t, err)
	km, ok := keymanager.(*derived.Keymanager)
	if !ok {
		t.Fatal("not a derived keymanager")
	}
	names, err := km.ValidatingAccountNames(ctx)
	assert.NoError(t, err)
	require.Equal(t, len(names), int(cfg.numAccounts))
}

// TestRecoverDerivedWallet_OneAccount is a test for regression in cases where the number of accounts recovered is 1
func TestRecoverDerivedWallet_OneAccount(t *testing.T) {
	cfg := setupRecoverCfg(t)
	cfg.numAccounts = 1
	cliCtx := createRecoverCliCtx(t, cfg)
	require.NoError(t, RecoverWalletCli(cliCtx))

	_, err := wallet.OpenWallet(cliCtx.Context, &wallet.Config{
		WalletDir:      cfg.walletDir,
		WalletPassword: password,
	})
	assert.NoError(t, err)
}

func TestRecoverDerivedWallet_AlreadyExists(t *testing.T) {
	cfg := setupRecoverCfg(t)
	cfg.numAccounts = 4
	cliCtx := createRecoverCliCtx(t, cfg)
	require.NoError(t, RecoverWalletCli(cliCtx))

	// Trying to recover an HD wallet into a directory that already exists should give an error
	require.ErrorContains(t, "a wallet already exists at this location", RecoverWalletCli(cliCtx))
}
