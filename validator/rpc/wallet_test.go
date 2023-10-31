package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/google/uuid"
	"github.com/prysmaticlabs/prysm/v4/async/event"
	"github.com/prysmaticlabs/prysm/v4/config/features"
	"github.com/prysmaticlabs/prysm/v4/crypto/bls"
	"github.com/prysmaticlabs/prysm/v4/crypto/rand"
	"github.com/prysmaticlabs/prysm/v4/io/file"
	pb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1/validator-client"
	"github.com/prysmaticlabs/prysm/v4/testing/assert"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
	"github.com/prysmaticlabs/prysm/v4/validator/accounts"
	"github.com/prysmaticlabs/prysm/v4/validator/accounts/iface"
	mock "github.com/prysmaticlabs/prysm/v4/validator/accounts/testing"
	"github.com/prysmaticlabs/prysm/v4/validator/accounts/wallet"
	"github.com/prysmaticlabs/prysm/v4/validator/client"
	"github.com/prysmaticlabs/prysm/v4/validator/keymanager"
	"github.com/tyler-smith/go-bip39"
	keystorev4 "github.com/wealdtech/go-eth2-wallet-encryptor-keystorev4"
)

const strongPass = "29384283xasjasd32%%&*@*#*"

func TestServer_CreateWallet_Local(t *testing.T) {
	ctx := context.Background()
	localWalletDir := setupWalletDir(t)
	defaultWalletPath = localWalletDir
	opts := []accounts.Option{
		accounts.WithWalletDir(defaultWalletPath),
		accounts.WithKeymanagerType(keymanager.Derived),
		accounts.WithWalletPassword(strongPass),
		accounts.WithSkipMnemonicConfirm(true),
	}
	acc, err := accounts.NewCLIManager(opts...)
	require.NoError(t, err)
	w, err := acc.WalletCreate(ctx)
	require.NoError(t, err)
	km, err := w.InitializeKeymanager(ctx, iface.InitKeymanagerConfig{ListenForChanges: false})
	require.NoError(t, err)
	vs, err := client.NewValidatorService(ctx, &client.Config{
		Wallet: w,
		Validator: &mock.Validator{
			Km: km,
		},
	})
	require.NoError(t, err)
	s := &Server{
		walletInitializedFeed: new(event.Feed),
		walletDir:             defaultWalletPath,
		validatorService:      vs,
	}

	_, err = s.CreateWallet(ctx, &pb.CreateWalletRequest{
		Keymanager:     pb.KeymanagerKind_IMPORTED,
		WalletPassword: strongPass,
	})
	require.NoError(t, err)

	encryptor := keystorev4.New()
	keystores := make([]string, 3)
	passwords := make([]string, 3)
	for i := 0; i < len(keystores); i++ {
		privKey, err := bls.RandKey()
		require.NoError(t, err)
		pubKey := fmt.Sprintf("%x", privKey.PublicKey().Marshal())
		id, err := uuid.NewRandom()
		require.NoError(t, err)
		cryptoFields, err := encryptor.Encrypt(privKey.Marshal(), strongPass)
		require.NoError(t, err)
		item := &keymanager.Keystore{
			Crypto:      cryptoFields,
			ID:          id.String(),
			Version:     encryptor.Version(),
			Pubkey:      pubKey,
			Description: encryptor.Name(),
		}
		encodedFile, err := json.MarshalIndent(item, "", "\t")
		require.NoError(t, err)
		keystores[i] = string(encodedFile)
		if i < len(passwords) {
			passwords[i] = strongPass
		}
	}

	importReq := &ImportKeystoresRequest{
		Keystores: keystores,
		Passwords: passwords,
	}

	var buf bytes.Buffer
	err = json.NewEncoder(&buf).Encode(importReq)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/eth/v1/keystores"), &buf)
	wr := httptest.NewRecorder()
	wr.Body = &bytes.Buffer{}
	s.ImportKeystores(wr, req)
	require.Equal(t, http.StatusOK, wr.Code)
	resp := &ImportKeystoresResponse{}
	require.NoError(t, json.Unmarshal(wr.Body.Bytes(), resp))
	for _, status := range resp.Data {
		require.Equal(t, keymanager.StatusImported, status.Status)
	}
	keys, err := km.FetchValidatingPublicKeys(ctx)
	require.NoError(t, err)
	require.Equal(t, len(keys), len(keystores))
}

func TestServer_CreateWallet_Local_PasswordTooWeak(t *testing.T) {
	localWalletDir := setupWalletDir(t)
	defaultWalletPath = localWalletDir
	ctx := context.Background()
	s := &Server{
		walletInitializedFeed: new(event.Feed),
		walletDir:             defaultWalletPath,
	}
	req := &pb.CreateWalletRequest{
		Keymanager:     pb.KeymanagerKind_IMPORTED,
		WalletPassword: "", // Weak password, empty string
	}
	_, err := s.CreateWallet(ctx, req)
	require.ErrorContains(t, "Password too weak", err)

	req = &pb.CreateWalletRequest{
		Keymanager:     pb.KeymanagerKind_IMPORTED,
		WalletPassword: "a", // Weak password, too short
	}
	_, err = s.CreateWallet(ctx, req)
	require.ErrorContains(t, "Password too weak", err)
}

