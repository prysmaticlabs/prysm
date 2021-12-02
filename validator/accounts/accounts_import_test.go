package accounts

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/crypto/bls"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
	prysmTime "github.com/prysmaticlabs/prysm/time"
	"github.com/prysmaticlabs/prysm/validator/accounts/iface"
	"github.com/prysmaticlabs/prysm/validator/accounts/wallet"
	"github.com/prysmaticlabs/prysm/validator/keymanager"
	"github.com/prysmaticlabs/prysm/validator/keymanager/imported"
	keystorev4 "github.com/wealdtech/go-eth2-wallet-encryptor-keystorev4"
)

func TestImport_Noninteractive(t *testing.T) {
	imported.ResetCaches()
	walletDir, passwordsDir, passwordFilePath := setupWalletAndPasswordsDir(t)
	keysDir := filepath.Join(t.TempDir(), "keysDir")
	require.NoError(t, os.MkdirAll(keysDir, os.ModePerm))

	cliCtx := setupWalletCtx(t, &testWalletConfig{
		walletDir:           walletDir,
		passwordsDir:        passwordsDir,
		keysDir:             keysDir,
		keymanagerKind:      keymanager.Imported,
		walletPasswordFile:  passwordFilePath,
		accountPasswordFile: passwordFilePath,
	})
	w, err := CreateWalletWithKeymanager(cliCtx.Context, &CreateWalletConfig{
		WalletCfg: &wallet.Config{
			WalletDir:      walletDir,
			KeymanagerKind: keymanager.Imported,
			WalletPassword: password,
		},
	})
	require.NoError(t, err)
	keymanager, err := imported.NewKeymanager(
		cliCtx.Context,
		&imported.SetupConfig{
			Wallet:           w,
			ListenForChanges: false,
		},
	)
	require.NoError(t, err)

	// Make sure there are no accounts at the start.
	accounts, err := keymanager.ValidatingAccountNames()
	require.NoError(t, err)
	assert.Equal(t, len(accounts), 0)

	// Create 2 keys.
	createKeystore(t, keysDir)
	time.Sleep(time.Second)
	createKeystore(t, keysDir)

	require.NoError(t, ImportAccountsCli(cliCtx))

	w, err = wallet.OpenWallet(cliCtx.Context, &wallet.Config{
		WalletDir:      walletDir,
		WalletPassword: password,
	})
	require.NoError(t, err)
	km, err := w.InitializeKeymanager(cliCtx.Context, iface.InitKeymanagerConfig{ListenForChanges: false})
	require.NoError(t, err)
	keys, err := km.FetchValidatingPublicKeys(cliCtx.Context)
	require.NoError(t, err)

	assert.Equal(t, 2, len(keys))
}

// TestImport_DuplicateKeys is a regression test that ensures correction function if duplicate keys are being imported
func TestImport_DuplicateKeys(t *testing.T) {
	imported.ResetCaches()
	walletDir, passwordsDir, passwordFilePath := setupWalletAndPasswordsDir(t)
	keysDir := filepath.Join(t.TempDir(), "keysDir")
	require.NoError(t, os.MkdirAll(keysDir, os.ModePerm))

	cliCtx := setupWalletCtx(t, &testWalletConfig{
		walletDir:           walletDir,
		passwordsDir:        passwordsDir,
		keysDir:             keysDir,
		keymanagerKind:      keymanager.Imported,
		walletPasswordFile:  passwordFilePath,
		accountPasswordFile: passwordFilePath,
	})
	w, err := CreateWalletWithKeymanager(cliCtx.Context, &CreateWalletConfig{
		WalletCfg: &wallet.Config{
			WalletDir:      walletDir,
			KeymanagerKind: keymanager.Imported,
			WalletPassword: password,
		},
	})
	require.NoError(t, err)

	// Create a key and then copy it to create a duplicate
	_, keystorePath := createKeystore(t, keysDir)
	time.Sleep(time.Second)
	input, err := ioutil.ReadFile(keystorePath)
	require.NoError(t, err)
	keystorePath2 := filepath.Join(keysDir, "copyOfKeystore.json")
	err = ioutil.WriteFile(keystorePath2, input, os.ModePerm)
	require.NoError(t, err)

	require.NoError(t, ImportAccountsCli(cliCtx))

	_, err = wallet.OpenWallet(cliCtx.Context, &wallet.Config{
		WalletDir:      walletDir,
		WalletPassword: password,
	})
	require.NoError(t, err)
	km, err := w.InitializeKeymanager(cliCtx.Context, iface.InitKeymanagerConfig{ListenForChanges: false})
	require.NoError(t, err)
	keys, err := km.FetchValidatingPublicKeys(cliCtx.Context)
	require.NoError(t, err)

	// There should only be 1 account as the duplicate keystore was ignored
	assert.Equal(t, 1, len(keys))
}

