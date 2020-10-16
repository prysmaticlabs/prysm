package derived

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/google/uuid"
	validatorpb "github.com/prysmaticlabs/prysm/proto/validator/accounts/v2"
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
	ctx := context.Background()
	_, _, err := dr.CreateAccount(ctx)
	require.NoError(t, err)

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

func TestDerivedKeymanager_FetchWithdrawalPrivateKeys(t *testing.T) {
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
		withdrawalKeyPath := fmt.Sprintf(WithdrawalKeyDerivationPathTemplate, i)
		withdrawalKey, err := util.PrivateKeyFromSeedAndPath(dr.seed, withdrawalKeyPath)
		require.NoError(t, err)
		wantedPrivateKeys[i] = bytesutil.ToBytes32(withdrawalKey.Marshal())
	}
	privateKeys, err := dr.FetchWithdrawalPrivateKeys(ctx)
	require.NoError(t, err)
	require.Equal(t, numAccounts, len(privateKeys))

	// FetchWithdrawalPrivateKeys is also used in generating the output of account list
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

func TestDerivedKeymanager_RefreshWalletPassword(t *testing.T) {
	password := "secretPassw0rd$1999"
	wallet := &mock.Wallet{
		Files:            make(map[string]map[string][]byte),
		AccountPasswords: make(map[string]string),
		WalletPassword:   password,
	}
	dr := &Keymanager{
		wallet: wallet,
		opts:   DefaultKeymanagerOpts(),
	}
	seedCfg, err := initializeWalletSeedFile(wallet.Password(), true /* skip mnemonic confirm */)
	require.NoError(t, err)
	dr.seedCfg = seedCfg
	decryptor := keystorev4.New()
	seed, err := decryptor.Decrypt(dr.seedCfg.Crypto, wallet.Password())
	require.NoError(t, err)
	dr.seed = seed
	require.NoError(t, dr.initializeKeysCachesFromSeed())

	// First, generate some accounts.
	numAccounts := 2
	ctx := context.Background()
	for i := 0; i < numAccounts; i++ {
		_, _, err := dr.CreateAccount(ctx)
		require.NoError(t, err)
	}

	// We attempt to decrypt with the wallet password and expect no error.
	_, err = decryptor.Decrypt(dr.seedCfg.Crypto, dr.wallet.Password())
	require.NoError(t, err)

	// We change the wallet password.
	wallet.WalletPassword = "NewPassw0rdz9**#"
	// Attempting to decrypt with this new wallet password should fail.
	_, err = decryptor.Decrypt(dr.seedCfg.Crypto, dr.wallet.Password())
	require.ErrorContains(t, "invalid checksum", err)

	// Call the refresh wallet password method, then attempting to decrypt should work.
	require.NoError(t, dr.RefreshWalletPassword(ctx))
	_, err = decryptor.Decrypt(dr.seedCfg.Crypto, dr.wallet.Password())
	require.NoError(t, err)
}
