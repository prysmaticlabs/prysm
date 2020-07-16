package v2

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil"
	v2 "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
)

func setupWallet(t *testing.T) *Wallet {
	testDir := testutil.TempDir()
	ctx := context.Background()
	if err := initializeDirectWallet(testDir, testDir); err != nil {
		t.Fatal(err)
	}
	cfg := &WalletConfig{
		PasswordsDir:   testDir,
		WalletDir:      testDir,
		KeymanagerKind: v2.Direct,
	}
	w, err := NewWallet(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	keymanager, err := w.InitializeKeymanager(ctx, true)
	if err != nil {
		t.Fatalf("Could not initialize keymanager: %v", err)
	}
	password := "10testPass!"
	if _, err := keymanager.CreateAccount(ctx, password); err != nil {
		t.Fatalf("Could not create account in wallet: %v", err)
	}
	return w
}

func TestZipAndUnzip(t *testing.T) {
	testDir := testutil.TempDir()
	wallet := setupWallet(t)

	accounts, err := wallet.AccountNames()
	if err != nil {
		t.Fatal(err)
	}
	if len(accounts) == 0 {
		t.Fatal("Expected more accounts, received 0")
	}
	if err := wallet.zipAccounts(accounts, testDir); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(testDir, archiveFilename)); os.IsNotExist(err) {
		t.Fatal("Expected file to exist")
	}

	exportFolder := "export"
	exportDir := filepath.Join(testDir, exportFolder)
	importedAccounts, err := unzipArchiveToTarget(testDir, exportDir)
	if err != nil {
		t.Fatal(err)
	}

	allAccountsStr := strings.Join(accounts, " ")
	for _, importedAccount := range importedAccounts {
		if !strings.Contains(allAccountsStr, importedAccount) {
			t.Fatalf("Expected %s to be in %s", importedAccount, allAccountsStr)
		}
	}
}
