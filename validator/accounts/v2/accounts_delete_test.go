package v2

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
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

	// Write a directory where we will backup accounts to.
	backupDir := filepath.Join(testutil.TempDir(), fmt.Sprintf("/%d", randPath), "backupDir")
	require.NoError(t, os.MkdirAll(backupDir, os.ModePerm))
	t.Cleanup(func() {
		require.NoError(t, os.RemoveAll(keysDir), "Failed to remove directory")
		require.NoError(t, os.RemoveAll(backupDir), "Failed to remove directory")
	})

	// Create 2 keystore files in the keys directory we can then
	// import from in our wallet.
	k1, _ := createKeystore(t, keysDir)
	time.Sleep(time.Second)
	k2, _ := createKeystore(t, keysDir)
	time.Sleep(time.Second)
	k3, _ := createKeystore(t, keysDir)
	generatedPubKeys := []string{k1.Pubkey, k2.Pubkey, k3.Pubkey}
	// Only delete keys 0 and 1.
	deletePublicKeys := strings.Join(generatedPubKeys[0:2], ",")

	// Write a password for the accounts we wish to backup to a file.
	backupPasswordFile := filepath.Join(backupDir, "backuppass.txt")
	err = ioutil.WriteFile(
		backupPasswordFile,
		[]byte("Passw0rdz4938%%"),
		params.BeaconIoConfig().ReadWritePermissions,
	)
	require.NoError(t, err)

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
	wallet, err := NewWallet(cliCtx, v2keymanager.Direct)
	require.NoError(t, err)
	require.NoError(t, wallet.SaveWallet())
	ctx := context.Background()
	encodedCfg, err := direct.MarshalConfigFile(ctx, direct.DefaultConfig())
	require.NoError(t, err)
	require.NoError(t, wallet.WriteKeymanagerConfigToDisk(ctx, encodedCfg))

	// We attempt to import accounts.
	require.NoError(t, ImportAccounts(cliCtx))

	// We attempt to delete the accounts specified.
	require.NoError(t, DeleteAccount(cliCtx))

	keymanager, err := direct.NewKeymanager(
		cliCtx,
		wallet,
		direct.DefaultConfig(),
	)
	require.NoError(t, err)
	remainingAccounts, err := keymanager.FetchValidatingPublicKeys(ctx)
	require.NoError(t, err)
	require.Equal(t, len(remainingAccounts), 1)
	remainingPublicKey, err := hex.DecodeString(k3.Pubkey)
	require.NoError(t, err)
	assert.DeepEqual(t, remainingAccounts[0], bytesutil.ToBytes48(remainingPublicKey))
}
