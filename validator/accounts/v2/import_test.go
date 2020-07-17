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
	exportDir := filepath.Join(testDir, exportDirName)
	importDir := filepath.Join(testDir, importDirName)
	importPasswordDir := filepath.Join(testDir, importPasswordDirName)

	passwordFilePath := filepath.Join(testDir, passwordFileName)
	assert.NoError(t, ioutil.WriteFile(passwordFilePath, []byte(password), os.ModePerm))
	defer func() {
		assert.NoError(t, os.RemoveAll(exportDir))
		assert.NoError(t, os.RemoveAll(importDir))
		assert.NoError(t, os.RemoveAll(importPasswordDir))
	}()
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	set.String(flags.WalletDirFlag.Name, importDir, "")
	set.String(flags.WalletPasswordsDirFlag.Name, importPasswordDir, "")
	set.String(flags.BackupPathFlag.Name, exportDir, "")
	set.String(flags.PasswordFileFlag.Name, passwordFilePath, "")
	assert.NoError(t, set.Set(flags.WalletDirFlag.Name, importDir))
	assert.NoError(t, set.Set(flags.WalletPasswordsDirFlag.Name, importPasswordDir))
	assert.NoError(t, set.Set(flags.PasswordFileFlag.Name, passwordFilePath))
	assert.NoError(t, set.Set(flags.BackupPathFlag.Name, exportDir))
	cliCtx := cli.NewContext(&app, set, nil)

	wallet := setupWallet(t, testDir)

	accounts, err := wallet.AccountNames()
	assert.NoError(t, err)
	if len(accounts) == 0 {
		t.Fatal("Expected more accounts, received 0")
	}
	err = wallet.zipAccounts(accounts, exportDir)
	assert.NoError(t, err)

	if _, err := os.Stat(filepath.Join(exportDir, archiveFilename)); os.IsNotExist(err) {
		t.Fatal("Expected file to exist")
	}

	err = ImportAccount(cliCtx)
	assert.NoError(t, err)
}
