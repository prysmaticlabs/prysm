package derived

import (
	"context"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	mock "github.com/prysmaticlabs/prysm/validator/accounts/testing"
)

func TestDerivationFromMnemonic(t *testing.T) {
	secretKeysCache = make(map[[48]byte]bls.SecretKey)
	mnemonic := "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"
	passphrase := "TREZOR"
	seed := "c55257c360c07c72029aebc1b53c05ed0362ada38ead3e3e9efa3708e53495531f09a6987599d18264c1e1c92f2cf141630c7a3c4ab7c81b2f001698e7463b04"
	masterSK := "6083874454709270928345386274498605044986640685124978867557563392430687146096"
	childIndex := 0
	childSK := "20397789859736650942317412262472558107875392172444076792671091975210932703118"
	ctx := context.Background()
	wallet := &mock.Wallet{
		Files:            make(map[string]map[string][]byte),
		AccountPasswords: make(map[string]string),
		WalletPassword:   "secretPassw0rd$1999",
	}
	km, err := KeymanagerForPhrase(ctx, &SetupConfig{
		Opts:             DefaultKeymanagerOpts(),
		Wallet:           wallet,
		Mnemonic:         mnemonic,
		Mnemonic25thWord: passphrase,
	})
	require.NoError(t, err)
	seedBytes, err := hex.DecodeString(seed)
	require.NoError(t, err)
	assert.DeepEqual(t, seedBytes, km.seed)

	// We create an account, then check the master SK and the child SK.
	withdrawalKeyPath := fmt.Sprintf(WithdrawalKeyDerivationPathTemplate, 0)
	validatingKeyPath := fmt.Sprintf(ValidatingKeyDerivationPathTemplate, childIndex)
	withdrawalKey, err := km.deriveKey(withdrawalKeyPath)
	require.NoError(t, err)
	validatingKey, err := km.deriveKey(validatingKeyPath)
	require.NoError(t, err)

	expectedMasterSK, err := bls.SecretKeyFromBigNum(masterSK)
	require.NoError(t, err)
	expectedChildSK, err := bls.SecretKeyFromBigNum(childSK)
	require.NoError(t, err)
	assert.DeepEqual(t, expectedMasterSK.Marshal(), withdrawalKey.Marshal())
	assert.DeepEqual(t, expectedChildSK.Marshal(), validatingKey.Marshal())
	fmt.Printf("Expected SK %#x\n", expectedMasterSK.Marshal())
	fmt.Printf("Got %#x\n", withdrawalKey.Marshal())
	fmt.Printf("Expected child SK %#x\n", expectedChildSK.Marshal())
	fmt.Printf("Got %#x\n", validatingKey.Marshal())
}
