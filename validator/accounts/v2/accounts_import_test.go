package v2

import (
	"context"
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
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
	exportDir := filepath.Join(testutil.TempDir(), fmt.Sprintf("/%d", randPath), "export")
	importDir := filepath.Join(testutil.TempDir(), fmt.Sprintf("/%d", randPath), "import")
	importPasswordDir := filepath.Join(testutil.TempDir(), fmt.Sprintf("/%d", randPath), "importpassword")
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
	wallet, err := NewWallet(cliCtx, v2keymanager.Direct)
	require.NoError(t, err)
	require.NoError(t, wallet.SaveWallet())
	ctx := context.Background()
	keymanagerCfg := direct.DefaultConfig()
	encodedCfg, err := direct.MarshalConfigFile(ctx, keymanagerCfg)
	require.NoError(t, err)
	require.NoError(t, wallet.WriteKeymanagerConfigToDisk(ctx, encodedCfg))
	keymanager, err := direct.NewKeymanager(
		ctx,
		wallet,
		keymanagerCfg,
	)
	require.NoError(t, err)
	_, err = keymanager.CreateAccount(ctx, password)
	require.NoError(t, err)

	accounts, err := keymanager.ValidatingAccountNames()
	require.NoError(t, err)
	assert.Equal(t, len(accounts), 1)

	require.NoError(t, wallet.zipAccounts(accounts, exportDir))
	if _, err := os.Stat(filepath.Join(exportDir, archiveFilename)); os.IsNotExist(err) {
		t.Fatal("Expected file to exist")
	}

	require.NoError(t, os.RemoveAll(walletDir), "Failed to remove directory")
	require.NoError(t, ImportAccount(cliCtx))

	wallet, err = OpenWallet(cliCtx)
	require.NoError(t, err)
	km, err := wallet.InitializeKeymanager(ctx, true)
	require.NoError(t, err)
	keys, err := km.FetchValidatingPublicKeys(ctx)
	require.NoError(t, err)

	assert.Equal(t, len(keys), 1)
}
