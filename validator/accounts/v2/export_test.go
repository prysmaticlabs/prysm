package v2

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/direct"
)

func TestZipAndUnzip(t *testing.T) {
	walletDir, passwordsDir := setupWalletAndPasswordsDir(t)
	randPath, err := rand.Int(rand.Reader, big.NewInt(1000000))
	require.NoError(t, err, "Could not generate random file path")
	exportDir := path.Join(testutil.TempDir(), fmt.Sprintf("/%d", randPath), "export")
	importDir := path.Join(testutil.TempDir(), fmt.Sprintf("/%d", randPath), "import")
	t.Cleanup(func() {
		require.NoError(t, os.RemoveAll(exportDir), "Failed to remove directory")
		require.NoError(t, os.RemoveAll(importDir), "Failed to remove directory")
	})
	cliCtx := setupWalletCtx(t, &testWalletConfig{
		walletDir:      walletDir,
		passwordsDir:   passwordsDir,
		exportDir:      exportDir,
		keymanagerKind: v2keymanager.Direct,
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
	walletDir, passwordsDir := setupWalletAndPasswordsDir(t)
	randPath, err := rand.Int(rand.Reader, big.NewInt(1000000))
	require.NoError(t, err, "Could not generate random file path")
	exportDir := path.Join(testutil.TempDir(), fmt.Sprintf("/%d", randPath), "export")
	t.Cleanup(func() {
		require.NoError(t, os.RemoveAll(exportDir), "Failed to remove directory")
	})
	accounts := "all"
	cliCtx := setupWalletCtx(t, &testWalletConfig{
		walletDir:        walletDir,
		passwordsDir:     passwordsDir,
		exportDir:        exportDir,
		accountsToExport: accounts,
		keymanagerKind:   v2keymanager.Direct,
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
	require.NoError(t, ExportAccount(cliCtx))
	if _, err := os.Stat(filepath.Join(exportDir, archiveFilename)); os.IsNotExist(err) {
		t.Fatal("Expected file to exist")
	}
}
