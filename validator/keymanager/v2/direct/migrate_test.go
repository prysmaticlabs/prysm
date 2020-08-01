package direct

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/petnames"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	mock "github.com/prysmaticlabs/prysm/validator/accounts/v2/testing"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	logTest "github.com/sirupsen/logrus/hooks/test"
	keystorev4 "github.com/wealdtech/go-eth2-wallet-encryptor-keystorev4"
)

func TestDirectKeymanager_MigrateToSingleKeystoreFormat(t *testing.T) {
	hook := logTest.NewGlobal()
	password := "secretPassw0rd$1999"
	wallet := &mock.Wallet{
		Files:            make(map[string]map[string][]byte),
		AccountPasswords: make(map[string]string),
		WalletPassword:   password,
	}
	dr := &Keymanager{
		keysCache:     make(map[[48]byte]bls.SecretKey),
		wallet:        wallet,
		accountsStore: &AccountStore{},
	}
	ctx := context.Background()

	// Generate several old account keystore and save to the wallet path.
	numAccounts := 5
	wallet.Directories = make([]string, numAccounts)
	wantedValidatingKeys := make([]bls.SecretKey, numAccounts)
	for i := 0; i < numAccounts; i++ {
		validatingKey := bls.RandKey()
		accountName, keystore := generateOldAccountKeystore(t, dr, validatingKey, password)
		wallet.Directories[i] = accountName
		wantedValidatingKeys[i] = validatingKey
		encodedKeystore, err := json.MarshalIndent(keystore, "", "\t")
		require.NoError(t, err)
		require.NoError(t, dr.wallet.WritePasswordToDisk(ctx, accountName+".pass", password))
		require.NoError(t, dr.wallet.WriteFileAtPath(ctx, accountName, KeystoreFileName, encodedKeystore))
	}

	// Now, we run the migration strategy.
	require.NoError(t, dr.migrateToSingleKeystore(ctx))

	// We retrieve the new accounts keystore format containing all keys in a single file.
	var encodedAccountsFile []byte
	for k, v := range wallet.Files[accountsPath] {
		if strings.Contains(k, "keystore") {
			encodedAccountsFile = v
		}
	}
	require.NotNil(t, encodedAccountsFile, "could not find keystore file")
	accountsKeystore := &v2keymanager.Keystore{}
	require.NoError(t, json.Unmarshal(encodedAccountsFile, accountsKeystore))

	// We extract the accounts from the keystore.
	decryptor := keystorev4.New()
	encodedAccounts, err := decryptor.Decrypt(accountsKeystore.Crypto, password)
	require.NoError(t, err, "Could not decrypt validator accounts")
	store := &AccountStore{}
	require.NoError(t, json.Unmarshal(encodedAccounts, store))

	// We expect the migration strategy to have succeeded, with all accounts from earlier
	// now being stored in a single keystore file.
	require.Equal(t, numAccounts, len(store.PublicKeys))
	require.Equal(t, numAccounts, len(store.PrivateKeys))
	for i := 0; i < numAccounts; i++ {
		privKey, err := bls.SecretKeyFromBytes(store.PrivateKeys[i])
		require.NoError(t, err)
		assert.DeepEqual(t, privKey.Marshal(), wantedValidatingKeys[i].Marshal())
	}
	testutil.AssertLogsContain(t, hook, "Now migrating accounts to a more efficient format")
}

func generateOldAccountKeystore(
	t testing.TB, dr *Keymanager, validatingKey bls.SecretKey, password string,
) (string, *v2keymanager.Keystore) {
	accountName := petnames.DeterministicName(validatingKey.PublicKey().Marshal(), "-")
	// Generates a new EIP-2335 compliant keystore file
	// from a BLS private key and marshals it as JSON.
	encryptor := keystorev4.New()
	cryptoFields, err := encryptor.Encrypt(validatingKey.Marshal(), password)
	require.NoError(t, err)
	id, err := uuid.NewRandom()
	require.NoError(t, err)
	return accountName, &v2keymanager.Keystore{
		Crypto:  cryptoFields,
		ID:      id.String(),
		Pubkey:  fmt.Sprintf("%x", validatingKey.PublicKey().Marshal()),
		Version: encryptor.Version(),
		Name:    encryptor.Name(),
	}
}
