package derived

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/google/uuid"
	validatorpb "github.com/prysmaticlabs/prysm/proto/validator/accounts/v2"
	"github.com/prysmaticlabs/prysm/shared/abool"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/rand"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	mock "github.com/prysmaticlabs/prysm/validator/accounts/testing"
	logTest "github.com/sirupsen/logrus/hooks/test"
	"github.com/tyler-smith/go-bip39"
	util "github.com/wealdtech/go-eth2-util"
	keystorev4 "github.com/wealdtech/go-eth2-wallet-encryptor-keystorev4"
)

// We test that using a '25th word' mnemonic passphrase leads to different
// public keys derived than not specifying the passphrase.
func TestDerivedKeymanager_MnemnonicPassphrase_DifferentResults(t *testing.T) {
	sampleMnemonic := "tumble turn jewel sudden social great water general cabin jacket bounce dry flip monster advance problem social half flee inform century chicken hard reason"
	ctx := context.Background()
	wallet := &mock.Wallet{
		Files:            make(map[string]map[string][]byte),
		AccountPasswords: make(map[string]string),
		WalletPassword:   "secretPassw0rd$1999",
	}
	km, err := NewKeymanager(ctx, &SetupConfig{
		Opts:   DefaultKeymanagerOpts(),
		Wallet: wallet,
	})
	require.NoError(t, err)
	numAccounts := 5
	err = km.RecoverAccountsFromMnemonic(ctx, sampleMnemonic, "mnemonicpass", numAccounts)
	require.NoError(t, err)
	without25thWord, err := km.FetchValidatingPublicKeys(ctx)
	require.NoError(t, err)
	wallet = &mock.Wallet{
		Files:            make(map[string]map[string][]byte),
		AccountPasswords: make(map[string]string),
		WalletPassword:   "secretPassw0rd$1999",
	}
	km, err = NewKeymanager(ctx, &SetupConfig{
		Opts:   DefaultKeymanagerOpts(),
		Wallet: wallet,
	})
	require.NoError(t, err)
	// No mnemonic passphrase this time.
	err = km.RecoverAccountsFromMnemonic(ctx, sampleMnemonic, "", numAccounts)
	with25thWord, err := km.FetchValidatingPublicKeys(ctx)
	require.NoError(t, err)
	for i, k := range with25thWord {
		without := without25thWord[i]
		assert.DeepNotEqual(t, k, without)
	}
}

func TestDerivedKeymanager_RecoverSeedRoundTrip(t *testing.T) {
	mnemonicEntropy := make([]byte, 32)
	n, err := rand.NewGenerator().Read(mnemonicEntropy)
	require.NoError(t, err)
	require.Equal(t, n, len(mnemonicEntropy))
	mnemonic, err := bip39.NewMnemonic(mnemonicEntropy)
	require.NoError(t, err)
	wanted := bip39.NewSeed(mnemonic, "")

	got, err := seedFromMnemonic(mnemonic, "" /* no passphrase */)
	require.NoError(t, err)
	// Ensure the derived seed matches.
	assert.DeepEqual(t, wanted, got)
}

func TestDerivedKeymanager_FetchValidatingPublicKeys(t *testing.T) {
	wallet := &mock.Wallet{
		Files:            make(map[string]map[string][]byte),
		AccountPasswords: make(map[string]string),
		WalletPassword:   "secretPassw0rd$1999",
	}
	dr := &Keymanager{
		wallet: wallet,
		seedCfg: &SeedConfig{
			NextAccount: 0,
		},
		seed: make([]byte, 32),
		opts: DefaultKeymanagerOpts(),
	}
	require.NoError(t, dr.initializeKeysCachesFromSeed())
	// First, generate accounts and their keystore.json files.
	ctx := context.Background()
	numAccounts := 20
	wantedPublicKeys := make([][48]byte, numAccounts)
	for i := 0; i < numAccounts; i++ {
		_, _, err := dr.CreateAccount(ctx)
		require.NoError(t, err)
		validatingKeyPath := fmt.Sprintf(ValidatingKeyDerivationPathTemplate, i)
		validatingKey, err := util.PrivateKeyFromSeedAndPath(dr.seed, validatingKeyPath)
		require.NoError(t, err)
		wantedPublicKeys[i] = bytesutil.ToBytes48(validatingKey.PublicKey().Marshal())
	}
	publicKeys, err := dr.FetchValidatingPublicKeys(ctx)
	require.NoError(t, err)
	require.Equal(t, numAccounts, len(publicKeys))

	// FetchValidatingPublicKeys is also used in generating the output of account list
	// therefore the results must be in the same order as the order in which the accounts were derived
	for i, key := range wantedPublicKeys {
		assert.Equal(t, key, publicKeys[i])
	}
}

func TestDerivedKeymanager_FetchValidatingPrivateKeys(t *testing.T) {
	wallet := &mock.Wallet{
		Files:            make(map[string]map[string][]byte),
		AccountPasswords: make(map[string]string),
		WalletPassword:   "secretPassw0rd$1999",
	}
	dr := &Keymanager{
		wallet: wallet,
		seedCfg: &SeedConfig{
			NextAccount: 0,
		},
		seed: make([]byte, 32),
		opts: DefaultKeymanagerOpts(),
	}
	require.NoError(t, dr.initializeKeysCachesFromSeed())
	// First, generate accounts and their keystore.json files.
	ctx := context.Background()
	numAccounts := 20
	wantedPrivateKeys := make([][32]byte, numAccounts)
	for i := 0; i < numAccounts; i++ {
		_, _, err := dr.CreateAccount(ctx)
		require.NoError(t, err)
		validatingKeyPath := fmt.Sprintf(ValidatingKeyDerivationPathTemplate, i)
		validatingKey, err := util.PrivateKeyFromSeedAndPath(dr.seed, validatingKeyPath)
		require.NoError(t, err)
		wantedPrivateKeys[i] = bytesutil.ToBytes32(validatingKey.Marshal())
	}
	privateKeys, err := dr.FetchValidatingPrivateKeys(ctx)
	require.NoError(t, err)
	require.Equal(t, numAccounts, len(privateKeys))

	// FetchValidatingPrivateKeys is also used in generating the output of account list
	// therefore the results must be in the same order as the order in which the accounts were derived
	for i, key := range wantedPrivateKeys {
		assert.Equal(t, key, privateKeys[i])
	}
}

func TestDerivedKeymanager_Sign(t *testing.T) {
	wallet := &mock.Wallet{
		Files:            make(map[string]map[string][]byte),
		AccountPasswords: make(map[string]string),
		WalletPassword:   "secretPassw0rd$1999",
	}
	seed := make([]byte, 32)
	copy(seed, "hello world")
	dr := &Keymanager{
		wallet: wallet,
		seed:   seed,
		seedCfg: &SeedConfig{
			NextAccount: 0,
		},
		opts: DefaultKeymanagerOpts(),
	}
	require.NoError(t, dr.initializeKeysCachesFromSeed())

	// First, generate some accounts.
	numAccounts := 2
	ctx := context.Background()
	for i := 0; i < numAccounts; i++ {
		_, _, err := dr.CreateAccount(ctx)
		require.NoError(t, err)
	}
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
	assert.ErrorContains(t, "nil public key", err)
}

func TestDerivedKeymanager_Sign_NoPublicKeyInCache(t *testing.T) {
	req := &validatorpb.SignRequest{
		PublicKey: []byte("hello world"),
	}
	dr := &Keymanager{}
	_, err := dr.Sign(context.Background(), req)
	assert.ErrorContains(t, "no signing key found", err)
}
