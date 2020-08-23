package derived

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/google/uuid"
	validatorpb "github.com/prysmaticlabs/prysm/proto/validator/accounts/v2"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/rand"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	mock "github.com/prysmaticlabs/prysm/validator/accounts/v2/testing"
	logTest "github.com/sirupsen/logrus/hooks/test"
	"github.com/tyler-smith/go-bip39"
	util "github.com/wealdtech/go-eth2-util"
	keystorev4 "github.com/wealdtech/go-eth2-wallet-encryptor-keystorev4"
)

func TestDerivedKeymanager_RecoverSeedRoundTrip(t *testing.T) {
	mnemonicEntropy := make([]byte, 32)
	n, err := rand.NewGenerator().Read(mnemonicEntropy)
	require.NoError(t, err)
	require.Equal(t, n, len(mnemonicEntropy))
	mnemonic, err := bip39.NewMnemonic(mnemonicEntropy)
	require.NoError(t, err)
	walletSeed := bip39.NewSeed(mnemonic, "")
	encryptor := keystorev4.New()
	password := "Passwz0rdz2020%"
	cryptoFields, err := encryptor.Encrypt(walletSeed, password)
	require.NoError(t, err)
	id, err := uuid.NewRandom()
	require.NoError(t, err)
	cfg := &SeedConfig{
		Crypto:      cryptoFields,
		ID:          id.String(),
		NextAccount: 0,
		Version:     encryptor.Version(),
		Name:        encryptor.Name(),
	}

	// Ensure we can decrypt the newly recovered config.
	decryptor := keystorev4.New()
	seed, err := decryptor.Decrypt(cfg.Crypto, password)
	assert.NoError(t, err)

	// Ensure the decrypted seed matches the old wallet seed and the new wallet seed.
	assert.DeepEqual(t, walletSeed, seed)
}

func TestDerivedKeymanager_CreateAccount(t *testing.T) {
	hook := logTest.NewGlobal()
	wallet := &mock.Wallet{
		Files:            make(map[string]map[string][]byte),
		AccountPasswords: make(map[string]string),
	}
	seed := make([]byte, 32)
	copy(seed, "hello world")
	password := "secretPassw0rd$1999"
	dr := &Keymanager{
		wallet: wallet,
		seed:   seed,
		seedCfg: &SeedConfig{
			NextAccount: 0,
		},
		accountsPassword: password,
	}
	ctx := context.Background()
	accountName, err := dr.CreateAccount(ctx, true /*logAccountInfo*/)
	require.NoError(t, err)
	assert.Equal(t, "0", accountName)

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
	require.LogsContain(t, hook, "Successfully created new validator account")
}

func TestDerivedKeymanager_FetchValidatingPublicKeys(t *testing.T) {
	wallet := &mock.Wallet{
		Files:            make(map[string]map[string][]byte),
		AccountPasswords: make(map[string]string),
	}
	dr := &Keymanager{
		wallet:    wallet,
		keysCache: make(map[[48]byte]bls.SecretKey),
		seedCfg: &SeedConfig{
			NextAccount: 0,
		},
		seed:             make([]byte, 32),
		accountsPassword: "hello world",
	}
	// First, generate accounts and their keystore.json files.
	ctx := context.Background()
	numAccounts := 20
	wantedPublicKeys := make([][48]byte, numAccounts)
	var err error
	var accountName string
	for i := 0; i < numAccounts; i++ {
		accountName, err = dr.CreateAccount(ctx, false /*logAccountInfo*/)
		require.NoError(t, err)
		validatingKeyPath := fmt.Sprintf(ValidatingKeyDerivationPathTemplate, i)
		validatingKey, err := util.PrivateKeyFromSeedAndPath(dr.seed, validatingKeyPath)
		require.NoError(t, err)
		wantedPublicKeys[i] = bytesutil.ToBytes48(validatingKey.PublicKey().Marshal())
	}
	assert.Equal(t, fmt.Sprintf("%d", numAccounts-1), accountName)

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

func TestDerivedKeymanager_Sign(t *testing.T) {
	wallet := &mock.Wallet{
		Files:            make(map[string]map[string][]byte),
		AccountPasswords: make(map[string]string),
	}
	seed := make([]byte, 32)
	copy(seed, "hello world")
	dr := &Keymanager{
		wallet:    wallet,
		seed:      seed,
		keysCache: make(map[[48]byte]bls.SecretKey),
		seedCfg: &SeedConfig{
			NextAccount: 0,
		},
		accountsPassword: "hello world",
	}

	// First, generate some accounts.
	numAccounts := 2
	ctx := context.Background()
	var err error
	var accountName string
	for i := 0; i < numAccounts; i++ {
		accountName, err = dr.CreateAccount(ctx, false /*logAccountInfo*/)
		require.NoError(t, err)
	}
	assert.Equal(t, fmt.Sprintf("%d", numAccounts-1), accountName)

	// Initialize the secret keys cache for the keymanager.
	require.NoError(t, dr.initializeSecretKeysCache())
	publicKeys, err := dr.FetchValidatingPublicKeys(ctx)
	require.NoError(t, err)

	// We prepare naive data to sign.
	data := []byte("eth2data")
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

	// Check if the signature verifies.
	assert.Equal(t, true, sig.Verify(pubKey, data))
	// Check if the bad signature fails.
	assert.Equal(t, false, sig.Verify(wrongPubKey, data))
}

func TestDerivedKeymanager_Sign_NoPublicKeySpecified(t *testing.T) {
	req := &validatorpb.SignRequest{
		PublicKey: nil,
	}
	dr := &Keymanager{}
	_, err := dr.Sign(context.Background(), req)
	assert.NotNil(t, err)
	assert.Equal(t, strings.Contains(err.Error(), "nil public key"), true)
}

func TestDerivedKeymanager_Sign_NoPublicKeyInCache(t *testing.T) {
	req := &validatorpb.SignRequest{
		PublicKey: []byte("hello world"),
	}
	dr := &Keymanager{
		keysCache: make(map[[48]byte]bls.SecretKey),
	}
	_, err := dr.Sign(context.Background(), req)
	assert.NotNil(t, err)
	assert.Equal(t, strings.Contains(err.Error(), "no signing key found"), true)
}
