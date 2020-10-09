package v2

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/prysmaticlabs/prysm/validator/accounts/v2/wallet"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/direct"
)

func TestDeleteAccounts_Noninteractive(t *testing.T) {
	walletDir, _, passwordFilePath := setupWalletAndPasswordsDir(t)
	randPath, err := rand.Int(rand.Reader, big.NewInt(1000000))
	require.NoError(t, err, "Could not generate random file path")
	// Write a directory where we will import keys from.
	keysDir := filepath.Join(testutil.TempDir(), fmt.Sprintf("/%d", randPath), "keysDir")
	require.NoError(t, os.MkdirAll(keysDir, os.ModePerm))

	// Create 3 keystore files in the keys directory we can then
	// import from in our wallet.
	k1, _ := createKeystore(t, keysDir)
	time.Sleep(time.Second)
	k2, _ := createKeystore(t, keysDir)
	time.Sleep(time.Second)
	k3, _ := createKeystore(t, keysDir)
	generatedPubKeys := []string{k1.Pubkey, k2.Pubkey, k3.Pubkey}
	// Only delete keys 0 and 1.
	deletePublicKeys := strings.Join(generatedPubKeys[0:2], ",")

	// We initialize a wallet with a direct keymanager.
	cliCtx := setupWalletCtx(t, &testWalletConfig{
		// Wallet configuration flags.
		walletDir:           walletDir,
		keymanagerKind:      v2keymanager.Direct,
		walletPasswordFile:  passwordFilePath,
		accountPasswordFile: passwordFilePath,
		// Flags required for ImportAccounts to work.
		keysDir: keysDir,
		// Flags required for DeleteAccounts to work.
		deletePublicKeys: deletePublicKeys,
	})
	w, err := CreateWalletWithKeymanager(cliCtx.Context, &CreateWalletConfig{
		WalletCfg: &wallet.Config{
			WalletDir:      walletDir,
			KeymanagerKind: v2keymanager.Direct,
			WalletPassword: password,
		},
	})
	require.NoError(t, err)
	require.NoError(t, w.SaveHashedPassword(cliCtx.Context))

	// We attempt to import accounts.
	require.NoError(t, ImportAccountsCli(cliCtx))

	// We attempt to delete the accounts specified.
	require.NoError(t, DeleteAccountCli(cliCtx))

	keymanager, err := direct.NewKeymanager(
		cliCtx.Context,
		&direct.SetupConfig{
			Wallet: w,
			Opts:   direct.DefaultKeymanagerOpts(),
		},
	)
	require.NoError(t, err)
	remainingAccounts, err := keymanager.FetchValidatingPublicKeys(cliCtx.Context)
	require.NoError(t, err)
	require.Equal(t, len(remainingAccounts), 1)
	remainingPublicKey, err := hex.DecodeString(k3.Pubkey)
	require.NoError(t, err)
	assert.DeepEqual(t, remainingAccounts[0], bytesutil.ToBytes48(remainingPublicKey))
}
