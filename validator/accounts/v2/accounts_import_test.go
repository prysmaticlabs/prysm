package v2

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/direct"
	keystorev4 "github.com/wealdtech/go-eth2-wallet-encryptor-keystorev4"
)

func TestImport_Noninteractive(t *testing.T) {
	walletDir, passwordsDir, passwordFilePath := setupWalletAndPasswordsDir(t)
	randPath, err := rand.Int(rand.Reader, big.NewInt(1000000))
	require.NoError(t, err, "Could not generate random file path")
	keysDir := filepath.Join(testutil.TempDir(), fmt.Sprintf("/%d", randPath), "keysDir")
	require.NoError(t, os.MkdirAll(keysDir, os.ModePerm))
	t.Cleanup(func() {
		require.NoError(t, os.RemoveAll(keysDir), "Failed to remove directory")
	})

	cliCtx := setupWalletCtx(t, &testWalletConfig{
		walletDir:           walletDir,
		passwordsDir:        passwordsDir,
		keysDir:             keysDir,
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
	keymanager, err := direct.NewKeymanager(
		ctx,
		wallet,
		direct.DefaultConfig(),
	)
	require.NoError(t, err)

	// Make sure there are no accounts at the start.
	accounts, err := keymanager.ValidatingAccountNames()
	require.NoError(t, err)
	assert.Equal(t, len(accounts), 0)

	// Create 2 keys.
	createKeystore(t, keysDir)
	time.Sleep(time.Second)
	createKeystore(t, keysDir)

	require.NoError(t, ImportAccount(cliCtx))

	wallet, err = OpenWallet(cliCtx)
	require.NoError(t, err)
	km, err := wallet.InitializeKeymanager(ctx, true)
	require.NoError(t, err)
	keys, err := km.FetchValidatingPublicKeys(ctx)
	require.NoError(t, err)

	assert.Equal(t, 2, len(keys))
}

func TestImport_Noninteractive_Filepath(t *testing.T) {
	walletDir, passwordsDir, passwordFilePath := setupWalletAndPasswordsDir(t)
	randPath, err := rand.Int(rand.Reader, big.NewInt(1000000))
	require.NoError(t, err, "Could not generate random file path")
	keysDir := filepath.Join(testutil.TempDir(), fmt.Sprintf("/%d", randPath), "keysDir")
	require.NoError(t, os.MkdirAll(keysDir, os.ModePerm))
	t.Cleanup(func() {
		require.NoError(t, os.RemoveAll(keysDir), "Failed to remove directory")
	})

	cliCtx := setupWalletCtx(t, &testWalletConfig{
		walletDir:           walletDir,
		passwordsDir:        passwordsDir,
		keysDir:             createKeystore(t, keysDir), // Using direct filepath to the new keystore.
		keymanagerKind:      v2keymanager.Direct,
		walletPasswordFile:  passwordFilePath,
		accountPasswordFile: passwordFilePath,
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

	// Make sure there are no accounts at the start.
	accounts, err := keymanager.ValidatingAccountNames()
	require.NoError(t, err)
	assert.Equal(t, len(accounts), 0)

	require.NoError(t, ImportAccount(cliCtx))

	wallet, err = OpenWallet(cliCtx)
	require.NoError(t, err)
	km, err := wallet.InitializeKeymanager(ctx, true)
	require.NoError(t, err)
	keys, err := km.FetchValidatingPublicKeys(ctx)
	require.NoError(t, err)

	assert.Equal(t, 1, len(keys))
}

// Returns the fullPath to the newly created keystore file.
func createKeystore(t *testing.T, path string) string {
	validatingKey := bls.RandKey()
	encryptor := keystorev4.New()
	cryptoFields, err := encryptor.Encrypt(validatingKey.Marshal(), password)
	require.NoError(t, err)
	id, err := uuid.NewRandom()
	require.NoError(t, err)
	keystoreFile := &v2keymanager.Keystore{
		Crypto:  cryptoFields,
		ID:      id.String(),
		Pubkey:  fmt.Sprintf("%x", validatingKey.PublicKey().Marshal()),
		Version: encryptor.Version(),
		Name:    encryptor.Name(),
	}
	encoded, err := json.MarshalIndent(keystoreFile, "", "\t")
	require.NoError(t, err)
	// Write the encoded keystore to disk with the timestamp appended
	createdAt := roughtime.Now().Unix()
	fullPath := filepath.Join(path, fmt.Sprintf(direct.KeystoreFileNameFormat, createdAt))
	require.NoError(t, ioutil.WriteFile(fullPath, encoded, os.ModePerm))
	return fullPath
}
