package direct

import (
	"bytes"
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	validatorpb "github.com/prysmaticlabs/prysm/proto/validator/accounts/v2"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
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
	wallet := &mock.Wallet{
		Files:            make(map[string]map[string][]byte),
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
	accountName, err := dr.CreateAccount(ctx, password)
	if err != nil {
		t.Fatal(err)
	}

	// Ensure the keystore file was written to the wallet
	// and ensure we can decrypt it using the EIP-2335 standard.
	encodedKeystore, ok := wallet.Files[accountName][keystoreFileName]
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
	encodedDepositData, ok := wallet.Files[accountName][depositDataFileName]
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
	wallet := &mock.Wallet{
		Files:            make(map[string]map[string][]byte),
		AccountPasswords: make(map[string]string),
	}
	dr := &Keymanager{
		wallet:    wallet,
		keysCache: make(map[[48]byte]bls.SecretKey),
	}
	// First, generate accounts and their keystore.json files.
	ctx := context.Background()
	numAccounts := 20
	wantedPublicKeys := generateAccounts(t, numAccounts, dr)
	publicKeys, err := dr.FetchValidatingPublicKeys(ctx)
	if err != nil {
		t.Fatal(err)
	}
	// The results are not guaranteed to be ordered, so we ensure each
	// key we expect exists in the results via a map.
	keysMap := make(map[[48]byte]bool)
	for _, key := range publicKeys {
		keysMap[key] = true
	}
	for _, wanted := range wantedPublicKeys {
		if _, ok := keysMap[wanted]; !ok {
			t.Errorf("Could not find expected public key %#x in results", wanted)
		}
	}
}

func TestKeymanager_Sign(t *testing.T) {
	wallet := &mock.Wallet{
		Files:            make(map[string]map[string][]byte),
		AccountPasswords: make(map[string]string),
	}
	dr := &Keymanager{
		wallet:    wallet,
		keysCache: make(map[[48]byte]bls.SecretKey),
	}

	// First, generate accounts and their keystore.json files.
	numAccounts := 2
	generateAccounts(t, numAccounts, dr)
	ctx := context.Background()
	publicKeys, err := dr.FetchValidatingPublicKeys(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// We prepare naive data to sign.
	data := []byte("hello world")
	signRequest := &validatorpb.SignRequest{
		PublicKey: publicKeys[0][:],
		Data:      data,
	}
	sig, err := dr.Sign(ctx, signRequest)
	if err != nil {
		t.Fatal(err)
	}
	pubKey, err := bls.PublicKeyFromBytes(publicKeys[0][:])
	if err != nil {
		t.Fatal(err)
	}
	wrongPubKey, err := bls.PublicKeyFromBytes(publicKeys[1][:])
	if err != nil {
		t.Fatal(err)
	}
	if !sig.Verify(pubKey, data) {
		t.Fatalf("Expected sig to verify for pubkey %#x and data %v", pubKey.Marshal(), data)
	}
	if sig.Verify(wrongPubKey, data) {
		t.Fatalf("Expected sig not to verify for pubkey %#x and data %v", wrongPubKey.Marshal(), data)
	}
}

func TestKeymanager_Sign_WrongRequestType(t *testing.T) {
	type badSignReq struct{}
	dr := &Keymanager{}
	_, err := dr.Sign(context.Background(), &badSignReq{})
	if err == nil {
		t.Error("Expected error, received nil")
	}
	if !strings.Contains(err.Error(), "received wrong type") {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestKeymanager_Sign_NoPublicKeySpecified(t *testing.T) {
	req := &validatorpb.SignRequest{
		PublicKey: nil,
	}
	dr := &Keymanager{}
	_, err := dr.Sign(context.Background(), req)
	if err == nil {
		t.Error("Expected error, received nil")
	}
	if !strings.Contains(err.Error(), "nil public key") {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestKeymanager_Sign_NoPublicKeyInCache(t *testing.T) {
	req := &validatorpb.SignRequest{
		PublicKey: []byte("hello world"),
	}
	dr := &Keymanager{
		keysCache: make(map[[48]byte]bls.SecretKey),
	}
	_, err := dr.Sign(context.Background(), req)
	if err == nil {
		t.Error("Expected error, received nil")
	}
	if !strings.Contains(err.Error(), "no signing key found in keys cache") {
		t.Errorf("Unexpected error: %v", err)
	}
}

func BenchmarkKeymanager_FetchValidatingPublicKeys(b *testing.B) {
	b.StopTimer()
	wallet := &mock.Wallet{
		Files:            make(map[string]map[string][]byte),
		AccountPasswords: make(map[string]string),
	}
	dr := &Keymanager{
		wallet:    wallet,
		keysCache: make(map[[48]byte]bls.SecretKey),
	}
	// First, generate accounts and their keystore.json files.
	numAccounts := 1000
	generateAccounts(b, numAccounts, dr)
	ctx := context.Background()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if _, err := dr.FetchValidatingPublicKeys(ctx); err != nil {
			b.Fatal(err)
		}
	}
}

func generateAccounts(t testing.TB, numAccounts int, dr *Keymanager) [][48]byte {
	ctx := context.Background()
	wantedPublicKeys := make([][48]byte, numAccounts)
	for i := 0; i < numAccounts; i++ {
		encryptor := keystorev4.New()
		validatingKey := bls.RandKey()
		wantedPublicKeys[i] = bytesutil.ToBytes48(validatingKey.PublicKey().Marshal())
		password := strconv.Itoa(i)
		keystoreFile, err := encryptor.Encrypt(validatingKey.Marshal(), []byte(password))
		if err != nil {
			t.Fatal(err)
		}
		encoded, err := json.MarshalIndent(keystoreFile, "", "\t")
		if err != nil {
			t.Fatal(err)
		}
		accountName, err := dr.wallet.WriteAccountToDisk(ctx, password)
		if err != nil {
			t.Fatal(err)
		}
		if err := dr.wallet.WriteFileForAccount(ctx, accountName, keystoreFileName, encoded); err != nil {
			t.Fatal(err)
		}
	}
	return wantedPublicKeys
}
