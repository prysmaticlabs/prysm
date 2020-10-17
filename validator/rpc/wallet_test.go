package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/google/uuid"
	pb "github.com/prysmaticlabs/prysm/proto/validator/accounts/v2"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/prysmaticlabs/prysm/validator/accounts"
	"github.com/prysmaticlabs/prysm/validator/accounts/wallet"
	"github.com/prysmaticlabs/prysm/validator/keymanager"
	"github.com/prysmaticlabs/prysm/validator/keymanager/imported"
	keystorev4 "github.com/wealdtech/go-eth2-wallet-encryptor-keystorev4"
)

func createImportedWalletWithAccounts(t testing.TB, numAccounts int) (*Server, [][]byte) {
	localWalletDir := setupWalletDir(t)
	defaultWalletPath = localWalletDir
	ctx := context.Background()
	strongPass := "29384283xasjasd32%%&*@*#*"
	w, err := accounts.CreateWalletWithKeymanager(ctx, &accounts.CreateWalletConfig{
		WalletCfg: &wallet.Config{
			WalletDir:      defaultWalletPath,
			KeymanagerKind: keymanager.Imported,
			WalletPassword: strongPass,
		},
		SkipMnemonicConfirm: true,
	})
	require.NoError(t, err)
	require.NoError(t, w.SaveHashedPassword(ctx))
	km, err := w.InitializeKeymanager(ctx, true /* skip mnemonic confirm */)
	require.NoError(t, err)
	ss := &Server{
		keymanager:            km,
		wallet:                w,
		walletDir:             defaultWalletPath,
		walletInitializedFeed: new(event.Feed),
	}
	// First we import accounts into the wallet.
	encryptor := keystorev4.New()
	keystores := make([]string, numAccounts)
	pubKeys := make([][]byte, len(keystores))
	for i := 0; i < len(keystores); i++ {
		privKey := bls.RandKey()
		pubKey := fmt.Sprintf("%x", privKey.PublicKey().Marshal())
		id, err := uuid.NewRandom()
		require.NoError(t, err)
		cryptoFields, err := encryptor.Encrypt(privKey.Marshal(), strongPass)
		require.NoError(t, err)
		item := &keymanager.Keystore{
			Crypto:  cryptoFields,
			ID:      id.String(),
			Version: encryptor.Version(),
			Pubkey:  pubKey,
			Name:    encryptor.Name(),
		}
		encodedFile, err := json.MarshalIndent(item, "", "\t")
		require.NoError(t, err)
		keystores[i] = string(encodedFile)
		pubKeys[i] = privKey.PublicKey().Marshal()
	}
	_, err = ss.ImportKeystores(ctx, &pb.ImportKeystoresRequest{
		KeystoresImported: keystores,
		KeystoresPassword: strongPass,
	})
	require.NoError(t, err)
	ss.keymanager, err = ss.wallet.InitializeKeymanager(ctx, true /* skip mnemonic confirm */)
	require.NoError(t, err)
	return ss, pubKeys
}

func TestServer_CreateWallet_Imported(t *testing.T) {
	localWalletDir := setupWalletDir(t)
	defaultWalletPath = localWalletDir
	ctx := context.Background()
	strongPass := "29384283xasjasd32%%&*@*#*"
	s := &Server{
		walletInitializedFeed: new(event.Feed),
		walletDir:             defaultWalletPath,
	}
	_, err := s.Signup(ctx, &pb.AuthRequest{
		Password:  strongPass,
		WalletDir: defaultWalletPath,
	})
	require.NoError(t, err)
	req := &pb.CreateWalletRequest{
		WalletPath:     localWalletDir,
		Keymanager:     pb.KeymanagerKind_IMPORTED,
		WalletPassword: strongPass,
	}
	// We delete the directory at defaultWalletPath as CreateWallet will return an error if it tries to create a wallet
	// where a directory already exists
	require.NoError(t, os.RemoveAll(defaultWalletPath))
	_, err = s.CreateWallet(ctx, req)
	require.NoError(t, err)

	importReq := &pb.ImportKeystoresRequest{
		KeystoresPassword: strongPass,
		KeystoresImported: []string{"badjson"},
	}
	_, err = s.ImportKeystores(ctx, importReq)
	require.ErrorContains(t, "Not a valid EIP-2335 keystore", err)

	encryptor := keystorev4.New()
	keystores := make([]string, 3)
	for i := 0; i < len(keystores); i++ {
		privKey := bls.RandKey()
		pubKey := fmt.Sprintf("%x", privKey.PublicKey().Marshal())
		id, err := uuid.NewRandom()
		require.NoError(t, err)
		cryptoFields, err := encryptor.Encrypt(privKey.Marshal(), strongPass)
		require.NoError(t, err)
		item := &keymanager.Keystore{
			Crypto:  cryptoFields,
			ID:      id.String(),
			Version: encryptor.Version(),
			Pubkey:  pubKey,
			Name:    encryptor.Name(),
		}
		encodedFile, err := json.MarshalIndent(item, "", "\t")
		require.NoError(t, err)
		keystores[i] = string(encodedFile)
	}
	importReq.KeystoresImported = keystores
	_, err = s.ImportKeystores(ctx, importReq)
	require.NoError(t, err)
}

