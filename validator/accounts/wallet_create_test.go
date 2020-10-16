package accounts

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/prysmaticlabs/prysm/validator/accounts/wallet"
	"github.com/prysmaticlabs/prysm/validator/flags"
	"github.com/prysmaticlabs/prysm/validator/keymanager"
	"github.com/prysmaticlabs/prysm/validator/keymanager/derived"
	"github.com/prysmaticlabs/prysm/validator/keymanager/imported"
	"github.com/prysmaticlabs/prysm/validator/keymanager/remote"
	"github.com/sirupsen/logrus"
	logTest "github.com/sirupsen/logrus/hooks/test"
	"github.com/urfave/cli/v2"
	keystorev4 "github.com/wealdtech/go-eth2-wallet-encryptor-keystorev4"
)

const (
	walletDirName    = "wallet"
	passwordFileName = "password.txt"
	password         = "OhWOWthisisatest42!$"
	mnemonicFileName = "mnemonic.txt"
	mnemonic         = "garage car helmet trade salmon embrace market giant movie wet same champion dawn chair shield drill amazing panther accident puzzle garden mosquito kind arena"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(ioutil.Discard)
}

type testWalletConfig struct {
	walletDir               string
	passwordsDir            string
	backupDir               string
	keysDir                 string
	deletePublicKeys        string
	voluntaryExitPublicKeys string
	backupPublicKeys        string
	backupPasswordFile      string
	walletPasswordFile      string
	accountPasswordFile     string
	privateKeyFile          string
	skipDepositConfirm      bool
	numAccounts             int64
	keymanagerKind          keymanager.Kind
}

func setupWalletCtx(
	tb testing.TB,
	cfg *testWalletConfig,
) *cli.Context {
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	set.String(flags.WalletDirFlag.Name, cfg.walletDir, "")
	set.String(flags.KeysDirFlag.Name, cfg.keysDir, "")
	set.String(flags.KeymanagerKindFlag.Name, cfg.keymanagerKind.String(), "")
	set.String(flags.DeletePublicKeysFlag.Name, cfg.deletePublicKeys, "")
	set.String(flags.VoluntaryExitPublicKeysFlag.Name, cfg.voluntaryExitPublicKeys, "")
	set.String(flags.BackupDirFlag.Name, cfg.backupDir, "")
	set.String(flags.BackupPasswordFile.Name, cfg.backupPasswordFile, "")
	set.String(flags.BackupPublicKeysFlag.Name, cfg.backupPublicKeys, "")
	set.String(flags.WalletPasswordFileFlag.Name, cfg.walletPasswordFile, "")
	set.String(flags.AccountPasswordFileFlag.Name, cfg.accountPasswordFile, "")
	set.Int64(flags.NumAccountsFlag.Name, cfg.numAccounts, "")
	set.Bool(flags.SkipDepositConfirmationFlag.Name, cfg.skipDepositConfirm, "")

	if cfg.privateKeyFile != "" {
		set.String(flags.ImportPrivateKeyFileFlag.Name, cfg.privateKeyFile, "")
		assert.NoError(tb, set.Set(flags.ImportPrivateKeyFileFlag.Name, cfg.privateKeyFile))
	}
	assert.NoError(tb, set.Set(flags.WalletDirFlag.Name, cfg.walletDir))
	assert.NoError(tb, set.Set(flags.KeysDirFlag.Name, cfg.keysDir))
	assert.NoError(tb, set.Set(flags.KeymanagerKindFlag.Name, cfg.keymanagerKind.String()))
	assert.NoError(tb, set.Set(flags.DeletePublicKeysFlag.Name, cfg.deletePublicKeys))
	assert.NoError(tb, set.Set(flags.VoluntaryExitPublicKeysFlag.Name, cfg.voluntaryExitPublicKeys))
	assert.NoError(tb, set.Set(flags.BackupDirFlag.Name, cfg.backupDir))
	assert.NoError(tb, set.Set(flags.BackupPublicKeysFlag.Name, cfg.backupPublicKeys))
	assert.NoError(tb, set.Set(flags.BackupPasswordFile.Name, cfg.backupPasswordFile))
	assert.NoError(tb, set.Set(flags.WalletPasswordFileFlag.Name, cfg.walletPasswordFile))
	assert.NoError(tb, set.Set(flags.AccountPasswordFileFlag.Name, cfg.accountPasswordFile))
	assert.NoError(tb, set.Set(flags.NumAccountsFlag.Name, strconv.Itoa(int(cfg.numAccounts))))
	assert.NoError(tb, set.Set(flags.SkipDepositConfirmationFlag.Name, strconv.FormatBool(cfg.skipDepositConfirm)))
	return cli.NewContext(&app, set, nil)
}

