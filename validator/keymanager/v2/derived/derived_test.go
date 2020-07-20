package derived

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
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

func TestDerivedKeymanager_CreateAccount(t *testing.T) {
	hook := logTest.NewGlobal()
	wallet := &mock.Wallet{
		Files:            make(map[string]map[string][]byte),
		AccountPasswords: make(map[string]string),
	}
	seed := make([]byte, 32)
	copy(seed, "hello world")
	dr := &Keymanager{
		wallet: wallet,
		seed:   seed,
		seedCfg: &SeedConfig{
			NextAccount: 0,
		},
	}
	ctx := context.Background()
	password := "secretPassw0rd$1999"
	accountName, err := dr.CreateAccount(ctx, password)
	require.NoError(t, err)
	assert.Equal(t, "0", accountName)

	// Ensure the keystore file was written to the wallet
	// and ensure we can decrypt it using the EIP-2335 standard.
	validatingAccount0 := fmt.Sprintf(ValidatingKeyDerivationPathTemplate, 0)
	encodedKeystore, ok := wallet.Files[validatingAccount0][KeystoreFileName]
	require.Equal(t, ok, true, fmt.Sprintf("Expected to have stored %s in wallet", KeystoreFileName))
	keystoreFile := &v2keymanager.Keystore{}
	require.NoError(t, json.Unmarshal(encodedKeystore, keystoreFile))

	// We extract the validator signing private key from the keystore
	// by utilizing the password and initialize a new BLS secret key from
	// its raw bytes.
	decryptor := keystorev4.New()
	rawValidatingKey, err := decryptor.Decrypt(keystoreFile.Crypto, []byte(password))
	require.NoError(t, err, "Could not decrypt validator signing key")

	validatingKey, err := bls.SecretKeyFromBytes(rawValidatingKey)
	require.NoError(t, err, "Could not instantiate bls secret key from bytes")

	// Ensure the keystore file was written to the wallet
	// and ensure we can decrypt it using the EIP-2335 standard.
	withdrawalAccount0 := fmt.Sprintf(WithdrawalKeyDerivationPathTemplate, 0)
	encodedKeystore, ok = wallet.Files[withdrawalAccount0][KeystoreFileName]
	require.Equal(t, ok, true, fmt.Sprintf("Expected to have stored %s in wallet", KeystoreFileName))
	keystoreFile = &v2keymanager.Keystore{}
	require.NoError(t, json.Unmarshal(encodedKeystore, keystoreFile))

	// We extract the validator signing private key from the keystore
	// by utilizing the password and initialize a new BLS secret key from
	// its raw bytes.
	rawWithdrawalKey, err := decryptor.Decrypt(keystoreFile.Crypto, []byte(password))
	require.NoError(t, err, "Could not decrypt validator withdrawal key")

	withdrawalKey, err := bls.SecretKeyFromBytes(rawWithdrawalKey)
	require.NoError(t, err, "Could not instantiate bls secret key from bytes")

	// Assert the new value for next account increased and also
	// check the config file was updated on disk with this new value.
	assert.Equal(t, uint64(1), dr.seedCfg.NextAccount, "Wrong value for next account")
	encryptedSeedFile, err := wallet.ReadEncryptedSeedFromDisk(ctx)
	require.NoError(t, err)
	enc, err := ioutil.ReadAll(encryptedSeedFile)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, encryptedSeedFile.Close())
	}()
	seedConfig := &SeedConfig{}
	require.NoError(t, json.Unmarshal(enc, seedConfig))
	assert.Equal(t, uint64(1), seedConfig.NextAccount, "Wrong value for next account")

	// Ensure the new account information is displayed to stdout.
	testutil.AssertLogsContain(t, hook, "Successfully created new validator account")
	testutil.AssertLogsContain(t, hook, fmt.Sprintf("%#x", validatingKey.PublicKey().Marshal()))
	testutil.AssertLogsContain(t, hook, fmt.Sprintf("%#x", withdrawalKey.PublicKey().Marshal()))
}
