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
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	mock "github.com/prysmaticlabs/prysm/validator/accounts/v2/testing"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	logTest "github.com/sirupsen/logrus/hooks/test"
	keystorev4 "github.com/wealdtech/go-eth2-wallet-encryptor-keystorev4"
)

func TestDirectKeymanager_CreateAccount(t *testing.T) {
	hook := logTest.NewGlobal()
	wallet := &mock.Wallet{
		Files:            make(map[string]map[string][]byte),
		AccountPasswords: make(map[string]string),
	}
	dr := &Keymanager{
		wallet: wallet,
	}
	ctx := context.Background()
	password := "secretPassw0rd$1999"
	accountName, err := dr.CreateAccount(ctx, password)
	require.NoError(t, err)

	// Ensure the keystore file was written to the wallet
	// and ensure we can decrypt it using the EIP-2335 standard.
	var encodedKeystore []byte
	for k, v := range wallet.Files[accountName] {
		if strings.Contains(k, "keystore") {
			encodedKeystore = v
		}
	}
	require.NotNil(t, encodedKeystore, "could not find keystore file")
	keystoreFile := &v2keymanager.Keystore{}
	require.NoError(t, json.Unmarshal(encodedKeystore, keystoreFile))

	// We extract the validator signing private key from the keystore
	// by utilizing the password and initialize a new BLS secret key from
	// its raw bytes.
	decryptor := keystorev4.New()
	rawSigningKey, err := decryptor.Decrypt(keystoreFile.Crypto, password)
	require.NoError(t, err, "Could not decrypt validator signing key")
	validatorSigningKey, err := bls.SecretKeyFromBytes(rawSigningKey)
	require.NoError(t, err, "Could not instantiate bls secret key from bytes")

	// Decode the deposit_data.ssz file and confirm
	// the public key matches the public key from the
	// account's decrypted keystore.
	encodedDepositData, ok := wallet.Files[accountName][depositDataFileName]
	require.Equal(t, true, ok, "Expected to have stored %s in wallet", depositDataFileName)
	depositData := &ethpb.Deposit_Data{}
	require.NoError(t, ssz.Unmarshal(encodedDepositData, depositData))

	depositPublicKey := depositData.PublicKey
	publicKey := validatorSigningKey.PublicKey().Marshal()
	if !bytes.Equal(depositPublicKey, publicKey) {
		t.Errorf(
			"Expected deposit data public key %#x to match public key from keystore %#x",
			depositPublicKey,
			publicKey,
		)
	}

	testutil.AssertLogsContain(t, hook, "Successfully created new validator account")
}

func TestDirectKeymanager_FetchValidatingPublicKeys(t *testing.T) {
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
	numAccounts := 1
	accountNames, wantedPublicKeys := generateAccounts(t, numAccounts, dr)
	wallet.Directories = accountNames
	publicKeys, err := dr.FetchValidatingPublicKeys(ctx)
	require.NoError(t, err)
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

func TestDirectKeymanager_Sign(t *testing.T) {
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
	accountNames, _ := generateAccounts(t, numAccounts, dr)
	wallet.Directories = accountNames

	ctx := context.Background()
	require.NoError(t, dr.initializeSecretKeysCache(ctx))
	publicKeys, err := dr.FetchValidatingPublicKeys(ctx)
	require.NoError(t, err)

	// We prepare naive data to sign.
	data := []byte("hello world")
	signRequest := &validatorpb.SignRequest{
		PublicKey:   publicKeys[0][:],
		SigningRoot: data,
	}
	sig, err := dr.Sign(ctx, signRequest)
	require.NoError(t, err)
	pubKey, err := bls.PublicKeyFromBytes(publicKeys[0][:])
	require.NoError(t, err)
	wrongPubKey, err := bls.PublicKeyFromBytes(publicKeys[1][:])
	require.NoError(t, err)
	if !sig.Verify(pubKey, data) {
		t.Fatalf("Expected sig to verify for pubkey %#x and data %v", pubKey.Marshal(), data)
	}
	if sig.Verify(wrongPubKey, data) {
		t.Fatalf("Expected sig not to verify for pubkey %#x and data %v", wrongPubKey.Marshal(), data)
	}
}
func TestDirectKeymanager_Sign_NoPublicKeySpecified(t *testing.T) {
	req := &validatorpb.SignRequest{
		PublicKey: nil,
	}
	dr := &Keymanager{}
	_, err := dr.Sign(context.Background(), req)
	assert.ErrorContains(t, "nil public key", err)
}

func TestDirectKeymanager_Sign_NoPublicKeyInCache(t *testing.T) {
	req := &validatorpb.SignRequest{
		PublicKey: []byte("hello world"),
	}
	dr := &Keymanager{
		keysCache: make(map[[48]byte]bls.SecretKey),
	}
	_, err := dr.Sign(context.Background(), req)
	assert.ErrorContains(t, "no signing key found in keys cache", err)
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
		_, err := dr.FetchValidatingPublicKeys(ctx)
		require.NoError(b, err)
	}
}

func generateAccounts(t testing.TB, numAccounts int, dr *Keymanager) ([]string, [][48]byte) {
	ctx := context.Background()
	accountNames := make([]string, numAccounts)
	wantedPublicKeys := make([][48]byte, numAccounts)
	for i := 0; i < numAccounts; i++ {
		validatingKey := bls.RandKey()
		wantedPublicKeys[i] = bytesutil.ToBytes48(validatingKey.PublicKey().Marshal())
		password := strconv.Itoa(i)
		encoded, err := dr.generateKeystoreFile(validatingKey, password)
		require.NoError(t, err)
		accountName, err := dr.generateAccountName(validatingKey.PublicKey().Marshal())
		require.NoError(t, err)
		assert.NoError(t, err, dr.wallet.WriteFileAtPath(ctx, accountName, KeystoreFileName, encoded))
		assert.NoError(t, err, dr.wallet.WritePasswordToDisk(ctx, accountName+PasswordFileSuffix, password))
		accountNames[i] = accountName
	}
	return accountNames, wantedPublicKeys
}