func TestServer_CreateWallet_Derived(t *testing.T) {
	localWalletDir := setupWalletDir(t)
	defaultWalletPath = localWalletDir
	ctx := context.Background()
	strongPass := "29384283xasjasd32%%&*@*#*"
	s := &Server{
		walletInitializedFeed: new(event.Feed),
	}
	req := &pb.CreateWalletRequest{
		WalletPath:     localWalletDir,
		Keymanager:     pb.KeymanagerKind_DERIVED,
		WalletPassword: strongPass,
		NumAccounts:    0,
	}
	// We delete the directory at defaultWalletPath as CreateWallet will return an error if it tries to create a wallet
	// where a directory already exists
	require.NoError(t, os.RemoveAll(defaultWalletPath))
	_, err := s.CreateWallet(ctx, req)
	require.ErrorContains(t, "Must create at least 1 validator account", err)

	req.NumAccounts = 2
	_, err = s.CreateWallet(ctx, req)
	require.ErrorContains(t, "Must include mnemonic", err)

	mnemonicResp, err := s.GenerateMnemonic(ctx, &ptypes.Empty{})
	require.NoError(t, err)
	req.Mnemonic = mnemonicResp.Mnemonic

	_, err = s.CreateWallet(ctx, req)
	require.NoError(t, err)
}

func TestServer_WalletConfig_NoWalletFound(t *testing.T) {
	s := &Server{}
	resp, err := s.WalletConfig(context.Background(), &ptypes.Empty{})
	require.NoError(t, err)
	assert.DeepEqual(t, resp, &pb.WalletResponse{})
}

func TestServer_WalletConfig(t *testing.T) {
	localWalletDir := setupWalletDir(t)
	defaultWalletPath = localWalletDir
	ctx := context.Background()
	strongPass := "29384283xasjasd32%%&*@*#*"
	s := &Server{
		walletInitializedFeed: new(event.Feed),
		walletDir:             defaultWalletPath,
	}
	// We attempt to create the wallet.
	w, err := accounts.CreateWalletWithKeymanager(ctx, &accounts.CreateWalletConfig{
		WalletCfg: &wallet.Config{
			WalletDir:      defaultWalletPath,
			KeymanagerKind: keymanager.Imported,
			WalletPassword: strongPass,
		},
		SkipMnemonicConfirm: true,
	})
	require.NoError(t, err)
	require.NoError(t, w.SaveHashedPassword(ctx))
	km, err := w.InitializeKeymanager(ctx, true /* skip mnemonic confirm */)
	require.NoError(t, err)
	s.wallet = w
	s.keymanager = km
	resp, err := s.WalletConfig(ctx, &ptypes.Empty{})
	require.NoError(t, err)

	expectedConfig := imported.DefaultKeymanagerOpts()
	enc, err := json.Marshal(expectedConfig)
	require.NoError(t, err)
	var jsonMap map[string]string
	require.NoError(t, json.Unmarshal(enc, &jsonMap))
	assert.DeepEqual(t, resp, &pb.WalletResponse{
		WalletPath:       localWalletDir,
		KeymanagerKind:   pb.KeymanagerKind_IMPORTED,
		KeymanagerConfig: jsonMap,
	})
}

func TestServer_ChangePassword_Preconditions(t *testing.T) {
	localWalletDir := setupWalletDir(t)
	defaultWalletPath = localWalletDir
	ctx := context.Background()
	strongPass := "29384283xasjasd32%%&*@*#*"
	ss := &Server{
		walletDir: defaultWalletPath,
	}
	_, err := ss.ChangePassword(ctx, &pb.ChangePasswordRequest{
		CurrentPassword: strongPass,
		Password:        "",
	})
	assert.ErrorContains(t, noWalletMsg, err)
	// We attempt to create the wallet.
	w, err := accounts.CreateWalletWithKeymanager(ctx, &accounts.CreateWalletConfig{
		WalletCfg: &wallet.Config{
			WalletDir:      defaultWalletPath,
			KeymanagerKind: keymanager.Derived,
			WalletPassword: strongPass,
		},
		SkipMnemonicConfirm: true,
	})
	require.NoError(t, err)
	km, err := w.InitializeKeymanager(ctx, true /* skip mnemonic confirm */)
	require.NoError(t, err)
	ss.wallet = w
	ss.walletInitialized = true
	ss.keymanager = km
	_, err = ss.ChangePassword(ctx, &pb.ChangePasswordRequest{
		CurrentPassword: strongPass,
		Password:        "",
	})
	assert.ErrorContains(t, "Could not validate wallet password", err)
	_, err = ss.ChangePassword(ctx, &pb.ChangePasswordRequest{
		CurrentPassword:      strongPass,
		Password:             "abc",
		PasswordConfirmation: "def",
	})
	assert.ErrorContains(t, "does not match", err)
}

