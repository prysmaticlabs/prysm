package v1_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v1"
	keystorev4 "github.com/wealdtech/go-eth2-wallet-encryptor-keystorev4"
	nd "github.com/wealdtech/go-eth2-wallet-nd/v2"
	filesystem "github.com/wealdtech/go-eth2-wallet-store-filesystem"
	e2wtypes "github.com/wealdtech/go-eth2-wallet-types/v2"
)

func SetupWallet(t *testing.T) string {
	path, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	store := filesystem.New(filesystem.WithLocation(path))
	encryptor := keystorev4.New()

	// Create wallets with keys.
	ctx := context.Background()
	w1, err := nd.CreateWallet(ctx, "Wallet 1", store, encryptor)
	creator, ok := w1.(e2wtypes.WalletAccountCreator)
	require.Equal(t, true, ok)
	require.NoError(t, err, "Failed to create wallet")
	require.NoError(t, err, "Failed to unlock wallet")
	_, err = creator.CreateAccount(ctx, "Account 1", []byte("foo"))
	require.NoError(t, err, "Failed to create account 1")
	_, err = creator.CreateAccount(ctx, "Account 2", []byte("bar"))
	require.NoError(t, err, "Failed to create account 2")

	return path
}

func wallet(t *testing.T, opts string) keymanager.KeyManager {
	km, _, err := keymanager.NewWallet(opts)
	require.NoError(t, err)
	return km
}

func TestMultiplePassphrases(t *testing.T) {
	path := SetupWallet(t)
	defer func() {
		if err := os.RemoveAll(path); err != nil {
			t.Log(err)
		}
	}()
	tests := []struct {
		name     string
		wallet   keymanager.KeyManager
		accounts int
	}{
		{
			name:     "Neither",
			wallet:   wallet(t, fmt.Sprintf(`{"location":%q,"accounts":["Wallet 1"],"passphrases":["neither"]}`, path)),
			accounts: 0,
		},
		{
			name:     "Foo",
			wallet:   wallet(t, fmt.Sprintf(`{"location":%q,"accounts":["Wallet 1"],"passphrases":["foo"]}`, path)),
			accounts: 1,
		},
		{
			name:     "Bar",
			wallet:   wallet(t, fmt.Sprintf(`{"location":%q,"accounts":["Wallet 1"],"passphrases":["bar"]}`, path)),
			accounts: 1,
		},
		{
			name:     "Both",
			wallet:   wallet(t, fmt.Sprintf(`{"location":%q,"accounts":["Wallet 1"],"passphrases":["foo","bar"]}`, path)),
			accounts: 2,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			keys, err := test.wallet.FetchValidatingKeys()
			assert.NoError(t, err)
			assert.Equal(t, test.accounts, len(keys), "Found %d keys", len(keys))
		})
	}
}

func TestEnvPassphrases(t *testing.T) {
	path := SetupWallet(t)
	defer func() {
		if err := os.RemoveAll(path); err != nil {
			t.Log(err)
		}
	}()

	err := os.Setenv("TESTENVPASSPHRASES_NEITHER", "neither")
	require.NoError(t, err, "Error setting environment variable TESTENVPASSPHRASES_NEITHER")
	defer func() {
		err := os.Unsetenv("TESTENVPASSPHRASES_NEITHER")
		require.NoError(t, err, "Error unsetting environment variable TESTENVPASSPHRASES_NEITHER")
	}()
	err = os.Setenv("TESTENVPASSPHRASES_FOO", "foo")
	require.NoError(t, err, "Error setting environment variable TESTENVPASSPHRASES_FOO")
	defer func() {
		err := os.Unsetenv("TESTENVPASSPHRASES_FOO")
		require.NoError(t, err, "Error unsetting environment variable TESTENVPASSPHRASES_FOO")
	}()
	err = os.Setenv("TESTENVPASSPHRASES_BAR", "bar")
	require.NoError(t, err, "Error setting environment variable TESTENVPASSPHRASES_BAR")
	defer func() {
		err := os.Unsetenv("TESTENVPASSPHRASES_BAR")
		require.NoError(t, err, "Error unsetting environment variable TESTENVPASSPHRASES_BAR")
	}()

	tests := []struct {
		name     string
		wallet   keymanager.KeyManager
		accounts int
	}{
		{
			name:     "Neither",
			wallet:   wallet(t, fmt.Sprintf(`{"location":%q,"accounts":["Wallet 1"],"passphrases":["$TESTENVPASSPHRASES_NEITHER"]}`, path)),
			accounts: 0,
		},
		{
			name:     "Foo",
			wallet:   wallet(t, fmt.Sprintf(`{"location":%q,"accounts":["Wallet 1"],"passphrases":["$TESTENVPASSPHRASES_FOO"]}`, path)),
			accounts: 1,
		},
		{
			name:     "Bar",
			wallet:   wallet(t, fmt.Sprintf(`{"location":%q,"accounts":["Wallet 1"],"passphrases":["$TESTENVPASSPHRASES_BAR"]}`, path)),
			accounts: 1,
		},
		{
			name:     "Both",
			wallet:   wallet(t, fmt.Sprintf(`{"location":%q,"accounts":["Wallet 1"],"passphrases":["$TESTENVPASSPHRASES_FOO","$TESTENVPASSPHRASES_BAR"]}`, path)),
			accounts: 2,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			keys, err := test.wallet.FetchValidatingKeys()
			assert.NoError(t, err)
			assert.Equal(t, test.accounts, len(keys), "Found %d keys", len(keys))
		})
	}
}
