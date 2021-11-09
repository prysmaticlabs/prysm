package imported

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/crypto/rand"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
	mock "github.com/prysmaticlabs/prysm/validator/accounts/testing"
	constant "github.com/prysmaticlabs/prysm/validator/testing"
	"github.com/tyler-smith/go-bip39"
)

// We test that using a '25th word' mnemonic passphrase leads to different
// public keys derived than not specifying the passphrase.
func TestImportedKeymanager_Recover_25Words(t *testing.T) {
	ctx := context.Background()
	wallet := &mock.Wallet{
		Files:            make(map[string]map[string][]byte),
		AccountPasswords: make(map[string]string),
		WalletPassword:   password,
	}
	km, err := NewKeymanager(ctx, &SetupConfig{
		Wallet:           wallet,
		ListenForChanges: false,
	})
	require.NoError(t, err)
	numAccounts := 5
	err = km.RecoverAccountsFromMnemonic(ctx, constant.TestMnemonic, "mnemonicpass", numAccounts)
	require.NoError(t, err)
	without25thWord, err := km.FetchValidatingPublicKeys(ctx)
	require.NoError(t, err)
	wallet = &mock.Wallet{
		Files:            make(map[string]map[string][]byte),
		AccountPasswords: make(map[string]string),
		WalletPassword:   password,
	}
	km, err = NewKeymanager(ctx, &SetupConfig{
		Wallet:           wallet,
		ListenForChanges: false,
	})
	require.NoError(t, err)
	// No mnemonic passphrase this time.
	err = km.RecoverAccountsFromMnemonic(ctx, constant.TestMnemonic, "", numAccounts)
	require.NoError(t, err)
	with25thWord, err := km.FetchValidatingPublicKeys(ctx)
	require.NoError(t, err)
	for i, k := range with25thWord {
		without := without25thWord[i]
		assert.DeepNotEqual(t, k, without)
	}
}

func TestImportedKeymanager_Recover_RoundTrip(t *testing.T) {
	mnemonicEntropy := make([]byte, 32)
	n, err := rand.NewGenerator().Read(mnemonicEntropy)
	require.NoError(t, err)
	require.Equal(t, n, len(mnemonicEntropy))
	mnemonic, err := bip39.NewMnemonic(mnemonicEntropy)
	require.NoError(t, err)
	wanted := bip39.NewSeed(mnemonic, "")

	got, err := seedFromMnemonic(mnemonic, "" /* no passphrase */)
	require.NoError(t, err)
	// Ensure the derived seed matches.
	assert.DeepEqual(t, wanted, got)
}
