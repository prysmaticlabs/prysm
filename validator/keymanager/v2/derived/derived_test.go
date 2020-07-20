package derived

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	mock "github.com/prysmaticlabs/prysm/validator/accounts/v2/testing"
	"github.com/tyler-smith/go-bip39"
	keystorev4 "github.com/wealdtech/go-eth2-wallet-encryptor-keystorev4"
)

type mockMnemonicGenerator struct {
	generatedMnemonics []string
}

func (m *mockMnemonicGenerator) Generate(data []byte) (string, error) {
	newMnemonic, err := bip39.NewMnemonic(data)
	if err != nil {
		return "", err
	}
	m.generatedMnemonics = append(m.generatedMnemonics, newMnemonic)
	return newMnemonic, nil
}

func (m *mockMnemonicGenerator) ConfirmAcknowledgement(phrase string) error {
	return nil
}

func TestDerivedKeymanager_CreateAccount(t *testing.T) {
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
	keystoreFile := &Keystore{}
	require.NoError(t, json.Unmarshal(encodedKeystore, keystoreFile))

	// We extract the validator signing private key from the keystore
	// by utilizing the password and initialize a new BLS secret key from
	// its raw bytes.
	decryptor := keystorev4.New()
	rawValidatingKey, err := decryptor.Decrypt(keystoreFile.Crypto, []byte(password))
	require.NoError(t, err, "Could not decrypt validator signing key")

	validatingKey, err := bls.SecretKeyFromBytes(rawValidatingKey)
	require.NoError(t, err, "Could not instantiate bls secret key from bytes")
	_ = validatingKey

	// Ensure the keystore file was written to the wallet
	// and ensure we can decrypt it using the EIP-2335 standard.
	withdrawalAccount0 := fmt.Sprintf(WithdrawalKeyDerivationPathTemplate, 0)
	encodedKeystore, ok = wallet.Files[withdrawalAccount0][KeystoreFileName]
	require.Equal(t, ok, true, fmt.Sprintf("Expected to have stored %s in wallet", KeystoreFileName))
	keystoreFile = &Keystore{}
	require.NoError(t, json.Unmarshal(encodedKeystore, keystoreFile))

	// We extract the validator signing private key from the keystore
	// by utilizing the password and initialize a new BLS secret key from
	// its raw bytes.
	rawWithdrawalKey, err := decryptor.Decrypt(keystoreFile.Crypto, []byte(password))
	require.NoError(t, err, "Could not decrypt validator withdrawal key")

	withdrawalKey, err := bls.SecretKeyFromBytes(rawWithdrawalKey)
	require.NoError(t, err, "Could not instantiate bls secret key from bytes")
	_ = withdrawalKey

	//// We ensure the mnemonic phrase has successfully been generated.
	//if len(mnemonicGenerator.generatedMnemonics) != 1 {
	//	t.Fatal("Expected to have generated new mnemonic for private key")
	//}
	//mnemonicPhrase := mnemonicGenerator.generatedMnemonics[0]
	//rawWithdrawalBytes, err := bip39.EntropyFromMnemonic(mnemonicPhrase)
	//require.NoError(t, err)
	//withdrawalKey, err := bls.SecretKeyFromBytes(rawWithdrawalBytes)
	//require.NoError(t, err, "Could not instantiate bls secret key from bytes")
	//_ = withdrawalKey
}