func setupWalletAndPasswordsDir(t testing.TB) (string, string, string) {
	randPath, err := rand.Int(rand.Reader, big.NewInt(1000000))
	require.NoError(t, err, "Could not generate random file path")
	walletDir := filepath.Join(testutil.TempDir(), fmt.Sprintf("/%d", randPath), "wallet")
	require.NoError(t, os.RemoveAll(walletDir), "Failed to remove directory")
	passwordsDir := filepath.Join(testutil.TempDir(), fmt.Sprintf("/%d", randPath), "passwords")
	require.NoError(t, os.RemoveAll(passwordsDir), "Failed to remove directory")
	passwordFileDir := filepath.Join(testutil.TempDir(), fmt.Sprintf("/%d", randPath), "passwordFile")
	require.NoError(t, os.MkdirAll(passwordFileDir, os.ModePerm))
	passwordFilePath := filepath.Join(passwordFileDir, passwordFileName)
	require.NoError(t, ioutil.WriteFile(passwordFilePath, []byte(password), os.ModePerm))
	t.Cleanup(func() {
		require.NoError(t, os.RemoveAll(walletDir), "Failed to remove directory")
		require.NoError(t, os.RemoveAll(passwordFileDir), "Failed to remove directory")
		require.NoError(t, os.RemoveAll(passwordsDir), "Failed to remove directory")
	})
	return walletDir, passwordsDir, passwordFilePath
}

func TestCreateOrOpenWallet(t *testing.T) {
	hook := logTest.NewGlobal()
	walletDir, passwordsDir, walletPasswordFile := setupWalletAndPasswordsDir(t)
	cliCtx := setupWalletCtx(t, &testWalletConfig{
		walletDir:          walletDir,
		passwordsDir:       passwordsDir,
		keymanagerKind:     keymanager.Imported,
		walletPasswordFile: walletPasswordFile,
	})
	createImportedWallet := func(cliCtx *cli.Context) (*wallet.Wallet, error) {
		cfg, err := extractWalletCreationConfigFromCli(cliCtx, keymanager.Imported)
		if err != nil {
			return nil, err
		}
		w := wallet.New(&wallet.Config{
			KeymanagerKind: cfg.WalletCfg.KeymanagerKind,
			WalletDir:      cfg.WalletCfg.WalletDir,
			WalletPassword: cfg.WalletCfg.WalletPassword,
		})
		if err = createImportedKeymanagerWallet(cliCtx.Context, w); err != nil {
			return nil, errors.Wrap(err, "could not create keymanager")
		}
		log.WithField("wallet-path", cfg.WalletCfg.WalletDir).Info(
			"Successfully created new wallet",
		)
		return w, nil
	}
	createdWallet, err := wallet.OpenWalletOrElseCli(cliCtx, createImportedWallet)
	require.NoError(t, err)
	require.LogsContain(t, hook, "Successfully created new wallet")

	openedWallet, err := wallet.OpenWalletOrElseCli(cliCtx, createImportedWallet)
	require.NoError(t, err)
	assert.Equal(t, createdWallet.KeymanagerKind(), openedWallet.KeymanagerKind())
	assert.Equal(t, createdWallet.AccountsDir(), openedWallet.AccountsDir())
}