func TestServer_ChangePassword_ImportedKeymanager(t *testing.T) {
	ss, _ := createImportedWalletWithAccounts(t, 1)
	newPassword := "NewPassw0rdz%%%%pass"
	_, err := ss.ChangePassword(context.Background(), &pb.ChangePasswordRequest{
		CurrentPassword:      ss.wallet.Password(),
		Password:             newPassword,
		PasswordConfirmation: newPassword,
	})
	require.NoError(t, err)
	assert.Equal(t, ss.wallet.Password(), newPassword)
}

func TestServer_ChangePassword_DerivedKeymanager(t *testing.T) {
	localWalletDir := setupWalletDir(t)
	defaultWalletPath = localWalletDir
	ctx := context.Background()
	strongPass := "29384283xasjasd32%%&*@*#*"
	// We attempt to create the wallet.
	w, err := accounts.CreateWalletWithKeymanager(ctx, &accounts.CreateWalletConfig{
		WalletCfg: &wallet.Config{
			WalletDir:      defaultWalletPath,
			KeymanagerKind: keymanager.Derived,
			WalletPassword: strongPass,
		},
		SkipMnemonicConfirm: true,
	})
	require.NoError(t, err)
	km, err := w.InitializeKeymanager(ctx, true /* skip mnemonic confirm */)
	require.NoError(t, err)
	ss := &Server{
		walletDir: defaultWalletPath,
	}
	ss.wallet = w
	ss.walletInitialized = true
	ss.keymanager = km
	newPassword := "NewPassw0rdz%%%%pass"
	_, err = ss.ChangePassword(ctx, &pb.ChangePasswordRequest{
		CurrentPassword:      strongPass,
		Password:             newPassword,
		PasswordConfirmation: newPassword,
	})
	require.NoError(t, err)
	assert.Equal(t, w.Password(), newPassword)
}

func TestServer_HasWallet(t *testing.T) {
	localWalletDir := setupWalletDir(t)
	defaultWalletPath = localWalletDir
	ctx := context.Background()
	strongPass := "29384283xasjasd32%%&*@*#*"
	ss := &Server{
		walletDir: defaultWalletPath,
	}
	// First delete the created folder and check the response
	require.NoError(t, os.RemoveAll(defaultWalletPath))
	resp, err := ss.HasWallet(ctx, &ptypes.Empty{})
	require.NoError(t, err)
	assert.DeepEqual(t, &pb.HasWalletResponse{
		WalletExists: false,
	}, resp)

	// We now create the folder but without a valid wallet, i.e. lacking a subdirectory such as 'imported'
	// We expect an empty directory to behave similarly as if there were no directory
	require.NoError(t, os.MkdirAll(defaultWalletPath, os.ModePerm))
	resp, err = ss.HasWallet(ctx, &ptypes.Empty{})
	require.NoError(t, err)
	assert.DeepEqual(t, &pb.HasWalletResponse{
		WalletExists: false,
	}, resp)

	// We attempt to create the wallet.
	_, err = accounts.CreateWalletWithKeymanager(ctx, &accounts.CreateWalletConfig{
		WalletCfg: &wallet.Config{
			WalletDir:      defaultWalletPath,
			KeymanagerKind: keymanager.Imported,
			WalletPassword: strongPass,
		},
		SkipMnemonicConfirm: true,
	})
	require.NoError(t, err)
	resp, err = ss.HasWallet(ctx, &ptypes.Empty{})
	require.NoError(t, err)
	assert.DeepEqual(t, &pb.HasWalletResponse{
		WalletExists: true,
	}, resp)
}

func TestServer_ImportKeystores_FailedPreconditions_WrongKeymanagerKind(t *testing.T) {
	localWalletDir := setupWalletDir(t)
	defaultWalletPath = localWalletDir
	ctx := context.Background()
	strongPass := "29384283xasjasd32%%&*@*#*"
	w, err := accounts.CreateWalletWithKeymanager(ctx, &accounts.CreateWalletConfig{
		WalletCfg: &wallet.Config{
			WalletDir:      defaultWalletPath,
			KeymanagerKind: keymanager.Derived,
			WalletPassword: strongPass,
		},
		SkipMnemonicConfirm: true,
	})
	require.NoError(t, err)
	km, err := w.InitializeKeymanager(ctx, true /* skip mnemonic confirm */)
	require.NoError(t, err)
	ss := &Server{
		wallet:     w,
		keymanager: km,
	}
	_, err = ss.ImportKeystores(ctx, &pb.ImportKeystoresRequest{})
	assert.ErrorContains(t, "Only imported wallets can import more", err)
}

