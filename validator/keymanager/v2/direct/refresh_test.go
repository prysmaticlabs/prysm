package direct

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	mock "github.com/prysmaticlabs/prysm/validator/accounts/v2/testing"
)

func TestDirectKeymanager_reloadAccountsFromKeystore(t *testing.T) {
	password := "Passw03rdz293**%#2"
	wallet := &mock.Wallet{
		Files:            make(map[string]map[string][]byte),
		AccountPasswords: make(map[string]string),
	}
	dr := &Keymanager{
		wallet:              wallet,
		keysCache:           make(map[[48]byte]bls.SecretKey),
		accountsPassword:    password,
		accountsChangedFeed: new(event.Feed),
	}

	numAccounts := 20
	privKeys := make([][]byte, numAccounts)
	pubKeys := make([][]byte, numAccounts)
	for i := 0; i < numAccounts; i++ {
		privKey := bls.RandKey()
		privKeys[i] = privKey.Marshal()
		pubKeys[i] = privKey.PublicKey().Marshal()
	}

	accountsStore, err := dr.createAccountsKeystore(context.Background(), privKeys, pubKeys)
	require.NoError(t, err)
	require.NoError(t, dr.reloadAccountsFromKeystore(accountsStore))

	// Check the key was added to the keys cache.
	for _, keyBytes := range pubKeys {
		_, ok := dr.keysCache[bytesutil.ToBytes48(keyBytes)]
		require.Equal(t, true, ok)
	}

	// Check the key was added to the global accounts store.
	require.Equal(t, numAccounts, len(dr.accountsStore.PublicKeys))
	require.Equal(t, numAccounts, len(dr.accountsStore.PrivateKeys))
	assert.DeepEqual(t, dr.accountsStore.PublicKeys[0], pubKeys[0])
}
