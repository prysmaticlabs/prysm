package imported

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	validatorpb "github.com/prysmaticlabs/prysm/proto/validator/accounts/v2"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/petnames"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	mock "github.com/prysmaticlabs/prysm/validator/accounts/testing"
	"github.com/prysmaticlabs/prysm/validator/keymanager"
	logTest "github.com/sirupsen/logrus/hooks/test"
	keystorev4 "github.com/wealdtech/go-eth2-wallet-encryptor-keystorev4"
)

func TestImportedKeymanager_CreateAccount(t *testing.T) {
	hook := logTest.NewGlobal()
	password := "secretPassw0rd$1999"
	wallet := &mock.Wallet{
		Files:          make(map[string]map[string][]byte),
		WalletPassword: password,
	}
	dr := &Keymanager{
		wallet:        wallet,
		accountsStore: &AccountStore{},
	}
	ctx := context.Background()
	createdPubKey, _, err := dr.CreateAccount(ctx)
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
	keystoreFile := &keymanager.Keystore{}
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
	require.LogsContain(t, hook, petnames.DeterministicName(createdPubKey, "-"))
	require.LogsContain(t, hook, "Successfully created new validator account")
}

func TestImportedKeymanager_RemoveAccounts(t *testing.T) {
	hook := logTest.NewGlobal()
	password := "secretPassw0rd$1999"
	wallet := &mock.Wallet{
		Files:          make(map[string]map[string][]byte),
		WalletPassword: password,
	}
	dr := &Keymanager{
		wallet:        wallet,
		accountsStore: &AccountStore{},
	}
	numAccounts := 5
	ctx := context.Background()
	for i := 0; i < numAccounts; i++ {
		_, _, err := dr.CreateAccount(ctx)
		require.NoError(t, err)
	}
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
	store := &AccountStore{}
	require.NoError(t, json.Unmarshal(encodedAccounts, store))

	require.Equal(t, numAccounts-1, len(store.PublicKeys))
	require.Equal(t, numAccounts-1, len(store.PrivateKeys))
	require.LogsContain(t, hook, fmt.Sprintf("%#x", bytesutil.Trunc(accountPubKey[:])))
	require.LogsContain(t, hook, "Successfully deleted validator account")
}

func TestImportedKeymanager_FetchValidatingPublicKeys(t *testing.T) {
	password := "secretPassw0rd$1999"
	wallet := &mock.Wallet{
		Files:          make(map[string]map[string][]byte),
		WalletPassword: password,
	}
	dr := &Keymanager{
		wallet:        wallet,
		accountsStore: &AccountStore{},
	}
	// First, generate accounts and their keystore.json files.
	ctx := context.Background()
	numAccounts := 10
	wantedPubKeys := make([][48]byte, numAccounts)
	for i := 0; i < numAccounts; i++ {
		privKey := bls.RandKey()
		pubKey := bytesutil.ToBytes48(privKey.PublicKey().Marshal())
		wantedPubKeys[i] = pubKey
		dr.accountsStore.PublicKeys = append(dr.accountsStore.PublicKeys, pubKey[:])
		dr.accountsStore.PrivateKeys = append(dr.accountsStore.PrivateKeys, privKey.Marshal())
	}
	require.NoError(t, dr.initializeKeysCachesFromKeystore())
	publicKeys, err := dr.FetchValidatingPublicKeys(ctx)
	require.NoError(t, err)
	assert.Equal(t, numAccounts, len(publicKeys))
	// FetchValidatingPublicKeys is also used in generating the output of account list
	// therefore the results must be in the same order as the order in which the accounts were derived
	for i, key := range wantedPubKeys {
		assert.Equal(t, key, publicKeys[i])
	}
}

func TestImportedKeymanager_FetchValidatingPrivateKeys(t *testing.T) {
	password := "secretPassw0rd$1999"
	wallet := &mock.Wallet{
		Files:          make(map[string]map[string][]byte),
		WalletPassword: password,
	}
	dr := &Keymanager{
		wallet:        wallet,
		accountsStore: &AccountStore{},
	}
	// First, generate accounts and their keystore.json files.
	ctx := context.Background()
	numAccounts := 10
	wantedPrivateKeys := make([][32]byte, numAccounts)
	for i := 0; i < numAccounts; i++ {
		privKey := bls.RandKey()
		privKeyData := privKey.Marshal()
		pubKey := bytesutil.ToBytes48(privKey.PublicKey().Marshal())
		wantedPrivateKeys[i] = bytesutil.ToBytes32(privKeyData)
		dr.accountsStore.PublicKeys = append(dr.accountsStore.PublicKeys, pubKey[:])
		dr.accountsStore.PrivateKeys = append(dr.accountsStore.PrivateKeys, privKeyData)
	}
	require.NoError(t, dr.initializeKeysCachesFromKeystore())
	privateKeys, err := dr.FetchValidatingPrivateKeys(ctx)
	require.NoError(t, err)
	assert.Equal(t, numAccounts, len(privateKeys))
	// FetchValidatingPrivateKeys is also used in generating the output of account list
	// therefore the results must be in the same order as the order in which the accounts were created
	for i, key := range wantedPrivateKeys {
		assert.Equal(t, key, privateKeys[i])
	}
}

