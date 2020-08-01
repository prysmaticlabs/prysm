package direct

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/bls"
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
		Files:          make(map[string]map[string][]byte),
		WalletPassword: password,
	}
	dr := &Keymanager{
		keysCache:     make(map[[48]byte]bls.SecretKey),
		wallet:        wallet,
		accountsStore: &AccountStore{},
	}
	ctx := context.Background()
	accountName, err := dr.CreateAccount(ctx, password)
	require.NoError(t, err)

	// Ensure the keystore file was written to the wallet
	// and ensure we can decrypt it using the EIP-2335 standard.
	var encodedKeystore []byte
	for k, v := range wallet.Files[accountsPath] {
		if strings.Contains(k, "keystore") {
			encodedKeystore = v
		}
	}
	require.NotNil(t, encodedKeystore, "could not find keystore file")
	keystoreFile := &v2keymanager.Keystore{}
	require.NoError(t, json.Unmarshal(encodedKeystore, keystoreFile))

	// We extract the accounts from the keystore.
	decryptor := keystorev4.New()
	encodedAccounts, err := decryptor.Decrypt(keystoreFile.Crypto, password)
	require.NoError(t, err, "Could not decrypt validator accounts")
	store := &AccountStore{}
	require.NoError(t, json.Unmarshal(encodedAccounts, store))

	require.Equal(t, 1, len(store.PublicKeys))
	require.Equal(t, 1, len(store.PrivateKeys))
	privKey, err := bls.SecretKeyFromBytes(store.PrivateKeys[0])
	require.NoError(t, err)
	pubKey := privKey.PublicKey().Marshal()
	assert.DeepEqual(t, pubKey, store.PublicKeys[0])
	testutil.AssertLogsContain(t, hook, accountName)
	testutil.AssertLogsContain(t, hook, "Successfully created new validator account")
}
