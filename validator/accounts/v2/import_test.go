package v2

import (
	"context"
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/direct"
)

func TestImport_Noninteractive(t *testing.T) {
	walletDir, passwordsDir := setupWalletAndPasswordsDir(t)
	randPath, err := rand.Int(rand.Reader, big.NewInt(1000000))
	require.NoError(t, err, "Could not generate random file path")
	exportDir := path.Join(testutil.TempDir(), fmt.Sprintf("/%d", randPath), "export")
	importDir := path.Join(testutil.TempDir(), fmt.Sprintf("/%d", randPath), "import")
	importPasswordDir := path.Join(testutil.TempDir(), fmt.Sprintf("/%d", randPath), "importpassword")
	t.Cleanup(func() {
		require.NoError(t, os.RemoveAll(exportDir), "Failed to remove directory")
		require.NoError(t, os.RemoveAll(importDir), "Failed to remove directory")
		require.NoError(t, os.RemoveAll(importPasswordDir), "Failed to remove directory")
	})
	require.NoError(t, os.MkdirAll(importPasswordDir, os.ModePerm))
	passwordFilePath := filepath.Join(importPasswordDir, passwordFileName)
	require.NoError(t, ioutil.WriteFile(passwordFilePath, []byte(password), os.ModePerm))

	cliCtx := setupWalletCtx(t, &testWalletConfig{
		walletDir:      walletDir,
		passwordsDir:   passwordsDir,
		exportDir:      exportDir,
		keymanagerKind: v2keymanager.Direct,
		passwordFile:   passwordFilePath,
	})
	wallet, err := NewWallet(cliCtx)
	require.NoError(t, err)

	ctx := context.Background()
	keymanager, err := direct.NewKeymanager(
		ctx,
		wallet,
		direct.DefaultConfig(),
		true, /* skip mnemonic */
	)
	require.NoError(t, err)
	_, err = keymanager.CreateAccount(ctx, password)
	require.NoError(t, err)

	accounts, err := wallet.AccountNames()
	require.NoError(t, err)
	assert.Equal(t, len(accounts), 1)

	require.NoError(t, wallet.zipAccounts(accounts, exportDir))
	if _, err := os.Stat(filepath.Join(exportDir, archiveFilename)); os.IsNotExist(err) {
		t.Fatal("Expected file to exist")
	}
<<<<<<< Updated upstream
=======

	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	set.String(flags.WalletDirFlag.Name, importDir, "")
	set.String(flags.WalletPasswordsDirFlag.Name, importPasswordDir, "")
	set.String(flags.BackupDirFlag.Name, exportDir, "")
	set.String(flags.PasswordFileFlag.Name, passwordFilePath, "")
	assert.NoError(t, set.Set(flags.WalletDirFlag.Name, importDir))
	assert.NoError(t, set.Set(flags.WalletPasswordsDirFlag.Name, importPasswordDir))
	assert.NoError(t, set.Set(flags.BackupDirFlag.Name, exportDir))
	assert.NoError(t, set.Set(flags.PasswordFileFlag.Name, passwordFilePath))
	cliCtx := cli.NewContext(&app, set, nil)

>>>>>>> Stashed changes
	require.NoError(t, ImportAccount(cliCtx))
}
