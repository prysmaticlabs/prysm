package direct

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/depositutil"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	mock "github.com/prysmaticlabs/prysm/validator/accounts/v2/testing"

	logTest "github.com/sirupsen/logrus/hooks/test"
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

func TestKeymanager_CreateAccount(t *testing.T) {
	hook := logTest.NewGlobal()
	wallet := &mock.MockWallet{
		Files:            make(map[string][]byte),
		AccountPasswords: make(map[string]string),
	}
	mnemonicGenerator := &mockMnemonicGenerator{
		generatedMnemonics: make([]string, 0),
	}
	dr := &Keymanager{
		wallet:            wallet,
		mnemonicGenerator: mnemonicGenerator,
	}
	ctx := context.Background()
	password := "secretPassw0rd$1999"
	if err := dr.CreateAccount(ctx, password); err != nil {
		t.Fatal(err)
	}

	// Ensure the keystore file was written to the wallet
	// and ensure we can decrypt it using the EIP-2335 standard.
	encodedKeystore, ok := wallet.Files[keystoreFileName]
	if !ok {
		t.Fatalf("Expected to have stored %s in wallet", keystoreFileName)
	}
	keystoreJSON := make(map[string]interface{})
	if err := json.Unmarshal(encodedKeystore, &keystoreJSON); err != nil {
		t.Fatalf("Could not decode keystore json: %v", err)
	}

	// We extract the validator signing private key from the keystore
	// by utilizing the password and initialize a new BLS secret key from
	// its raw bytes.
	decryptor := keystorev4.New()
	rawSigningKey, err := decryptor.Decrypt(keystoreJSON, []byte(password))
	if err != nil {
		t.Fatalf("Could not decrypt validator signing key: %v", err)
	}
	validatorSigningKey, err := bls.SecretKeyFromBytes(rawSigningKey)
	if err != nil {
		t.Fatalf("Could not instantiate bls secret key from bytes: %v", err)
	}

	// Decode the deposit_data.ssz file and confirm
	// the public key matches the public key from the
	// account's decrypted keystore.
	encodedDepositData, ok := wallet.Files[depositDataFileName]
	if !ok {
		t.Fatalf("Expected to have stored %s in wallet", depositDataFileName)
	}
	depositData := &ethpb.Deposit_Data{}
	if err := ssz.Unmarshal(encodedDepositData, depositData); err != nil {
		t.Fatal(err)
	}

	depositPublicKey := depositData.PublicKey
	publicKey := validatorSigningKey.PublicKey().Marshal()
	if !bytes.Equal(depositPublicKey, publicKey) {
		t.Errorf(
			"Expected deposit data public key %#x to match public key from keystore %#x",
			depositPublicKey,
			publicKey,
		)
	}

	// We ensure the mnemonic phrase has successfully been generated.
	if len(mnemonicGenerator.generatedMnemonics) != 1 {
		t.Fatal("Expected to have generated new mnemonic for private key")
	}
	mnemonicPhrase := mnemonicGenerator.generatedMnemonics[0]
	rawWithdrawalBytes, err := bip39.EntropyFromMnemonic(mnemonicPhrase)
	if err != nil {
		t.Fatal(err)
	}
	validatorWithdrawalKey, err := bls.SecretKeyFromBytes(rawWithdrawalBytes)
	if err != nil {
		t.Fatalf("Could not instantiate bls secret key from bytes: %v", err)
	}

	// We then verify the withdrawal hash created from the recovered withdrawal key
	// given the mnemonic phrase does indeed verify with the deposit data that was persisted on disk.
	withdrawalHash := depositutil.WithdrawalCredentialsHash(validatorWithdrawalKey)
	if !bytes.Equal(withdrawalHash, depositData.WithdrawalCredentials) {
		t.Errorf(
			"Expected matching withdrawal credentials, got %#x, received %#x",
			withdrawalHash,
			depositData.WithdrawalCredentials,
		)
	}
	testutil.AssertLogsContain(t, hook, "Successfully created new validator account")
}

func TestKeymanager_FetchValidatingPublicKeys(t *testing.T) {

}