func TestCreateWallet_Imported(t *testing.T) {
	walletDir, passwordsDir, walletPasswordFile := setupWalletAndPasswordsDir(t)
	cliCtx := setupWalletCtx(t, &testWalletConfig{
		walletDir:          walletDir,
		passwordsDir:       passwordsDir,
		keymanagerKind:     keymanager.Imported,
		walletPasswordFile: walletPasswordFile,
	})

	// We attempt to create the wallet.
	_, err := CreateAndSaveWalletCli(cliCtx)
	require.NoError(t, err)

	// We attempt to open the newly created wallet.
	w, err := wallet.OpenWallet(cliCtx.Context, &wallet.Config{
		WalletDir: walletDir,
	})
	assert.NoError(t, err)

	// We read the keymanager config for the newly created wallet.
	encoded, err := w.ReadKeymanagerConfigFromDisk(cliCtx.Context)
	assert.NoError(t, err)
	cfg, err := imported.UnmarshalOptionsFile(encoded)
	assert.NoError(t, err)

	// We assert the created configuration was as desired.
	wantedCfg := imported.DefaultKeymanagerOpts()
	assert.DeepEqual(t, wantedCfg, cfg)
}

func TestCreateWallet_Derived(t *testing.T) {
	walletDir, passwordsDir, passwordFile := setupWalletAndPasswordsDir(t)
	cliCtx := setupWalletCtx(t, &testWalletConfig{
		walletDir:          walletDir,
		passwordsDir:       passwordsDir,
		walletPasswordFile: passwordFile,
		keymanagerKind:     keymanager.Derived,
	})

	// We attempt to create the wallet.
	_, err := CreateAndSaveWalletCli(cliCtx)
	require.NoError(t, err)

	// We attempt to open the newly created wallet.
	ctx := context.Background()
	w, err := wallet.OpenWallet(cliCtx.Context, &wallet.Config{
		WalletDir: walletDir,
	})
	assert.NoError(t, err)

	// We read the keymanager config for the newly created wallet.
	encoded, err := w.ReadKeymanagerConfigFromDisk(ctx)
	assert.NoError(t, err)
	cfg, err := derived.UnmarshalOptionsFile(encoded)
	assert.NoError(t, err)

	// We assert the created configuration was as desired.
	assert.DeepEqual(t, derived.DefaultKeymanagerOpts(), cfg)
}

// TestCreateWallet_WalletAlreadyExists checks for expected error if trying to create a wallet when there is one already.
func TestCreateWallet_WalletAlreadyExists(t *testing.T) {
	walletDir, passwordsDir, passwordFile := setupWalletAndPasswordsDir(t)
	cliCtx := setupWalletCtx(t, &testWalletConfig{
		walletDir:          walletDir,
		passwordsDir:       passwordsDir,
		walletPasswordFile: passwordFile,
		keymanagerKind:     keymanager.Derived,
	})

	// We attempt to create the wallet.
	_, err := CreateAndSaveWalletCli(cliCtx)
	require.NoError(t, err)

	// We attempt to create another wallet of the same type at the same location. We expect an error.
	_, err = CreateAndSaveWalletCli(cliCtx)
	require.ErrorContains(t, "already exists", err)

	cliCtx = setupWalletCtx(t, &testWalletConfig{
		walletDir:          walletDir,
		passwordsDir:       passwordsDir,
		walletPasswordFile: passwordFile,
		keymanagerKind:     keymanager.Imported,
	})

	// We attempt to create another wallet of different type at the same location. We expect an error.
	_, err = CreateAndSaveWalletCli(cliCtx)
	require.ErrorContains(t, "already exists", err)
}

