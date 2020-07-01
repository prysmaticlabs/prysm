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
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/direct"

	"github.com/sirupsen/logrus"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(ioutil.Discard)
}

var _ = direct.Wallet(&Wallet{})

type mockKeymanager struct {
	configFileContents []byte
}

func (m *mockKeymanager) CreateAccount(ctx context.Context, password string) error {
	return nil
}

func (m *mockKeymanager) MarshalConfigFile(ctx context.Context) ([]byte, error) {
	return m.configFileContents, nil
}

func setupWalletDir(t testing.TB) (string, string) {
	randPath, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		t.Fatalf("Could not generate random file path: %v", err)
	}
	walletDir := path.Join(testutil.TempDir(), fmt.Sprintf("/%d", randPath))
	if err := os.RemoveAll(walletDir); err != nil {
		t.Fatalf("Failed to remove directory: %v", err)
	}
	passwordsDir := path.Join(testutil.TempDir(), fmt.Sprintf("/%d", randPath))
	if err := os.RemoveAll(passwordsDir); err != nil {
		t.Fatalf("Failed to remove directory: %v", err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(walletDir); err != nil {
			t.Fatalf("Failed to remove directory: %v", err)
		}
		if err := os.RemoveAll(passwordsDir); err != nil {
			t.Fatalf("Failed to remove directory: %v", err)
		}
	})
	return walletDir, passwordsDir
}

func TestCreateAndReadWallet(t *testing.T) {
	ctx := context.Background()
	if _, err := CreateWallet(ctx, &WalletConfig{
		PasswordsDir: "",
		WalletDir:    "",
	}); err == nil {
		t.Error("Expected error when passing in empty directories, received nil")
	}
	walletDir, passwordsDir := setupWalletDir(t)
	keymanagerKind := v2keymanager.Direct
	wallet, err := CreateWallet(ctx, &WalletConfig{
		PasswordsDir:   passwordsDir,
		WalletDir:      walletDir,
		KeymanagerKind: keymanagerKind,
	})
	if err != nil {
		t.Fatal(err)
	}

	keymanager := &mockKeymanager{
		configFileContents: []byte("hello-world"),
	}
	keymanagerConfig, err := keymanager.MarshalConfigFile(ctx)
	if err != nil {
		t.Fatalf("Could not marshal keymanager config file: %v", err)
	}
	if err := wallet.WriteKeymanagerConfigToDisk(ctx, keymanagerConfig); err != nil {
		t.Fatalf("Could not write keymanager config file to disk: %v", err)
	}

	walletPath := path.Join(walletDir, keymanagerKind.String())
	configFilePath := path.Join(walletPath, keymanagerConfigFileName)
	if !fileExists(configFilePath) {
		t.Fatalf("Expected config file to have been created at path: %s", configFilePath)
	}

	// We should be able to now read the wallet as well.
	if _, err := CreateWallet(ctx, &WalletConfig{
		PasswordsDir: passwordsDir,
		WalletDir:    walletDir,
	}); err != nil {
		t.Fatal(err)
	}
}