func TestServer_ImportKeystores_FailedPreconditions(t *testing.T) {
	localWalletDir := setupWalletDir(t)
	defaultWalletPath = localWalletDir
	ctx := context.Background()
	strongPass := "29384283xasjasd32%%&*@*#*"
	w, err := accounts.CreateWalletWithKeymanager(ctx, &accounts.CreateWalletConfig{
		WalletCfg: &wallet.Config{
			WalletDir:      defaultWalletPath,
			KeymanagerKind: keymanager.Imported,
			WalletPassword: strongPass,
		},
		SkipMnemonicConfirm: true,
	})
	require.NoError(t, err)
	require.NoError(t, w.SaveHashedPassword(ctx))
	km, err := w.InitializeKeymanager(ctx, true /* skip mnemonic confirm */)
	require.NoError(t, err)
	ss := &Server{
		keymanager: km,
	}
	_, err = ss.ImportKeystores(ctx, &pb.ImportKeystoresRequest{})
	assert.ErrorContains(t, "No wallet initialized", err)
	ss.wallet = w
	_, err = ss.ImportKeystores(ctx, &pb.ImportKeystoresRequest{})
	assert.ErrorContains(t, "Password required for keystores", err)
	_, err = ss.ImportKeystores(ctx, &pb.ImportKeystoresRequest{
		KeystoresPassword: strongPass,
	})
	assert.ErrorContains(t, "No keystores included for import", err)
	_, err = ss.ImportKeystores(ctx, &pb.ImportKeystoresRequest{
		KeystoresPassword: strongPass,
		KeystoresImported: []string{"badjson"},
	})
	assert.ErrorContains(t, "Not a valid EIP-2335 keystore", err)
}

func TestServer_ImportKeystores_OK(t *testing.T) {
	imported.ResetCaches()
	localWalletDir := setupWalletDir(t)
	defaultWalletPath = localWalletDir
	ctx := context.Background()
	strongPass := "29384283xasjasd32%%&*@*#*"
	w, err := accounts.CreateWalletWithKeymanager(ctx, &accounts.CreateWalletConfig{
		WalletCfg: &wallet.Config{
			WalletDir:      defaultWalletPath,
			KeymanagerKind: keymanager.Imported,
			WalletPassword: strongPass,
		},
		SkipMnemonicConfirm: true,
	})
	require.NoError(t, err)
	require.NoError(t, w.SaveHashedPassword(ctx))
	km, err := w.InitializeKeymanager(ctx, true /* skip mnemonic confirm */)
	require.NoError(t, err)
	ss := &Server{
		keymanager:            km,
		wallet:                w,
		walletInitializedFeed: new(event.Feed),
	}

	// Create 3 keystores.
	encryptor := keystorev4.New()
	keystores := make([]string, 3)
	pubKeys := make([][]byte, 3)
	for i := 0; i < len(keystores); i++ {
		privKey := bls.RandKey()
		pubKey := fmt.Sprintf("%x", privKey.PublicKey().Marshal())
		id, err := uuid.NewRandom()
		require.NoError(t, err)
		cryptoFields, err := encryptor.Encrypt(privKey.Marshal(), strongPass)
		require.NoError(t, err)
		item := &keymanager.Keystore{
			Crypto:  cryptoFields,
			ID:      id.String(),
			Version: encryptor.Version(),
			Pubkey:  pubKey,
			Name:    encryptor.Name(),
		}
		encodedFile, err := json.MarshalIndent(item, "", "\t")
		require.NoError(t, err)
		keystores[i] = string(encodedFile)
		pubKeys[i] = privKey.PublicKey().Marshal()
	}

	// Check the wallet has no accounts to start with.
	keys, err := km.FetchValidatingPublicKeys(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, len(keys))

	// Import the 3 keystores and verify the wallet has 3 new accounts.
	res, err := ss.ImportKeystores(ctx, &pb.ImportKeystoresRequest{
		KeystoresPassword: strongPass,
		KeystoresImported: keystores,
	})
	require.NoError(t, err)
	assert.DeepEqual(t, &pb.ImportKeystoresResponse{
		ImportedPublicKeys: pubKeys,
	}, res)

	km, err = w.InitializeKeymanager(ctx, true /* skip mnemonic confirm */)
	require.NoError(t, err)
	keys, err = km.FetchValidatingPublicKeys(ctx)
	require.NoError(t, err)
	assert.Equal(t, 3, len(keys))
}