func TestImport_Noninteractive_RandomName(t *testing.T) {
	imported.ResetCaches()
	walletDir, passwordsDir, passwordFilePath := setupWalletAndPasswordsDir(t)
	keysDir := filepath.Join(t.TempDir(), "keysDir")
	require.NoError(t, os.MkdirAll(keysDir, os.ModePerm))

	cliCtx := setupWalletCtx(t, &testWalletConfig{
		walletDir:           walletDir,
		passwordsDir:        passwordsDir,
		keysDir:             keysDir,
		keymanagerKind:      keymanager.Imported,
		walletPasswordFile:  passwordFilePath,
		accountPasswordFile: passwordFilePath,
	})
	w, err := CreateWalletWithKeymanager(cliCtx.Context, &CreateWalletConfig{
		WalletCfg: &wallet.Config{
			WalletDir:      walletDir,
			KeymanagerKind: keymanager.Imported,
			WalletPassword: password,
		},
	})
	require.NoError(t, err)
	keymanager, err := imported.NewKeymanager(
		cliCtx.Context,
		&imported.SetupConfig{
			Wallet:           w,
			ListenForChanges: false,
		},
	)
	require.NoError(t, err)

	// Make sure there are no accounts at the start.
	accounts, err := keymanager.ValidatingAccountNames()
	require.NoError(t, err)
	assert.Equal(t, len(accounts), 0)

	// Create 2 keys.
	createRandomNameKeystore(t, keysDir)
	time.Sleep(time.Second)
	createRandomNameKeystore(t, keysDir)

	require.NoError(t, ImportAccountsCli(cliCtx))

	w, err = wallet.OpenWallet(cliCtx.Context, &wallet.Config{
		WalletDir:      walletDir,
		WalletPassword: password,
	})
	require.NoError(t, err)
	km, err := w.InitializeKeymanager(cliCtx.Context, iface.InitKeymanagerConfig{ListenForChanges: false})
	require.NoError(t, err)
	keys, err := km.FetchValidatingPublicKeys(cliCtx.Context)
	require.NoError(t, err)

	assert.Equal(t, 2, len(keys))
}

