package v2

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/derived"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/direct"
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
	keymanager, err := direct.NewKeymanager(ctx, wallet, direct.DefaultConfig(), true /* skip confirm */)
	require.NoError(t, err)
	numAccounts := 5
	depositDataForAccounts := make([][]byte, numAccounts)
	accountCreationTimestamps := make([][]byte, numAccounts)
	for i := 0; i < numAccounts; i++ {
		accountName, err := keymanager.CreateAccount(ctx, "hello world")
		require.NoError(t, err)
		depositData, err := wallet.ReadFileForAccount(accountName, direct.DepositTransactionFileName)
		require.NoError(t, err)
		depositDataForAccounts[i] = depositData
		unixTimestamp, err := wallet.ReadFileForAccount(accountName, direct.TimestampFileName)
		require.NoError(t, err)
		accountCreationTimestamps[i] = unixTimestamp
	}
	rescueStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

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

	accountNames, err := wallet.AccountNames()
	require.NoError(t, err)
	pubKeys, err := keymanager.FetchValidatingPublicKeys(ctx)
	require.NoError(t, err)

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
		unixTimestampStr, err := strconv.ParseInt(string(accountCreationTimestamps[i]), 10, 64)
		require.NoError(t, err)
		unixTimestamp := time.Unix(unixTimestampStr, 0)
		assert.Equal(t, strings.Contains(stringOutput, humanize.Time(unixTimestamp)), true)
	}
}

func TestListAccounts_DerivedKeymanager(t *testing.T) {
	walletDir, passwordsDir := setupWalletDir(t)
	keymanagerKind := v2keymanager.Derived
	ctx := context.Background()
	wallet, err := NewWallet(ctx, &WalletConfig{
		PasswordsDir:   passwordsDir,
		WalletDir:      walletDir,
		KeymanagerKind: keymanagerKind,
	})
	require.NoError(t, err)

	password := "hello world"
	seedConfig, err := derived.InitializeWalletSeedFile(ctx, password, true /* skip confirm */)
	require.NoError(t, err)

	// Create a new wallet seed file and write it to disk.
	seedConfigFile, err := derived.MarshalEncryptedSeedFile(ctx, seedConfig)
	require.NoError(t, err)
	require.NoError(t, wallet.WriteEncryptedSeedToDisk(ctx, seedConfigFile))

	keymanager, err := derived.NewKeymanager(
		ctx,
		wallet,
		derived.DefaultConfig(),
		true, /* skip confirm */
		password,
	)
	require.NoError(t, err)
	numAccounts := 5
	depositDataForAccounts := make([][]byte, numAccounts)
	accountCreationTimestamps := make([][]byte, numAccounts)
	for i := 0; i < numAccounts; i++ {
		_, err := keymanager.CreateAccount(ctx, password)
		require.NoError(t, err)
		withdrawalKeyPath := fmt.Sprintf(derived.WithdrawalKeyDerivationPathTemplate, i)
		depositData, err := wallet.ReadFileAtPath(ctx, withdrawalKeyPath, direct.DepositTransactionFileName)
		require.NoError(t, err)
		depositDataForAccounts[i] = depositData
		unixTimestamp, err := wallet.ReadFileAtPath(ctx, withdrawalKeyPath, direct.TimestampFileName)
		require.NoError(t, err)
		accountCreationTimestamps[i] = unixTimestamp
	}

	rescueStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	// We call the list direct keymanager accounts function.
	require.NoError(t, listDerivedKeymanagerAccounts(true /* show deposit data */, wallet, keymanager))

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

	accountNames, err := keymanager.ValidatingAccountNames(ctx)
	require.NoError(t, err)
	pubKeys, err := keymanager.FetchValidatingPublicKeys(ctx)
	require.NoError(t, err)

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
		unixTimestampStr, err := strconv.ParseInt(string(accountCreationTimestamps[i]), 10, 64)
		require.NoError(t, err)
		unixTimestamp := time.Unix(unixTimestampStr, 0)
		assert.Equal(t, strings.Contains(stringOutput, humanize.Time(unixTimestamp)), true)
	}
}
