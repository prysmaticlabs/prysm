package imported

import (
	"github.com/prysmaticlabs/prysm/crypto/bls"
	mock "github.com/prysmaticlabs/prysm/validator/accounts/testing"
)

func TestImportedKeymanager_FetchValidatingPublicKeys(t *testing.T) {
	wallet := &mock.Wallet{
		Files:          make(map[string]map[string][]byte),
		WalletPassword: password,
	}
	dr := &Keymanager{
		wallet:        wallet,
		accountsStore: &accountStore{},
	}
	// First, generate accounts and their keystore.json files.
	ctx := context.Background()
	numAccounts := 10
	wantedPubKeys := make([][48]byte, 0)
	for i := 0; i < numAccounts; i++ {
		privKey, err := bls.RandKey()
		require.NoError(t, err)
		pubKey := bytesutil.ToBytes48(privKey.PublicKey().Marshal())
		wantedPubKeys = append(wantedPubKeys, pubKey)
		dr.accountsStore.PublicKeys = append(dr.accountsStore.PublicKeys, pubKey[:])
		dr.accountsStore.PrivateKeys = append(dr.accountsStore.PrivateKeys, privKey.Marshal())
	}
	require.NoError(t, dr.initializeKeysCachesFromKeystore())
	publicKeys, err := dr.FetchValidatingPublicKeys(ctx)
	require.NoError(t, err)
	assert.Equal(t, numAccounts, len(publicKeys))
	// FetchValidatingPublicKeys is also used in generating the output of account list
	// therefore the results must be in the same order as the order in which the accounts were derived
	for i, key := range wantedPubKeys {
		assert.Equal(t, key, publicKeys[i])
	}
}

func TestImportedKeymanager_FetchValidatingPrivateKeys(t *testing.T) {
	wallet := &mock.Wallet{
		Files:          make(map[string]map[string][]byte),
		WalletPassword: password,
	}
	dr := &Keymanager{
		wallet:        wallet,
		accountsStore: &accountStore{},
	}
	// First, generate accounts and their keystore.json files.
	ctx := context.Background()
	numAccounts := 10
	wantedPrivateKeys := make([][32]byte, numAccounts)
	for i := 0; i < numAccounts; i++ {
		privKey, err := bls.RandKey()
		require.NoError(t, err)
		privKeyData := privKey.Marshal()
		pubKey := bytesutil.ToBytes48(privKey.PublicKey().Marshal())
		wantedPrivateKeys[i] = bytesutil.ToBytes32(privKeyData)
		dr.accountsStore.PublicKeys = append(dr.accountsStore.PublicKeys, pubKey[:])
		dr.accountsStore.PrivateKeys = append(dr.accountsStore.PrivateKeys, privKeyData)
	}
	require.NoError(t, dr.initializeKeysCachesFromKeystore())
	privateKeys, err := dr.FetchValidatingPrivateKeys(ctx)
	require.NoError(t, err)
	assert.Equal(t, numAccounts, len(privateKeys))
	// FetchValidatingPrivateKeys is also used in generating the output of account list
	// therefore the results must be in the same order as the order in which the accounts were created
	for i, key := range wantedPrivateKeys {
		assert.Equal(t, key, privateKeys[i])
	}
}
