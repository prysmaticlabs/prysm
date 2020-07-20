package v2

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/direct"
	mock "github.com/prysmaticlabs/prysm/validator/keymanager/v2/testing"
)

func TestListAccounts_DirectKeymanager(t *testing.T) {
	walletDir, passwordsDir := setupWalletDir(t)
	keymanagerKind := v2keymanager.Direct
	ctx := context.Background()
	wallet, err := NewWallet(ctx, &WalletConfig{
		PasswordsDir:   passwordsDir,
		WalletDir:      walletDir,
		KeymanagerKind: keymanagerKind,
	})
	require.NoError(t, err)
	numAccounts := 5
	accountNames := make([]string, numAccounts)
	pubKeys := make([][48]byte, numAccounts)
	password := "passw0rd2020%%"
	depositDataForAccounts := make([][]byte, numAccounts)
	for i := 0; i < numAccounts; i++ {
		// Generate a new account name and write the account
		// to disk using the wallet.
		name, err := wallet.generateAccountName()
		require.NoError(t, err)
		accountNames[i] = name
		// Generate a directory for the account name and
		// write its associated password to disk.
		accountPath := path.Join(wallet.accountsPath, name)
		require.NoError(t, os.MkdirAll(accountPath, DirectoryPermissions))
		require.NoError(t, wallet.writePasswordToFile(name, password))

		// Write the deposit data for each account.
		depositData := []byte(strconv.Itoa(i))
		depositDataForAccounts[i] = depositData
		require.NoError(t, wallet.WriteFileForAccount(ctx, name, direct.DepositTransactionFileName, depositData))

		// Write the creation timestamp for the account with unix timestamp 0.
		require.NoError(t, wallet.WriteFileForAccount(ctx, name, direct.TimestampFileName, []byte("0")))

		// Create public keys for the accounts.
		key := bls.RandKey()
		pubKeys[i] = bytesutil.ToBytes48(key.PublicKey().Marshal())
	}
	rescueStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	keymanager := &mock.MockKeymanager{
		PublicKeys: pubKeys,
	}
	// We call the list direct keymanager accounts function.
	require.NoError(t, listDirectKeymanagerAccounts(true /* show deposit data */, wallet, keymanager))

	require.NoError(t, w.Close())
	out, err := ioutil.ReadAll(r)
	require.NoError(t, err)
	os.Stdout = rescueStdout

	// Assert the keymanager kind is printed to stdout.
	stringOutput := string(out)
	if !strings.Contains(stringOutput, wallet.KeymanagerKind().String()) {
		t.Error("Did not find Keymanager kind in output")
	}

	// Assert the wallet and passwords paths are in stdout.
	if !strings.Contains(stringOutput, wallet.accountsPath) {
		t.Errorf("Did not find accounts path %s in output", wallet.accountsPath)
	}

	for i := 0; i < numAccounts; i++ {
		accountName := accountNames[i]
		// Assert the account name is printed to stdout.
		if !strings.Contains(stringOutput, accountName) {
			t.Errorf("Did not find account %s in output", accountName)
		}
		key := pubKeys[i]
		depositData := depositDataForAccounts[i]

		// Assert every public key is printed to stdout.
		if !strings.Contains(stringOutput, fmt.Sprintf("%#x", key)) {
			t.Errorf("Did not find pubkey %#x in output", key)
		}

		// Assert the deposit data for the account is printed to stdout.
		if !strings.Contains(stringOutput, fmt.Sprintf("%#x", depositData)) {
			t.Errorf("Did not find deposit data %#x in output", depositData)
		}

		// Assert the account creation time is displayed
		if !strings.Contains(stringOutput, fmt.Sprintf("%v", time.Unix(0, 0).String())) {
			t.Error("Did not display account creation timestamp")
		}
	}
}