func TestServer_RecoverWallet_Derived(t *testing.T) {
	localWalletDir := setupWalletDir(t)
	ctx := context.Background()
	s := &Server{
		walletInitializedFeed: new(event.Feed),
		walletDir:             localWalletDir,
	}
	req := &pb.RecoverWalletRequest{
		WalletPassword: strongPass,
		NumAccounts:    0,
	}
	_, err := s.RecoverWallet(ctx, req)
	require.ErrorContains(t, "Must create at least 1 validator account", err)

	req.NumAccounts = 2
	req.Language = "Swahili"
	_, err = s.RecoverWallet(ctx, req)
	require.ErrorContains(t, "input not in the list of supported languages", err)

	req.Language = "ENglish"
	_, err = s.RecoverWallet(ctx, req)
	require.ErrorContains(t, "invalid mnemonic in request", err)

	mnemonicRandomness := make([]byte, 32)
	_, err = rand.NewGenerator().Read(mnemonicRandomness)
	require.NoError(t, err)
	mnemonic, err := bip39.NewMnemonic(mnemonicRandomness)
	require.NoError(t, err)
	req.Mnemonic = mnemonic

	req.Mnemonic25ThWord = " "
	_, err = s.RecoverWallet(ctx, req)
	require.ErrorContains(t, "mnemonic 25th word cannot be empty", err)
	req.Mnemonic25ThWord = "outer"

	// Test weak password.
	req.WalletPassword = "123qwe"
	_, err = s.RecoverWallet(ctx, req)
	require.ErrorContains(t, "password did not pass validation", err)

	req.WalletPassword = strongPass
	// Create(derived) should fail then test recover.
	reqCreate := &pb.CreateWalletRequest{
		Keymanager:     pb.KeymanagerKind_DERIVED,
		WalletPassword: strongPass,
		NumAccounts:    2,
		Mnemonic:       mnemonic,
	}
	_, err = s.CreateWallet(ctx, reqCreate)
	require.ErrorContains(t, "create wallet not supported through web", err, "Create wallet for DERIVED or REMOTE types not supported through web, either import keystore or recover")

	// This defer will be the last to execute in this func.
	resetCfgFalse := features.InitWithReset(&features.Flags{
		WriteWalletPasswordOnWebOnboarding: false,
	})
	defer resetCfgFalse()

	resetCfgTrue := features.InitWithReset(&features.Flags{
		WriteWalletPasswordOnWebOnboarding: true,
	})
	defer resetCfgTrue()

	// Finally test recover.
	_, err = s.RecoverWallet(ctx, req)
	require.NoError(t, err)

	// Password File should have been written.
	passwordFilePath := filepath.Join(localWalletDir, wallet.DefaultWalletPasswordFile)
	assert.Equal(t, true, file.Exists(passwordFilePath))

	// Attempting to write again should trigger an error.
	err = writeWalletPasswordToDisk(localWalletDir, "somepassword")
	require.ErrorContains(t, "cannot write wallet password file as it already exists", err)

}

func TestServer_ValidateKeystores_FailedPreconditions(t *testing.T) {
	ctx := context.Background()
	strongPass := "29384283xasjasd32%%&*@*#*"
	ss := &Server{}
	_, err := ss.ValidateKeystores(ctx, &pb.ValidateKeystoresRequest{})
	assert.ErrorContains(t, "Password required for keystores", err)
	_, err = ss.ValidateKeystores(ctx, &pb.ValidateKeystoresRequest{
		KeystoresPassword: strongPass,
	})
	assert.ErrorContains(t, "No keystores included in request", err)
	_, err = ss.ValidateKeystores(ctx, &pb.ValidateKeystoresRequest{
		KeystoresPassword: strongPass,
		Keystores:         []string{"badjson"},
	})
	assert.ErrorContains(t, "Not a valid EIP-2335 keystore", err)
}