// TestCorrectPassphrase_Derived makes sure the wallet created uses the provided passphrase
func TestCorrectPassphrase_Derived(t *testing.T) {
	walletDir, _, passwordFile := setupWalletAndPasswordsDir(t)

	//Specify the password locally to this file for convenience.
	password := "Pa$sW0rD0__Fo0xPr"
	require.NoError(t, ioutil.WriteFile(passwordFile, []byte(password), os.ModePerm))

	cliCtx := setupWalletCtx(t, &testWalletConfig{
		walletDir:          walletDir,
		walletPasswordFile: passwordFile,
		keymanagerKind:     keymanager.Derived,
		skipDepositConfirm: true,
	})

	// We attempt to create the wallet.
	_, err := CreateAndSaveWalletCli(cliCtx)
	require.Equal(t, nil, err, "error in CreateAndSaveWalletCli()")

	w := wallet.New(&wallet.Config{
		WalletDir:      walletDir,
		KeymanagerKind: keymanager.Derived,
	})

	seedConfigFile, err := w.ReadEncryptedSeedFromDisk(cliCtx.Context)
	require.Equal(t, nil, err, "could not read encrypted seed file from disk")
	defer func() {
		err := seedConfigFile.Close()
		require.Equal(t, nil, err, "Could not close encrypted seed file")
	}()
	encodedSeedFile, err := ioutil.ReadAll(seedConfigFile)
	require.Equal(t, nil, err, "could not read seed configuration file contents")

	seedConfig := &derived.SeedConfig{}
	err = json.Unmarshal(encodedSeedFile, seedConfig)
	require.Equal(t, nil, err, "could not unmarshal seed configuration")

	decryptor := keystorev4.New()
	_, err = decryptor.Decrypt(seedConfig.Crypto, password)
	require.Equal(t, nil, err, "could not decrypt seed configuration with password")
}

func TestCreateWallet_Remote(t *testing.T) {
	walletDir, _, walletPasswordFile := setupWalletAndPasswordsDir(t)
	wantCfg := &remote.KeymanagerOpts{
		RemoteCertificate: &remote.CertificateConfig{
			ClientCertPath: "/tmp/client.crt",
			ClientKeyPath:  "/tmp/client.key",
			CACertPath:     "/tmp/ca.crt",
		},
		RemoteAddr: "host.example.com:4000",
	}
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	keymanagerKind := "remote"
	set.String(flags.WalletDirFlag.Name, walletDir, "")
	set.String(flags.WalletPasswordFileFlag.Name, walletDir, "")
	set.String(flags.KeymanagerKindFlag.Name, keymanagerKind, "")
	set.String(flags.GrpcRemoteAddressFlag.Name, wantCfg.RemoteAddr, "")
	set.String(flags.RemoteSignerCertPathFlag.Name, wantCfg.RemoteCertificate.ClientCertPath, "")
	set.String(flags.RemoteSignerKeyPathFlag.Name, wantCfg.RemoteCertificate.ClientKeyPath, "")
	set.String(flags.RemoteSignerCACertPathFlag.Name, wantCfg.RemoteCertificate.CACertPath, "")
	assert.NoError(t, set.Set(flags.WalletDirFlag.Name, walletDir))
	assert.NoError(t, set.Set(flags.WalletPasswordFileFlag.Name, walletPasswordFile))
	assert.NoError(t, set.Set(flags.KeymanagerKindFlag.Name, keymanagerKind))
	assert.NoError(t, set.Set(flags.GrpcRemoteAddressFlag.Name, wantCfg.RemoteAddr))
	assert.NoError(t, set.Set(flags.RemoteSignerCertPathFlag.Name, wantCfg.RemoteCertificate.ClientCertPath))
	assert.NoError(t, set.Set(flags.RemoteSignerKeyPathFlag.Name, wantCfg.RemoteCertificate.ClientKeyPath))
	assert.NoError(t, set.Set(flags.RemoteSignerCACertPathFlag.Name, wantCfg.RemoteCertificate.CACertPath))
	cliCtx := cli.NewContext(&app, set, nil)

	// We attempt to create the wallet.
	_, err := CreateAndSaveWalletCli(cliCtx)
	require.NoError(t, err)

	// We attempt to open the newly created wallet.
	ctx := context.Background()
	w, err := wallet.OpenWallet(cliCtx.Context, &wallet.Config{
		WalletDir: walletDir,
	})
	assert.NoError(t, err)

	// We read the keymanager config for the newly created wallet.
	encoded, err := w.ReadKeymanagerConfigFromDisk(ctx)
	assert.NoError(t, err)
	cfg, err := remote.UnmarshalOptionsFile(encoded)
	assert.NoError(t, err)

	// We assert the created configuration was as desired.
	assert.DeepEqual(t, wantCfg, cfg)
}