func TestImportedKeymanager_Sign(t *testing.T) {
	password := "secretPassw0rd$1999"
	wallet := &mock.Wallet{
		Files:            make(map[string]map[string][]byte),
		AccountPasswords: make(map[string]string),
		WalletPassword:   password,
	}
	dr := &Keymanager{
		wallet:        wallet,
		accountsStore: &AccountStore{},
	}

	// First, generate accounts and their keystore.json files.
	ctx := context.Background()
	numAccounts := 10
	for i := 0; i < numAccounts; i++ {
		_, _, err := dr.CreateAccount(ctx)
		require.NoError(t, err)
	}

	var encodedKeystore []byte
	for k, v := range wallet.Files[AccountsPath] {
		if strings.Contains(k, "keystore") {
			encodedKeystore = v
		}
	}
	keystoreFile := &keymanager.Keystore{}
	require.NoError(t, json.Unmarshal(encodedKeystore, keystoreFile))

	// We extract the validator signing private key from the keystore
	// by utilizing the password and initialize a new BLS secret key from
	// its raw bytes.
	decryptor := keystorev4.New()
	enc, err := decryptor.Decrypt(keystoreFile.Crypto, dr.wallet.Password())
	require.NoError(t, err)
	store := &AccountStore{}
	require.NoError(t, json.Unmarshal(enc, store))
	require.Equal(t, len(store.PublicKeys), len(store.PrivateKeys))
	require.NotEqual(t, 0, len(store.PublicKeys))
	dr.accountsStore = store
	require.NoError(t, dr.initializeKeysCachesFromKeystore())
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

func TestImportedKeymanager_Sign_NoPublicKeySpecified(t *testing.T) {
	req := &validatorpb.SignRequest{
		PublicKey: nil,
	}
	dr := &Keymanager{}
	_, err := dr.Sign(context.Background(), req)
	assert.ErrorContains(t, "nil public key", err)
}

func TestImportedKeymanager_Sign_NoPublicKeyInCache(t *testing.T) {
	req := &validatorpb.SignRequest{
		PublicKey: []byte("hello world"),
	}
	secretKeysCache = make(map[[48]byte]bls.SecretKey)
	dr := &Keymanager{}
	_, err := dr.Sign(context.Background(), req)
	assert.ErrorContains(t, "no signing key found in keys cache", err)
}

func TestImportedKeymanager_RefreshWalletPassword(t *testing.T) {
	password := "secretPassw0rd$1999"
	wallet := &mock.Wallet{
		Files:            make(map[string]map[string][]byte),
		AccountPasswords: make(map[string]string),
		WalletPassword:   password,
	}
	dr := &Keymanager{
		wallet:        wallet,
		accountsStore: &AccountStore{},
	}

	ctx := context.Background()
	numAccounts := 5
	for i := 0; i < numAccounts; i++ {
		_, _, err := dr.CreateAccount(ctx)
		require.NoError(t, err)
	}

	var encodedKeystore []byte
	for k, v := range wallet.Files[AccountsPath] {
		if strings.Contains(k, "keystore") {
			encodedKeystore = v
		}
	}
	keystoreFile := &keymanager.Keystore{}
	require.NoError(t, json.Unmarshal(encodedKeystore, keystoreFile))

	// We attempt to decrypt with the wallet password and expect no error.
	decryptor := keystorev4.New()
	_, err := decryptor.Decrypt(keystoreFile.Crypto, dr.wallet.Password())
	require.NoError(t, err)

	// We change the wallet password.
	wallet.WalletPassword = "NewPassw0rdz9**#"
	// Attempting to decrypt with this new wallet password should fail.
	_, err = decryptor.Decrypt(keystoreFile.Crypto, dr.wallet.Password())
	require.ErrorContains(t, "invalid checksum", err)

	// Call the refresh wallet password method, then attempting to decrypt should work.
	require.NoError(t, dr.RefreshWalletPassword(ctx))
	for k, v := range wallet.Files[AccountsPath] {
		if strings.Contains(k, "keystore") {
			encodedKeystore = v
		}
	}
	keystoreFile = &keymanager.Keystore{}
	require.NoError(t, json.Unmarshal(encodedKeystore, keystoreFile))
	_, err = decryptor.Decrypt(keystoreFile.Crypto, dr.wallet.Password())
	require.NoError(t, err)
}
