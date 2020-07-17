package v2

import (
	"context"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/validator/flags"
	"github.com/urfave/cli/v2"

	"github.com/prysmaticlabs/prysm/shared/testutil"
	v2 "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
)

const password = "10testPass!"

func setupWallet(t *testing.T, testDir string) *Wallet {
	walletDir := filepath.Join(testDir, "/wallet")
	passwordsDir := filepath.Join(testDir, "/walletpasswords")
	ctx := context.Background()
	if err := initializeDirectWallet(walletDir, passwordsDir); err != nil {
		t.Fatal(err)
	}
	cfg := &WalletConfig{
		WalletDir:      walletDir,
		PasswordsDir:   passwordsDir,
		KeymanagerKind: v2.Direct,
	}
	w, err := NewWallet(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	keymanager, err := w.InitializeKeymanager(ctx, true)
	if err != nil {
		t.Fatalf("Could not initialize keymanager: %v", err)
	}
	if _, err := keymanager.CreateAccount(ctx, password); err != nil {
		t.Fatalf("Could not create account in wallet: %v", err)
	}
	return w
}

func TestZipAndUnzip(t *testing.T) {
	testDir := testutil.TempDir()
	exportDir := filepath.Join(testDir, "/export")
	wallet := setupWallet(t, testDir)

	accounts, err := wallet.AccountNames()
	if err != nil {
		t.Fatal(err)
	}
	if len(accounts) == 0 {
		t.Fatal("Expected more accounts, received 0")
	}
	if err := wallet.zipAccounts(accounts, exportDir); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(exportDir, archiveFilename)); os.IsNotExist(err) {
		t.Fatal("Expected file to exist")
	}

	importFolder := "import"
	importDir := filepath.Join(testDir, importFolder)
	importedAccounts, err := unzipArchiveToTarget(exportDir, importDir)
	if err != nil {
		t.Fatal(err)
	}

	allAccountsStr := strings.Join(accounts, " ")
	for _, importedAccount := range importedAccounts {
		if !strings.Contains(allAccountsStr, importedAccount) {
			t.Fatalf("Expected %s to be in %s", importedAccount, allAccountsStr)
		}
	}
}

func TestExport_Noninteractive(t *testing.T) {
	testDir := testutil.TempDir()
	walletDir := filepath.Join(testDir, "/wallet")
	passwordsDir := filepath.Join(testDir, "/walletpasswords")
	exportDir := filepath.Join(testDir, "/export")
	accounts := "all"
	defer func() {
		assert.NoError(t, os.RemoveAll(walletDir))
		assert.NoError(t, os.RemoveAll(passwordsDir))
		assert.NoError(t, os.RemoveAll(exportDir))
	}()
	setupWallet(t, testDir)
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	set.String(flags.WalletDirFlag.Name, walletDir, "")
	set.String(flags.WalletPasswordsDirFlag.Name, passwordsDir, "")
	set.String(flags.BackupPathFlag.Name, exportDir, "")
	set.String(flags.AccountsFlag.Name, accounts, "")
	assert.NoError(t, set.Set(flags.WalletDirFlag.Name, walletDir))
	assert.NoError(t, set.Set(flags.WalletPasswordsDirFlag.Name, passwordsDir))
	assert.NoError(t, set.Set(flags.BackupPathFlag.Name, exportDir))
	assert.NoError(t, set.Set(flags.AccountsFlag.Name, accounts))
	cliCtx := cli.NewContext(&app, set, nil)

	if err := ExportAccount(cliCtx); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(exportDir, archiveFilename)); os.IsNotExist(err) {
		t.Fatal("Expected file to exist")
	}
}
