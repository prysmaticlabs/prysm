package v2

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"

	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/validator/flags"
	"github.com/urfave/cli/v2"
)

func setupWalletCtx(
	t *testing.T,
	testDir string,
	exportDir string,
	accountsFlag string,
	keymanagerKind v2keymanager.Kind,
) *cli.Context {
	walletDir := filepath.Join(testDir, walletDirName)
	passwordsDir := filepath.Join(testDir, passwordDirName)
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	set.String(flags.WalletPasswordsDirFlag.Name, passwordsDir, "")
	set.String(flags.WalletDirFlag.Name, walletDir, "")
	set.String(flags.KeymanagerKindFlag.Name, keymanagerKind.String(), "")
	set.String(flags.BackupPathFlag.Name, exportDir, "")
	set.String(flags.AccountsFlag.Name, accountsFlag, "")
	assert.NoError(t, set.Set(flags.WalletPasswordsDirFlag.Name, passwordsDir))
	assert.NoError(t, set.Set(flags.WalletDirFlag.Name, walletDir))
	assert.NoError(t, set.Set(flags.KeymanagerKindFlag.Name, keymanagerKind.String()))
	assert.NoError(t, set.Set(flags.BackupPathFlag.Name, exportDir))
	assert.NoError(t, set.Set(flags.AccountsFlag.Name, accountsFlag))
	return cli.NewContext(&app, set, nil)
}

func TestZipAndUnzip(t *testing.T) {
	testDir := testutil.TempDir()
	walletDir := filepath.Join(testDir, walletDirName)
	passwordsDir := filepath.Join(testDir, passwordDirName)
	exportDir := filepath.Join(testDir, exportDirName)
	importDir := filepath.Join(testDir, importDirName)
	defer func() {
		assert.NoError(t, os.RemoveAll(walletDir))
		assert.NoError(t, os.RemoveAll(passwordsDir))
		assert.NoError(t, os.RemoveAll(exportDir))
		assert.NoError(t, os.RemoveAll(importDir))
	}()
	cliCtx := setupWalletCtx(t, testDir, exportDir, "", v2keymanager.Direct)
	wallet, err := NewWallet(cliCtx)
	require.NoError(t, err)

	accounts, err := wallet.AccountNames()
	require.NoError(t, err)

	if len(accounts) == 0 {
		t.Fatal("Expected more accounts, received 0")
	}
	err = wallet.zipAccounts(accounts, exportDir)
	require.NoError(t, err)

	if _, err := os.Stat(filepath.Join(exportDir, archiveFilename)); os.IsNotExist(err) {
		t.Fatal("Expected file to exist")
	}

	importedAccounts, err := unzipArchiveToTarget(exportDir, importDir)
	require.NoError(t, err)

	allAccountsStr := strings.Join(accounts, " ")
	for _, importedAccount := range importedAccounts {
		if !strings.Contains(allAccountsStr, importedAccount) {
			t.Fatalf("Expected %s to be in %s", importedAccount, allAccountsStr)
		}
	}
}

func TestExport_Noninteractive(t *testing.T) {
	testDir := testutil.TempDir()
	walletDir := filepath.Join(testDir, walletDirName)
	passwordsDir := filepath.Join(testDir, passwordDirName)
	exportDir := filepath.Join(testDir, exportDirName)
	accounts := "all"
	defer func() {
		assert.NoError(t, os.RemoveAll(walletDir))
		assert.NoError(t, os.RemoveAll(passwordsDir))
		assert.NoError(t, os.RemoveAll(exportDir))
	}()
	cliCtx := setupWalletCtx(t, testDir, exportDir, accounts, v2keymanager.Direct)
	require.NoError(t, ExportAccount(cliCtx))
	if _, err := os.Stat(filepath.Join(exportDir, archiveFilename)); os.IsNotExist(err) {
		t.Fatal("Expected file to exist")
	}
}