func TestImport_Noninteractive_Filepath(t *testing.T) {
	imported.ResetCaches()
	walletDir, passwordsDir, passwordFilePath := setupWalletAndPasswordsDir(t)
	keysDir := filepath.Join(t.TempDir(), "keysDir")
	require.NoError(t, os.MkdirAll(keysDir, os.ModePerm))

	_, keystorePath := createKeystore(t, keysDir)
	cliCtx := setupWalletCtx(t, &testWalletConfig{
		walletDir:           walletDir,
		passwordsDir:        passwordsDir,
		keysDir:             keystorePath,
		keymanagerKind:      keymanager.Imported,
		walletPasswordFile:  passwordFilePath,
		accountPasswordFile: passwordFilePath,
	})
	w, err := CreateWalletWithKeymanager(cliCtx.Context, &CreateWalletConfig{
		WalletCfg: &wallet.Config{
			WalletDir:      walletDir,
			KeymanagerKind: keymanager.Imported,
			WalletPassword: password,
		},
	})
	require.NoError(t, err)
	keymanager, err := imported.NewKeymanager(
		cliCtx.Context,
		&imported.SetupConfig{
			Wallet:           w,
			ListenForChanges: false,
		},
	)
	require.NoError(t, err)

	// Make sure there are no accounts at the start.
	accounts, err := keymanager.ValidatingAccountNames()
	require.NoError(t, err)
	assert.Equal(t, len(accounts), 0)

	require.NoError(t, ImportAccountsCli(cliCtx))

	w, err = wallet.OpenWallet(cliCtx.Context, &wallet.Config{
		WalletDir:      walletDir,
		WalletPassword: password,
	})
	require.NoError(t, err)
	km, err := w.InitializeKeymanager(cliCtx.Context, iface.InitKeymanagerConfig{ListenForChanges: false})
	require.NoError(t, err)
	keys, err := km.FetchValidatingPublicKeys(cliCtx.Context)
	require.NoError(t, err)

	assert.Equal(t, 1, len(keys))
}

func TestImport_SortByDerivationPath(t *testing.T) {
	imported.ResetCaches()
	type test struct {
		name  string
		input []string
		want  []string
	}
	tests := []test{
		{
			name: "Basic sort",
			input: []string{
				"keystore_m_12381_3600_2_0_0.json",
				"keystore_m_12381_3600_1_0_0.json",
				"keystore_m_12381_3600_0_0_0.json",
			},
			want: []string{
				"keystore_m_12381_3600_0_0_0.json",
				"keystore_m_12381_3600_1_0_0.json",
				"keystore_m_12381_3600_2_0_0.json",
			},
		},
		{
			name: "Large digit accounts",
			input: []string{
				"keystore_m_12381_3600_30020330_0_0.json",
				"keystore_m_12381_3600_430490934_0_0.json",
				"keystore_m_12381_3600_0_0_0.json",
				"keystore_m_12381_3600_333_0_0.json",
			},
			want: []string{
				"keystore_m_12381_3600_0_0_0.json",
				"keystore_m_12381_3600_333_0_0.json",
				"keystore_m_12381_3600_30020330_0_0.json",
				"keystore_m_12381_3600_430490934_0_0.json",
			},
		},
		{
			name: "Some filenames with derivation path, others without",
			input: []string{
				"keystore_m_12381_3600_4_0_0.json",
				"keystore.json",
				"keystore-2309023.json",
				"keystore_m_12381_3600_1_0_0.json",
				"keystore_m_12381_3600_3_0_0.json",
			},
			want: []string{
				"keystore_m_12381_3600_1_0_0.json",
				"keystore_m_12381_3600_3_0_0.json",
				"keystore_m_12381_3600_4_0_0.json",
				"keystore.json",
				"keystore-2309023.json",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sort.Sort(byDerivationPath(tt.input))
			assert.DeepEqual(t, tt.want, tt.input)
		})
	}
}

