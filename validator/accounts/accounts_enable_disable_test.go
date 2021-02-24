package accounts

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
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/prysmaticlabs/prysm/validator/accounts/iface"
	"github.com/prysmaticlabs/prysm/validator/accounts/wallet"
	"github.com/prysmaticlabs/prysm/validator/keymanager"
)

func TestDisableAccounts_Noninteractive(t *testing.T) {
	walletDir, _, passwordFilePath := setupWalletAndPasswordsDir(t)
	randPath, err := rand.Int(rand.Reader, big.NewInt(1000000))
	require.NoError(t, err, "Could not generate random file path")
	// Write a directory where we will import keys from.
	keysDir := filepath.Join(t.TempDir(), fmt.Sprintf("/%d", randPath), "keysDir")
	require.NoError(t, os.MkdirAll(keysDir, os.ModePerm))

	// Create 3 keystore files in the keys directory we can then
	// import from in our wallet.
	k1, _ := createKeystore(t, keysDir)
	time.Sleep(time.Second)
	k2, _ := createKeystore(t, keysDir)
	time.Sleep(time.Second)
	k3, _ := createKeystore(t, keysDir)
	generatedPubKeys := []string{k1.Pubkey, k2.Pubkey, k3.Pubkey}
	// Only disable keys 0 and 1.
	disablePublicKeys := strings.Join(generatedPubKeys[0:2], ",")
	// We initialize a wallet with a imported keymanager.
	cliCtx := setupWalletCtx(t, &testWalletConfig{
		// Wallet configuration flags.
		walletDir:           walletDir,
		keymanagerKind:      keymanager.Imported,
		walletPasswordFile:  passwordFilePath,
		accountPasswordFile: passwordFilePath,
		// Flags required for ImportAccounts to work.
		keysDir: keysDir,
		// Flags required for DisableAccounts to work.
		disablePublicKeys: disablePublicKeys,
	})
	w, err := CreateWalletWithKeymanager(cliCtx.Context, &CreateWalletConfig{
		WalletCfg: &wallet.Config{
			WalletDir:      walletDir,
			KeymanagerKind: keymanager.Imported,
			WalletPassword: password,
		},
	})
	require.NoError(t, err)

	// We attempt to import accounts.
	require.NoError(t, ImportAccountsCli(cliCtx))

	// We attempt to disable the accounts specified.
	require.NoError(t, DisableAccountsCli(cliCtx))

	keymanager, err := w.InitializeKeymanager(cliCtx.Context, iface.InitKeymanagerConfig{ListenForChanges: false})
	require.NoError(t, err)
	remainingAccounts, err := keymanager.FetchValidatingPublicKeys(cliCtx.Context)
	require.NoError(t, err)
	require.Equal(t, len(remainingAccounts), 1)
	remainingPublicKey, err := hex.DecodeString(k3.Pubkey)
	require.NoError(t, err)
	assert.DeepEqual(t, remainingAccounts[0], bytesutil.ToBytes48(remainingPublicKey))
}

func TestEnableAccounts_Noninteractive(t *testing.T) {
	walletDir, _, passwordFilePath := setupWalletAndPasswordsDir(t)
	randPath, err := rand.Int(rand.Reader, big.NewInt(1000000))
	require.NoError(t, err, "Could not generate random file path")
	// Write a directory where we will import keys from.
	keysDir := filepath.Join(t.TempDir(), fmt.Sprintf("/%d", randPath), "keysDir")
	require.NoError(t, os.MkdirAll(keysDir, os.ModePerm))

	// Create 3 keystore files in the keys directory we can then
	// import from in our wallet.
	k1, _ := createKeystore(t, keysDir)
	time.Sleep(time.Second)
	k2, _ := createKeystore(t, keysDir)
	time.Sleep(time.Second)
	k3, _ := createKeystore(t, keysDir)
	generatedPubKeys := []string{k1.Pubkey, k2.Pubkey, k3.Pubkey}
	// Disable all keys.
	disablePublicKeys := strings.Join(generatedPubKeys, ",")
	// Only enable keys 0 and 1.
	enablePublicKeys := strings.Join(generatedPubKeys[0:2], ",")
	// We initialize a wallet with a imported keymanager.
	cliCtx := setupWalletCtx(t, &testWalletConfig{
		walletDir:           walletDir,
		keymanagerKind:      keymanager.Imported,
		walletPasswordFile:  passwordFilePath,
		accountPasswordFile: passwordFilePath,
		keysDir:             keysDir,
		disablePublicKeys:   disablePublicKeys,
		// Flags required for EnableAccounts to work.
		enablePublicKeys: enablePublicKeys,
	})
	w, err := CreateWalletWithKeymanager(cliCtx.Context, &CreateWalletConfig{
		WalletCfg: &wallet.Config{
			WalletDir:      walletDir,
			KeymanagerKind: keymanager.Imported,
			WalletPassword: password,
		},
	})
	require.NoError(t, err)

	// We attempt to import accounts.
	require.NoError(t, ImportAccountsCli(cliCtx))

	// We attempt to disable the accounts specified.
	require.NoError(t, DisableAccountsCli(cliCtx))

	km, err := w.InitializeKeymanager(cliCtx.Context, iface.InitKeymanagerConfig{ListenForChanges: false})
	require.NoError(t, err)
	remainingAccounts, err := km.FetchValidatingPublicKeys(cliCtx.Context)
	require.NoError(t, err)
	require.Equal(t, len(remainingAccounts), 0)

	// We attempt to enable the accounts specified.
	require.NoError(t, EnableAccountsCli(cliCtx))

	km, err = w.InitializeKeymanager(cliCtx.Context, iface.InitKeymanagerConfig{ListenForChanges: false})
	require.NoError(t, err)
	remainingAccounts, err = km.FetchValidatingPublicKeys(cliCtx.Context)
	require.NoError(t, err)
	require.Equal(t, len(remainingAccounts), 2)
	remainingPublicKey1, err := hex.DecodeString(k1.Pubkey)
	require.NoError(t, err)
	remainingPublicKey2, err := hex.DecodeString(k2.Pubkey)
	require.NoError(t, err)
	assert.DeepEqual(t, remainingAccounts[0], bytesutil.ToBytes48(remainingPublicKey1))
	assert.DeepEqual(t, remainingAccounts[1], bytesutil.ToBytes48(remainingPublicKey2))
}
