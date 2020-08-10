package direct

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

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
	password := "secretPassw0rd$1999"
	wallet := &mock.Wallet{
		Files: make(map[string]map[string][]byte),
	}
	dr := &Keymanager{
		keysCache:        make(map[[48]byte]bls.SecretKey),
		wallet:           wallet,
		accountsStore:    &AccountStore{},
		accountsPassword: password,
	}
	ctx := context.Background()
	accountName, err := dr.CreateAccount(ctx)
	require.NoError(t, err)

	// Ensure the keystore file was written to the wallet
	// and ensure we can decrypt it using the EIP-2335 standard.
	var encodedKeystore []byte
	for k, v := range wallet.Files[AccountsPath] {
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

func TestDirectKeymanager_RemoveAccount(t *testing.T) {
	hook := logTest.NewGlobal()
	password := "secretPassw0rd$1999"
	wallet := &mock.Wallet{
		Files: make(map[string]map[string][]byte),
	}
	dr := &Keymanager{
		keysCache:     make(map[[48]byte]bls.SecretKey),
		wallet:        wallet,
		accountsStore: &AccountStore{},
	}
	numAccounts := 5
	ctx := context.Background()
	for i := 0; i < numAccounts; i++ {
		_, err := dr.CreateAccount(ctx)
		require.NoError(t, err)
	}
	accounts, err := dr.ValidatingAccountNames()
	require.NoError(t, err)
	require.Equal(t, numAccounts, len(accounts))

	accountToRemove := uint64(2)
	accountName := accounts[accountToRemove]
	// Remove an account from the keystore.
	require.NoError(t, dr.DeleteAccount(ctx, accountToRemove))
	// Ensure the keystore file was written to the wallet
	// and ensure we can decrypt it using the EIP-2335 standard.
	var encodedKeystore []byte
	for k, v := range wallet.Files[AccountsPath] {
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

	require.Equal(t, numAccounts-1, len(store.PublicKeys))
	require.Equal(t, numAccounts-1, len(store.PrivateKeys))
	testutil.AssertLogsContain(t, hook, accountName)
	testutil.AssertLogsContain(t, hook, "Successfully removed validator account")
}

func TestDirectKeymanager_FetchValidatingPublicKeys(t *testing.T) {
	password := "secretPassw0rd$1999"
	wallet := &mock.Wallet{
		Files: make(map[string]map[string][]byte),
	}
	dr := &Keymanager{
		wallet:           wallet,
		keysCache:        make(map[[48]byte]bls.SecretKey),
		accountsStore:    &AccountStore{},
		accountsPassword: password,
	}
	// First, generate accounts and their keystore.json files.
	ctx := context.Background()
	numAccounts := 10
	wantedPubKeys := make([][48]byte, numAccounts)
	for i := 0; i < numAccounts; i++ {
		privKey := bls.RandKey()
		pubKey := bytesutil.ToBytes48(privKey.PublicKey().Marshal())
		dr.keysCache[pubKey] = privKey
		wantedPubKeys[i] = pubKey
		dr.accountsStore.PublicKeys = append(dr.accountsStore.PublicKeys, pubKey[:])
		dr.accountsStore.PrivateKeys = append(dr.accountsStore.PrivateKeys, privKey.Marshal())
	}

	publicKeys, err := dr.FetchValidatingPublicKeys(ctx)
	require.NoError(t, err)
	// The results are not guaranteed to be ordered, so we ensure each
	// key we expect exists in the results via a map.
	keysMap := make(map[[48]byte]bool)
	for _, key := range publicKeys {
		keysMap[key] = true
	}
	for _, wanted := range wantedPubKeys {
		if _, ok := keysMap[wanted]; !ok {
			t.Errorf("Could not find expected public key %#x in results", wanted)
		}
	}
}

func TestDirectKeymanager_Sign(t *testing.T) {
	password := "secretPassw0rd$1999"
	wallet := &mock.Wallet{
		Files:            make(map[string]map[string][]byte),
		AccountPasswords: make(map[string]string),
	}
	dr := &Keymanager{
		wallet:           wallet,
		accountsStore:    &AccountStore{},
		keysCache:        make(map[[48]byte]bls.SecretKey),
		accountsPassword: password,
	}

	// First, generate accounts and their keystore.json files.
	ctx := context.Background()
	numAccounts := 10
	for i := 0; i < numAccounts; i++ {
		_, err := dr.CreateAccount(ctx)
		require.NoError(t, err)
	}

	var encodedKeystore []byte
	for k, v := range wallet.Files[AccountsPath] {
		if strings.Contains(k, "keystore") {
			encodedKeystore = v
		}
	}
	keystoreFile := &v2keymanager.Keystore{}
	require.NoError(t, json.Unmarshal(encodedKeystore, keystoreFile))

	// We extract the validator signing private key from the keystore
	// by utilizing the password and initialize a new BLS secret key from
	// its raw bytes.
	decryptor := keystorev4.New()
	enc, err := decryptor.Decrypt(keystoreFile.Crypto, dr.accountsPassword)
	require.NoError(t, err)
	store := &AccountStore{}
	require.NoError(t, json.Unmarshal(enc, store))
	require.Equal(t, len(store.PublicKeys), len(store.PrivateKeys))
	require.NotEqual(t, 0, len(store.PublicKeys))

	for i := 0; i < len(store.PublicKeys); i++ {
		privKey, err := bls.SecretKeyFromBytes(store.PrivateKeys[i])
		require.NoError(t, err)
		dr.keysCache[bytesutil.ToBytes48(store.PublicKeys[i])] = privKey
	}
	dr.accountsStore = store

	publicKeys, err := dr.FetchValidatingPublicKeys(ctx)
	require.NoError(t, err)
	require.Equal(t, len(publicKeys), len(store.PublicKeys))

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
