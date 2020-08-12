package v2

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/direct"
)

func TestDeleteAccounts_Noninteractive(t *testing.T) {
	walletDir, _, passwordFilePath := setupWalletAndPasswordsDir(t)
	ctx := context.Background()

	// We initialize a wallet with a direct keymanager.
	cliCtx := setupWalletCtx(t, &testWalletConfig{
		// Wallet configuration flags.
		walletDir:           walletDir,
		keymanagerKind:      v2keymanager.Direct,
		walletPasswordFile:  passwordFilePath,
		accountPasswordFile: passwordFilePath,
	})
	wallet, err := NewWallet(cliCtx, v2keymanager.Direct)
	require.NoError(t, err)
	require.NoError(t, wallet.SaveWallet())
	encodedCfg, err := direct.MarshalConfigFile(ctx, direct.DefaultConfig())
	require.NoError(t, err)
	require.NoError(t, wallet.WriteKeymanagerConfigToDisk(ctx, encodedCfg))
	keymanager, err := wallet.InitializeKeymanager(cliCtx, true /*skipMnemonicConfirm8*/)
	require.NoError(t, err)
	km, ok := keymanager.(*direct.Keymanager)
	if !ok {
		t.Fatal("not a direct keymanager")
	}
	_, err = km.CreateAccount(ctx)
	require.NoError(t, err)
	_, err = km.CreateAccount(ctx)
	require.NoError(t, err)
	_, err = km.CreateAccount(ctx)
	require.NoError(t, err)

	accounts, err := km.FetchValidatingPublicKeys(ctx)
	require.NoError(t, err)
	require.Equal(t, 3, len(accounts))
	allPublicKeys := make([]string, len(accounts))
	for i, account := range accounts {
		allPublicKeys[i] = fmt.Sprintf("%#x", account)
	}
	deletePublicKeysStr := strings.Join(allPublicKeys[:1], ",")
	cliCtx = setupWalletCtx(t, &testWalletConfig{
		// Wallet configuration flags.
		walletDir:           walletDir,
		keymanagerKind:      v2keymanager.Direct,
		walletPasswordFile:  passwordFilePath,
		accountPasswordFile: passwordFilePath,
		// Flags required for BackupAccounts to work.
		deletePublicKeys: deletePublicKeysStr,
	})

	keymanager, err = wallet.InitializeKeymanager(cliCtx, true /*skipMnemonicConfirm8*/)
	require.NoError(t, err)
	km, ok = keymanager.(*direct.Keymanager)
	require.Equal(t, true, ok)
	deletePublicKeysBytes := make([][]byte, 2)
	for i, account := range accounts[:1] {
		deletePublicKeysBytes[i] = account[:]
	}
	require.NoError(t, km.DeleteAccounts(ctx, deletePublicKeysBytes))

	newAccounts, err := km.FetchValidatingPublicKeys(ctx)
	require.NoError(t, err)

	require.Equal(t, len(newAccounts), 1)
	require.Equal(t, newAccounts[0], accounts[2])
}
