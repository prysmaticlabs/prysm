package imported

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/testing/require"
	mock "github.com/prysmaticlabs/prysm/validator/accounts/testing"
	"github.com/prysmaticlabs/prysm/validator/keymanager"
	logTest "github.com/sirupsen/logrus/hooks/test"
	keystorev4 "github.com/wealdtech/go-eth2-wallet-encryptor-keystorev4"
)

func TestImportedKeymanager_RemoveAccounts(t *testing.T) {
	hook := logTest.NewGlobal()
	wallet := &mock.Wallet{
		Files:          make(map[string]map[string][]byte),
		WalletPassword: password,
	}
	dr := &Keymanager{
		wallet:        wallet,
		accountsStore: &accountStore{},
	}
	numAccounts := 5
	ctx := context.Background()
	keystores := make([]*keymanager.Keystore, numAccounts)
	for i := 0; i < numAccounts; i++ {
		keystores[i] = createRandomKeystore(t, password)
	}
	require.NoError(t, dr.ImportKeystores(ctx, keystores, password))
	accounts, err := dr.FetchValidatingPublicKeys(ctx)
	require.NoError(t, err)
	require.Equal(t, numAccounts, len(accounts))

	accountToRemove := uint64(2)
	accountPubKey := accounts[accountToRemove]
	// Remove an account from the keystore.
	require.NoError(t, dr.DeleteAccounts(ctx, [][]byte{accountPubKey[:]}))
	// Ensure the keystore file was written to the wallet
	// and ensure we can decrypt it using the EIP-2335 standard.
	var encodedKeystore []byte
	for k, v := range wallet.Files[AccountsPath] {
		if strings.Contains(k, "keystore") {
			encodedKeystore = v
		}
	}
	require.NotNil(t, encodedKeystore, "could not find keystore file")
	keystoreFile := &keymanager.Keystore{}
	require.NoError(t, json.Unmarshal(encodedKeystore, keystoreFile))

	// We extract the accounts from the keystore.
	decryptor := keystorev4.New()
	encodedAccounts, err := decryptor.Decrypt(keystoreFile.Crypto, password)
	require.NoError(t, err, "Could not decrypt validator accounts")
	store := &accountStore{}
	require.NoError(t, json.Unmarshal(encodedAccounts, store))

	require.Equal(t, numAccounts-1, len(store.PublicKeys))
	require.Equal(t, numAccounts-1, len(store.PrivateKeys))
	require.LogsContain(t, hook, fmt.Sprintf("%#x", bytesutil.Trunc(accountPubKey[:])))
	require.LogsContain(t, hook, "Successfully deleted validator account")
}