func Test_importPrivateKeyAsAccount(t *testing.T) {
	walletDir, _, passwordFilePath := setupWalletAndPasswordsDir(t)
	privKeyDir := filepath.Join(t.TempDir(), "privKeys")
	require.NoError(t, os.MkdirAll(privKeyDir, os.ModePerm))
	privKeyFileName := filepath.Join(privKeyDir, "privatekey.txt")

	// We create a new private key and save it to a file on disk.
	privKey, err := bls.RandKey()
	require.NoError(t, err)
	privKeyHex := fmt.Sprintf("%x", privKey.Marshal())
	require.NoError(
		t,
		ioutil.WriteFile(privKeyFileName, []byte(privKeyHex), params.BeaconIoConfig().ReadWritePermissions),
	)

	// We instantiate a new wallet from a cli context.
	cliCtx := setupWalletCtx(t, &testWalletConfig{
		walletDir:          walletDir,
		keymanagerKind:     keymanager.Imported,
		walletPasswordFile: passwordFilePath,
		privateKeyFile:     privKeyFileName,
	})
	walletPass := "Passwordz0320$"
	wallet, err := CreateWalletWithKeymanager(cliCtx.Context, &CreateWalletConfig{
		WalletCfg: &wallet.Config{
			WalletDir:      walletDir,
			KeymanagerKind: keymanager.Imported,
			WalletPassword: walletPass,
		},
	})
	require.NoError(t, err)
	keymanager, err := imported.NewKeymanager(
		cliCtx.Context,
		&imported.SetupConfig{
			Wallet:           wallet,
			ListenForChanges: false,
		},
	)
	require.NoError(t, err)
	assert.NoError(t, importPrivateKeyAsAccount(cliCtx, wallet, keymanager))

	// We re-instantiate the keymanager and check we now have 1 public key.
	keymanager, err = imported.NewKeymanager(
		cliCtx.Context,
		&imported.SetupConfig{
			Wallet:           wallet,
			ListenForChanges: false,
		},
	)
	require.NoError(t, err)
	pubKeys, err := keymanager.FetchValidatingPublicKeys(cliCtx.Context)
	require.NoError(t, err)
	require.Equal(t, 1, len(pubKeys))
	assert.DeepEqual(t, pubKeys[0], bytesutil.ToBytes48(privKey.PublicKey().Marshal()))
}

// Returns the fullPath to the newly created keystore file.
func createKeystore(t *testing.T, path string) (*keymanager.Keystore, string) {
	validatingKey, err := bls.RandKey()
	require.NoError(t, err)
	encryptor := keystorev4.New()
	cryptoFields, err := encryptor.Encrypt(validatingKey.Marshal(), password)
	require.NoError(t, err)
	id, err := uuid.NewRandom()
	require.NoError(t, err)
	keystoreFile := &keymanager.Keystore{
		Crypto:  cryptoFields,
		ID:      id.String(),
		Pubkey:  fmt.Sprintf("%x", validatingKey.PublicKey().Marshal()),
		Version: encryptor.Version(),
		Name:    encryptor.Name(),
	}
	encoded, err := json.MarshalIndent(keystoreFile, "", "\t")
	require.NoError(t, err)
	// Write the encoded keystore to disk with the timestamp appended
	createdAt := prysmTime.Now().Unix()
	fullPath := filepath.Join(path, fmt.Sprintf(imported.KeystoreFileNameFormat, createdAt))
	require.NoError(t, ioutil.WriteFile(fullPath, encoded, os.ModePerm))
	return keystoreFile, fullPath
}

// Returns the fullPath to the newly created keystore file.
func createRandomNameKeystore(t *testing.T, path string) (*keymanager.Keystore, string) {
	validatingKey, err := bls.RandKey()
	require.NoError(t, err)
	encryptor := keystorev4.New()
	cryptoFields, err := encryptor.Encrypt(validatingKey.Marshal(), password)
	require.NoError(t, err)
	id, err := uuid.NewRandom()
	require.NoError(t, err)
	keystoreFile := &keymanager.Keystore{
		Crypto:  cryptoFields,
		ID:      id.String(),
		Pubkey:  fmt.Sprintf("%x", validatingKey.PublicKey().Marshal()),
		Version: encryptor.Version(),
		Name:    encryptor.Name(),
	}
	encoded, err := json.MarshalIndent(keystoreFile, "", "\t")
	require.NoError(t, err)
	// Write the encoded keystore to disk with the timestamp appended
	random, err := rand.Int(rand.Reader, big.NewInt(1000000))
	require.NoError(t, err)
	fullPath := filepath.Join(path, fmt.Sprintf("test-%d-keystore", random.Int64()))
	require.NoError(t, ioutil.WriteFile(fullPath, encoded, os.ModePerm))
	return keystoreFile, fullPath
}
