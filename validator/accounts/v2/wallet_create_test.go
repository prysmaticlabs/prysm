package v2

import (
	"context"
	"encoding/json"
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/prysmaticlabs/prysm/validator/flags"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/derived"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/direct"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/remote"
	logTest "github.com/sirupsen/logrus/hooks/test"
	"github.com/urfave/cli/v2"
	keystorev4 "github.com/wealdtech/go-eth2-wallet-encryptor-keystorev4"
)

func TestCreateOrOpenWallet(t *testing.T) {
	hook := logTest.NewGlobal()
	walletDir, passwordsDir, walletPasswordFile := setupWalletAndPasswordsDir(t)
	cliCtx := setupWalletCtx(t, &testWalletConfig{
		walletDir:          walletDir,
		passwordsDir:       passwordsDir,
		keymanagerKind:     v2keymanager.Direct,
		walletPasswordFile: walletPasswordFile,
	})
	createDirectWallet := func(cliCtx *cli.Context) (*Wallet, error) {
		cfg, err := extractWalletCreationConfigFromCli(cliCtx, v2keymanager.Direct)
		if err != nil {
			return nil, err
		}
		accountsPath := filepath.Join(cfg.WalletCfg.WalletDir, cfg.WalletCfg.KeymanagerKind.String())
		w := &Wallet{
			accountsPath:   accountsPath,
			keymanagerKind: cfg.WalletCfg.KeymanagerKind,
			walletDir:      cfg.WalletCfg.WalletDir,
			walletPassword: cfg.WalletCfg.WalletPassword,
		}
		if err = createDirectKeymanagerWallet(cliCtx.Context, w); err != nil {
			return nil, errors.Wrap(err, "could not create keymanager")
		}
		log.WithField("wallet-path", w.walletDir).Info(
			"Successfully created new wallet",
		)
		return w, nil
	}
	createdWallet, err := OpenWalletOrElseCli(cliCtx, createDirectWallet)
	require.NoError(t, err)
	require.LogsContain(t, hook, "Successfully created new wallet")

	openedWallet, err := OpenWalletOrElseCli(cliCtx, createDirectWallet)
	require.NoError(t, err)
	assert.Equal(t, createdWallet.KeymanagerKind(), openedWallet.KeymanagerKind())
	assert.Equal(t, createdWallet.AccountsDir(), openedWallet.AccountsDir())
}

func TestCreateWallet_Direct(t *testing.T) {
	walletDir, passwordsDir, walletPasswordFile := setupWalletAndPasswordsDir(t)
	cliCtx := setupWalletCtx(t, &testWalletConfig{
		walletDir:          walletDir,
		passwordsDir:       passwordsDir,
		keymanagerKind:     v2keymanager.Direct,
		walletPasswordFile: walletPasswordFile,
	})

	// We attempt to create the wallet.
	_, err := CreateAndSaveWalletCli(cliCtx)
	require.NoError(t, err)

	// We attempt to open the newly created wallet.
	wallet, err := OpenWallet(cliCtx.Context, &WalletConfig{
		WalletDir: walletDir,
	})
	assert.NoError(t, err)

	// We read the keymanager config for the newly created wallet.
	encoded, err := wallet.ReadKeymanagerConfigFromDisk(cliCtx.Context)
	assert.NoError(t, err)
	cfg, err := direct.UnmarshalOptionsFile(encoded)
	assert.NoError(t, err)

	// We assert the created configuration was as desired.
	wantedCfg := direct.DefaultKeymanagerOpts()
	assert.DeepEqual(t, wantedCfg, cfg)
}

func TestCreateWallet_Derived(t *testing.T) {
	walletDir, passwordsDir, passwordFile := setupWalletAndPasswordsDir(t)
	cliCtx := setupWalletCtx(t, &testWalletConfig{
		walletDir:          walletDir,
		passwordsDir:       passwordsDir,
		walletPasswordFile: passwordFile,
		keymanagerKind:     v2keymanager.Derived,
	})

	// We attempt to create the wallet.
	_, err := CreateAndSaveWalletCli(cliCtx)
	require.NoError(t, err)

	// We attempt to open the newly created wallet.
	ctx := context.Background()
	wallet, err := OpenWallet(cliCtx.Context, &WalletConfig{
		WalletDir: walletDir,
	})
	assert.NoError(t, err)

	// We read the keymanager config for the newly created wallet.
	encoded, err := wallet.ReadKeymanagerConfigFromDisk(ctx)
	assert.NoError(t, err)
	cfg, err := derived.UnmarshalOptionsFile(encoded)
	assert.NoError(t, err)

	// We assert the created configuration was as desired.
	assert.DeepEqual(t, derived.DefaultKeymanagerOpts(), cfg)
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
		keymanagerKind:     v2keymanager.Derived,
	})

	// We attempt to create the wallet.
	_, err := CreateAndSaveWalletCli(cliCtx)
	require.Equal(t, nil, err, "error in CreateAndSaveWalletCli()")

	keymanagerKind, err := readKeymanagerKindFromWalletPath(walletDir)
	require.Equal(t, nil, err, "could not read keymanager kind for wallet")

	walletPath := filepath.Join(walletDir, keymanagerKind.String())
	wallet := &Wallet{
		walletDir:      walletDir,
		accountsPath:   walletPath,
		keymanagerKind: keymanagerKind,
	}

	seedConfigFile, err := wallet.ReadEncryptedSeedFromDisk(cliCtx.Context)
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
	wallet, err := OpenWallet(cliCtx.Context, &WalletConfig{
		WalletDir: walletDir,
	})
	assert.NoError(t, err)

	// We read the keymanager config for the newly created wallet.
	encoded, err := wallet.ReadKeymanagerConfigFromDisk(ctx)
	assert.NoError(t, err)
	cfg, err := remote.UnmarshalOptionsFile(encoded)
	assert.NoError(t, err)

	// We assert the created configuration was as desired.
	assert.DeepEqual(t, wantCfg, cfg)
}
