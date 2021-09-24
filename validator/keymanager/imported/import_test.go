package imported

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/prysmaticlabs/prysm/crypto/bls"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
	mock "github.com/prysmaticlabs/prysm/validator/accounts/testing"
	"github.com/prysmaticlabs/prysm/validator/keymanager"
	logTest "github.com/sirupsen/logrus/hooks/test"
	keystorev4 "github.com/wealdtech/go-eth2-wallet-encryptor-keystorev4"
)

const password = "secretPassw0rd$1999"

func createRandomKeystore(t testing.TB, password string) *keymanager.Keystore {
	encryptor := keystorev4.New()
	id, err := uuid.NewRandom()
	require.NoError(t, err)
	validatingKey, err := bls.RandKey()
	require.NoError(t, err)
	pubKey := validatingKey.PublicKey().Marshal()
	cryptoFields, err := encryptor.Encrypt(validatingKey.Marshal(), password)
	require.NoError(t, err)
	return &keymanager.Keystore{
		Crypto:  cryptoFields,
		Pubkey:  fmt.Sprintf("%x", pubKey),
		ID:      id.String(),
		Version: encryptor.Version(),
		Name:    encryptor.Name(),
	}
}

func TestImportedKeymanager_CreateAccountsKeystore_NoDuplicates(t *testing.T) {
	numKeys := 50
	pubKeys := make([][]byte, numKeys)
	privKeys := make([][]byte, numKeys)
	for i := 0; i < numKeys; i++ {
		priv, err := bls.RandKey()
		require.NoError(t, err)
		privKeys[i] = priv.Marshal()
		pubKeys[i] = priv.PublicKey().Marshal()
	}
	wallet := &mock.Wallet{
		WalletPassword: "Passwordz0202$",
	}
	dr := &Keymanager{
		wallet: wallet,
	}
	ctx := context.Background()
	_, err := dr.CreateAccountsKeystore(ctx, privKeys, pubKeys)
	require.NoError(t, err)

	// We expect the 50 keys in the account store to match.
	require.NotNil(t, dr.accountsStore)
	require.Equal(t, len(dr.accountsStore.PublicKeys), len(dr.accountsStore.PrivateKeys))
	require.Equal(t, len(dr.accountsStore.PublicKeys), numKeys)
	for i := 0; i < len(dr.accountsStore.PrivateKeys); i++ {
		assert.DeepEqual(t, dr.accountsStore.PrivateKeys[i], privKeys[i])
		assert.DeepEqual(t, dr.accountsStore.PublicKeys[i], pubKeys[i])
	}

	// Re-run the create accounts keystore function with the same pubkeys.
	_, err = dr.CreateAccountsKeystore(ctx, privKeys, pubKeys)
	require.NoError(t, err)

	// We expect nothing to change.
	require.NotNil(t, dr.accountsStore)
	require.Equal(t, len(dr.accountsStore.PublicKeys), len(dr.accountsStore.PrivateKeys))
	require.Equal(t, len(dr.accountsStore.PublicKeys), numKeys)
	for i := 0; i < len(dr.accountsStore.PrivateKeys); i++ {
		assert.DeepEqual(t, dr.accountsStore.PrivateKeys[i], privKeys[i])
		assert.DeepEqual(t, dr.accountsStore.PublicKeys[i], pubKeys[i])
	}

	// Now, we run the function again but with a new priv and pubkey and this
	// time, we do expect a change.
	privKey, err := bls.RandKey()
	require.NoError(t, err)
	privKeys = append(privKeys, privKey.Marshal())
	pubKeys = append(pubKeys, privKey.PublicKey().Marshal())

	_, err = dr.CreateAccountsKeystore(ctx, privKeys, pubKeys)
	require.NoError(t, err)
	require.Equal(t, len(dr.accountsStore.PublicKeys), len(dr.accountsStore.PrivateKeys))

	// We should have 1 more new key in the store.
	require.Equal(t, numKeys+1, len(dr.accountsStore.PrivateKeys))
}

func TestImportedKeymanager_ImportKeystores(t *testing.T) {
	// Setup the keymanager.
	wallet := &mock.Wallet{
		Files:          make(map[string]map[string][]byte),
		WalletPassword: password,
	}
	dr := &Keymanager{
		wallet:        wallet,
		accountsStore: &accountStore{},
	}

	// Create a duplicate keystore and attempt to import it. This should complete correctly though log specific output.
	numAccounts := 5
	keystores := make([]*keymanager.Keystore, numAccounts+1)
	for i := 1; i < numAccounts+1; i++ {
		keystores[i] = createRandomKeystore(t, password)
	}
	keystores[0] = keystores[1]
	ctx := context.Background()
	hook := logTest.NewGlobal()
	require.NoError(t, dr.ImportKeystores(
		ctx,
		keystores,
		password,
	))
	require.LogsContain(t, hook, "Duplicate key")
	// Import them correctly even without the duplicate.
	require.NoError(t, dr.ImportKeystores(
		ctx,
		keystores[1:],
		password,
	))

	// Ensure the single, all-encompassing accounts keystore was written
	// to the wallet and ensure we can decrypt it using the EIP-2335 standard.
	var encodedKeystore []byte
	for k, v := range wallet.Files[AccountsPath] {
		if strings.Contains(k, "keystore") {
			encodedKeystore = v
		}
	}
	require.NotNil(t, encodedKeystore, "could not find keystore file")
	keystoreFile := &keymanager.Keystore{}
	require.NoError(t, json.Unmarshal(encodedKeystore, keystoreFile))

	// We decrypt the crypto fields of the accounts keystore.
	decryptor := keystorev4.New()
	encodedAccounts, err := decryptor.Decrypt(keystoreFile.Crypto, password)
	require.NoError(t, err, "Could not decrypt validator accounts")
	store := &accountStore{}
	require.NoError(t, json.Unmarshal(encodedAccounts, store))

	// We should have successfully imported all accounts
	// from external sources into a single AccountsStore
	// struct preserved within a single keystore file.
	assert.Equal(t, numAccounts, len(store.PublicKeys))
	assert.Equal(t, numAccounts, len(store.PrivateKeys))
}