func TestServer_ValidateKeystores_OK(t *testing.T) {
	ctx := context.Background()
	strongPass := "29384283xasjasd32%%&*@*#*"
	ss := &Server{}

	// Create 3 keystores with the strong password.
	encryptor := keystorev4.New()
	keystores := make([]string, 3)
	pubKeys := make([][]byte, 3)
	for i := 0; i < len(keystores); i++ {
		privKey, err := bls.RandKey()
		require.NoError(t, err)
		pubKey := fmt.Sprintf("%x", privKey.PublicKey().Marshal())
		id, err := uuid.NewRandom()
		require.NoError(t, err)
		cryptoFields, err := encryptor.Encrypt(privKey.Marshal(), strongPass)
		require.NoError(t, err)
		item := &keymanager.Keystore{
			Crypto:      cryptoFields,
			ID:          id.String(),
			Version:     encryptor.Version(),
			Pubkey:      pubKey,
			Description: encryptor.Name(),
		}
		encodedFile, err := json.MarshalIndent(item, "", "\t")
		require.NoError(t, err)
		keystores[i] = string(encodedFile)
		pubKeys[i] = privKey.PublicKey().Marshal()
	}

	// Validate the keystores and ensure no error.
	_, err := ss.ValidateKeystores(ctx, &pb.ValidateKeystoresRequest{
		KeystoresPassword: strongPass,
		Keystores:         keystores,
	})
	require.NoError(t, err)

	// Check that using a different password will return an error.
	_, err = ss.ValidateKeystores(ctx, &pb.ValidateKeystoresRequest{
		KeystoresPassword: "badpassword",
		Keystores:         keystores,
	})
	require.ErrorContains(t, "is incorrect", err)

	// Add a new keystore that was encrypted with a different password and expect
	// a failure from the function.
	differentPassword := "differentkeystorepass"
	privKey, err := bls.RandKey()
	require.NoError(t, err)
	pubKey := "somepubkey"
	id, err := uuid.NewRandom()
	require.NoError(t, err)
	cryptoFields, err := encryptor.Encrypt(privKey.Marshal(), differentPassword)
	require.NoError(t, err)
	item := &keymanager.Keystore{
		Crypto:      cryptoFields,
		ID:          id.String(),
		Version:     encryptor.Version(),
		Pubkey:      pubKey,
		Description: encryptor.Name(),
	}
	encodedFile, err := json.MarshalIndent(item, "", "\t")
	keystores = append(keystores, string(encodedFile))
	require.NoError(t, err)
	_, err = ss.ValidateKeystores(ctx, &pb.ValidateKeystoresRequest{
		KeystoresPassword: strongPass,
		Keystores:         keystores,
	})
	require.ErrorContains(t, "Password for keystore with public key somepubkey is incorrect", err)
}

func TestServer_WalletConfig_NoWalletFound(t *testing.T) {
	s := &Server{}
	resp, err := s.WalletConfig(context.Background(), &empty.Empty{})
	require.NoError(t, err)
	assert.DeepEqual(t, resp, &pb.WalletResponse{})
}

func TestServer_WalletConfig(t *testing.T) {
	localWalletDir := setupWalletDir(t)
	defaultWalletPath = localWalletDir
	ctx := context.Background()
	s := &Server{
		walletInitializedFeed: new(event.Feed),
		walletDir:             defaultWalletPath,
	}
	// We attempt to create the wallet.
	opts := []accounts.Option{
		accounts.WithWalletDir(defaultWalletPath),
		accounts.WithKeymanagerType(keymanager.Local),
		accounts.WithWalletPassword(strongPass),
		accounts.WithSkipMnemonicConfirm(true),
	}
	acc, err := accounts.NewCLIManager(opts...)
	require.NoError(t, err)
	w, err := acc.WalletCreate(ctx)
	require.NoError(t, err)
	km, err := w.InitializeKeymanager(ctx, iface.InitKeymanagerConfig{ListenForChanges: false})
	require.NoError(t, err)
	s.wallet = w
	vs, err := client.NewValidatorService(ctx, &client.Config{
		Wallet: w,
		Validator: &mock.Validator{
			Km: km,
		},
	})
	require.NoError(t, err)
	s.validatorService = vs
	resp, err := s.WalletConfig(ctx, &empty.Empty{})
	require.NoError(t, err)

	assert.DeepEqual(t, resp, &pb.WalletResponse{
		WalletPath:     localWalletDir,
		KeymanagerKind: pb.KeymanagerKind_IMPORTED,
	})
}

func Test_writeWalletPasswordToDisk(t *testing.T) {
	walletDir := setupWalletDir(t)
	resetCfg := features.InitWithReset(&features.Flags{
		WriteWalletPasswordOnWebOnboarding: false,
	})
	defer resetCfg()
	err := writeWalletPasswordToDisk(walletDir, "somepassword")
	require.NoError(t, err)

	// Expected a silent failure if the feature flag is not enabled.
	passwordFilePath := filepath.Join(walletDir, wallet.DefaultWalletPasswordFile)
	assert.Equal(t, false, file.Exists(passwordFilePath))
	resetCfg = features.InitWithReset(&features.Flags{
		WriteWalletPasswordOnWebOnboarding: true,
	})
	defer resetCfg()
	err = writeWalletPasswordToDisk(walletDir, "somepassword")
	require.NoError(t, err)

	// File should have been written.
	assert.Equal(t, true, file.Exists(passwordFilePath))

	// Attempting to write again should trigger an error.
	err = writeWalletPasswordToDisk(walletDir, "somepassword")
	require.NotNil(t, err)
}

func createRandomKeystore(t testing.TB, password string) *keymanager.Keystore {
	encryptor := keystorev4.New()
	id, err := uuid.NewRandom()
	require.NoError(t, err)
	validatingKey, err := bls.RandKey()
	require.NoError(t, err)
	pubKey := validatingKey.PublicKey().Marshal()
	cryptoFields, err := encryptor.Encrypt(validatingKey.Marshal(), password)
	require.NoError(t, err)
	return &keymanager.Keystore{
		Crypto:      cryptoFields,
		Pubkey:      fmt.Sprintf("%x", pubKey),
		ID:          id.String(),
		Version:     encryptor.Version(),
		Description: encryptor.Name(),
	}
}
