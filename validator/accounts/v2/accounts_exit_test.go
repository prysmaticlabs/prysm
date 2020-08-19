package v2

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/direct"
)

func TestExitAccounts_Ok(t *testing.T) {
	walletDir, _, passwordFilePath := setupWalletAndPasswordsDir(t)
	randPath, err := rand.Int(rand.Reader, big.NewInt(1000000))
	require.NoError(t, err, "Could not generate random file path")
	// Write a directory where we will import keys from.
	keysDir := filepath.Join(testutil.TempDir(), fmt.Sprintf("/%d", randPath), "keysDir")
	require.NoError(t, os.MkdirAll(keysDir, os.ModePerm))

	// Create keystore file in the keys directory we can then import from in our wallet.
	keystore, _ := createKeystore(t, keysDir)
	time.Sleep(time.Second)

	// We initialize a wallet with a direct keymanager.
	cliCtx := setupWalletCtx(t, &testWalletConfig{
		// Wallet configuration flags.
		walletDir:           walletDir,
		keymanagerKind:      v2keymanager.Direct,
		walletPasswordFile:  passwordFilePath,
		accountPasswordFile: passwordFilePath,
		// Flag required for ImportAccounts to work.
		keysDir: keysDir,
		// Flag required for ExitAccounts to work.
		voluntaryExitPublicKeys: keystore.Pubkey,
	})
	wallet, err := NewWallet(cliCtx, v2keymanager.Direct)
	require.NoError(t, err)
	require.NoError(t, wallet.SaveWallet())
	ctx := context.Background()
	encodedCfg, err := direct.MarshalConfigFile(ctx, direct.DefaultConfig())
	require.NoError(t, err)
	require.NoError(t, wallet.WriteKeymanagerConfigToDisk(ctx, encodedCfg))

	require.NoError(t, ImportAccounts(cliCtx))

	var stdin bytes.Buffer
	stdin.Write([]byte("Y\n"))
	require.NoError(t, ExitAccounts(cliCtx, &stdin))
}

func TestExitAccounts_EmptyWalletReturnsError(t *testing.T) {
	walletDir, _, passwordFilePath := setupWalletAndPasswordsDir(t)
	cliCtx := setupWalletCtx(t, &testWalletConfig{
		walletDir:           walletDir,
		keymanagerKind:      v2keymanager.Direct,
		walletPasswordFile:  passwordFilePath,
		accountPasswordFile: passwordFilePath,
	})
	wallet, err := NewWallet(cliCtx, v2keymanager.Direct)
	require.NoError(t, err)
	require.NoError(t, wallet.SaveWallet())

	ctx := context.Background()
	encodedCfg, err := direct.MarshalConfigFile(ctx, direct.DefaultConfig())
	require.NoError(t, err)
	require.NoError(t, wallet.WriteKeymanagerConfigToDisk(ctx, encodedCfg))

	err = ExitAccounts(cliCtx, os.Stdin)
	assert.ErrorContains(t, "wallet is empty, no accounts to perform voluntary exit", err)
}
