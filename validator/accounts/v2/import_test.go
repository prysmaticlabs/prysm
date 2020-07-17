package v2

import (
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/validator/flags"
	"github.com/urfave/cli/v2"
)

func TestImport_Noninteractive(t *testing.T) {
	testDir := testutil.TempDir()
	walletDir := filepath.Join(testDir, "/wallet")
	passwordsDir := filepath.Join(testDir, "/walletpasswords")
	exportDir := filepath.Join(testDir, "/export")

	passwordFilePath := filepath.Join(testDir, "password.txt")
	if err := ioutil.WriteFile(passwordFilePath, []byte(password), os.ModePerm); err != nil {
		t.Fatal(err)
	}
	defer func() {
		assert.NoError(t, os.RemoveAll(walletDir))
		assert.NoError(t, os.RemoveAll(passwordsDir))
		assert.NoError(t, os.RemoveAll(exportDir))
	}()
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	set.String(flags.WalletDirFlag.Name, walletDir, "")
	set.String(flags.WalletPasswordsDirFlag.Name, passwordsDir, "")
	set.String(flags.BackupPathFlag.Name, exportDir, "")
	set.String(flags.PasswordFileFlag.Name, passwordFilePath, "")
	assert.NoError(t, set.Set(flags.WalletDirFlag.Name, walletDir))
	assert.NoError(t, set.Set(flags.WalletPasswordsDirFlag.Name, passwordsDir))
	assert.NoError(t, set.Set(flags.PasswordFileFlag.Name, passwordFilePath))
	assert.NoError(t, set.Set(flags.BackupPathFlag.Name, exportDir))
	cliCtx := cli.NewContext(&app, set, nil)

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

	if err := ImportAccount(cliCtx); err != nil {
		t.Fatal(err)
	}
}
