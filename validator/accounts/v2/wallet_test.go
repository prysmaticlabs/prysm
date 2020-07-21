package v2

import (
	"context"
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"path"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	mock "github.com/prysmaticlabs/prysm/validator/keymanager/v2/testing"
	"github.com/sirupsen/logrus"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(ioutil.Discard)
}

func setupWalletDir(t testing.TB) (string, string) {
	randPath, err := rand.Int(rand.Reader, big.NewInt(1000000))
	require.NoError(t, err, "Could not generate random file path")
	walletDir := path.Join(testutil.TempDir(), fmt.Sprintf("/%d", randPath))
	require.NoError(t, os.RemoveAll(walletDir), "Failed to remove directory")
	passwordsDir := path.Join(testutil.TempDir(), fmt.Sprintf("/%d", randPath))
	require.NoError(t, os.RemoveAll(passwordsDir), "Failed to remove directory")
	t.Cleanup(func() {
		require.NoError(t, os.RemoveAll(walletDir), "Failed to remove directory")
		require.NoError(t, os.RemoveAll(passwordsDir), "Failed to remove directory")
	})
	return walletDir, passwordsDir
}

func TestCreateAndReadWallet(t *testing.T) {
	ctx := context.Background()
	if _, err := NewWallet(ctx, &WalletConfig{
		PasswordsDir: "",
		WalletDir:    "",
	}); err == nil {
		t.Error("Expected error when passing in empty directories, received nil")
	}
	walletDir, passwordsDir := setupWalletDir(t)
	keymanagerKind := v2keymanager.Direct
	wallet, err := NewWallet(ctx, &WalletConfig{
		PasswordsDir:   passwordsDir,
		WalletDir:      walletDir,
		KeymanagerKind: keymanagerKind,
	})
	require.NoError(t, err)

	keymanager := &mock.MockKeymanager{
		ConfigFileContents: []byte("hello-world"),
	}
	keymanagerConfig, err := keymanager.MarshalConfigFile(ctx)
	require.NoError(t, err, "Could not marshal keymanager config file")
	require.NoError(t, wallet.WriteKeymanagerConfigToDisk(ctx, keymanagerConfig), "Could not write keymanager config file to disk")

	walletPath := path.Join(walletDir, keymanagerKind.String())
	configFilePath := path.Join(walletPath, KeymanagerConfigFileName)
	require.Equal(t, true, fileExists(configFilePath), "Expected config file to have been created at path: %s", configFilePath)

	// We should be able to now read the wallet as well.
	_, err = NewWallet(ctx, &WalletConfig{
		PasswordsDir: passwordsDir,
		WalletDir:    walletDir,
	})
	require.NoError(t, err)
}
