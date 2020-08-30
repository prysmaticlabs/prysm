package v2

import (
	"bytes"
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
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestExitAccountsCli_Ok(t *testing.T) {
	logHook := logTest.NewGlobal()

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
	_, err = CreateWalletWithKeymanager(cliCtx.Context, &CreateWalletConfig{
		WalletCfg: &WalletConfig{
			WalletDir:      walletDir,
			KeymanagerKind: v2keymanager.Direct,
			WalletPassword: "Passwordz0320$",
		},
	})
	require.NoError(t, err)

	require.NoError(t, ImportAccountsCli(cliCtx))

	// Prepare user input for final confirmation step
	var stdin bytes.Buffer
	stdin.Write([]byte("Y\n"))

	require.NoError(t, ExitAccountsCli(cliCtx, &stdin))
	assert.LogsContain(t, logHook, "Voluntary exit was successful")
}

func TestExitAccountsCli_EmptyWalletReturnsError(t *testing.T) {
	walletDir, _, passwordFilePath := setupWalletAndPasswordsDir(t)
	cliCtx := setupWalletCtx(t, &testWalletConfig{
		walletDir:           walletDir,
		keymanagerKind:      v2keymanager.Direct,
		walletPasswordFile:  passwordFilePath,
		accountPasswordFile: passwordFilePath,
	})
	_, err := CreateWalletWithKeymanager(cliCtx.Context, &CreateWalletConfig{
		WalletCfg: &WalletConfig{
			WalletDir:      walletDir,
			KeymanagerKind: v2keymanager.Direct,
			WalletPassword: "Passwordz0320$",
		},
	})
	require.NoError(t, err)
	err = ExitAccountsCli(cliCtx, os.Stdin)
	assert.ErrorContains(t, "wallet is empty, no accounts to perform voluntary exit", err)
}
