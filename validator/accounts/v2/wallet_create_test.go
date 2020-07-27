package v2

import (
	"context"
	"flag"
	"os"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/prysmaticlabs/prysm/validator/flags"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/derived"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/direct"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/remote"
	"github.com/urfave/cli/v2"
)

func TestCreateWallet_Direct(t *testing.T) {
	walletDir, passwordsDir, _ := setupWalletAndPasswordsDir(t)
	cliCtx := setupWalletCtx(t, &testWalletConfig{
		walletDir:      walletDir,
		passwordsDir:   passwordsDir,
		keymanagerKind: v2keymanager.Direct,
	})

	// We attempt to create the wallet.
	require.NoError(t, CreateWallet(cliCtx))

	// We attempt to open the newly created wallet.
	ctx := context.Background()
	wallet, err := OpenWallet(cliCtx)
	assert.NoError(t, err)

	// We read the keymanager config for the newly created wallet.
	encoded, err := wallet.ReadKeymanagerConfigFromDisk(ctx)
	assert.NoError(t, err)
	cfg, err := direct.UnmarshalConfigFile(encoded)
	assert.NoError(t, err)

	// We assert the created configuration was as desired.
	assert.DeepEqual(t, direct.DefaultConfig(), cfg)
}

func TestCreateWallet_Derived(t *testing.T) {
	walletDir, passwordsDir, passwordFile := setupWalletAndPasswordsDir(t)
	cliCtx := setupWalletCtx(t, &testWalletConfig{
		walletDir:      walletDir,
		passwordsDir:   passwordsDir,
		passwordFile:   passwordFile,
		keymanagerKind: v2keymanager.Derived,
	})

	// We attempt to create the wallet.
	require.NoError(t, CreateWallet(cliCtx))

	// We attempt to open the newly created wallet.
	ctx := context.Background()
	wallet, err := OpenWallet(cliCtx)
	assert.NoError(t, err)

	// We read the keymanager config for the newly created wallet.
	encoded, err := wallet.ReadKeymanagerConfigFromDisk(ctx)
	assert.NoError(t, err)
	cfg, err := derived.UnmarshalConfigFile(encoded)
	assert.NoError(t, err)

	// We assert the created configuration was as desired.
	assert.DeepEqual(t, derived.DefaultConfig(), cfg)
}

func TestCreateWallet_Remote(t *testing.T) {
	walletDir := testutil.TempDir() + "/wallet"
	defer func() {
		assert.NoError(t, os.RemoveAll(walletDir))
	}()
	wantCfg := &remote.Config{
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
	set.String(flags.KeymanagerKindFlag.Name, keymanagerKind, "")
	set.String(flags.GrpcRemoteAddressFlag.Name, wantCfg.RemoteAddr, "")
	set.String(flags.RemoteSignerCertPathFlag.Name, wantCfg.RemoteCertificate.ClientCertPath, "")
	set.String(flags.RemoteSignerKeyPathFlag.Name, wantCfg.RemoteCertificate.ClientKeyPath, "")
	set.String(flags.RemoteSignerCACertPathFlag.Name, wantCfg.RemoteCertificate.CACertPath, "")
	assert.NoError(t, set.Set(flags.WalletDirFlag.Name, walletDir))
	assert.NoError(t, set.Set(flags.KeymanagerKindFlag.Name, keymanagerKind))
	assert.NoError(t, set.Set(flags.GrpcRemoteAddressFlag.Name, wantCfg.RemoteAddr))
	assert.NoError(t, set.Set(flags.RemoteSignerCertPathFlag.Name, wantCfg.RemoteCertificate.ClientCertPath))
	assert.NoError(t, set.Set(flags.RemoteSignerKeyPathFlag.Name, wantCfg.RemoteCertificate.ClientKeyPath))
	assert.NoError(t, set.Set(flags.RemoteSignerCACertPathFlag.Name, wantCfg.RemoteCertificate.CACertPath))
	cliCtx := cli.NewContext(&app, set, nil)

	// We attempt to create the wallet.
	require.NoError(t, CreateWallet(cliCtx))

	// We attempt to open the newly created wallet.
	ctx := context.Background()
	wallet, err := OpenWallet(cliCtx)
	assert.NoError(t, err)

	// We read the keymanager config for the newly created wallet.
	encoded, err := wallet.ReadKeymanagerConfigFromDisk(ctx)
	assert.NoError(t, err)
	cfg, err := remote.UnmarshalConfigFile(encoded)
	assert.NoError(t, err)

	// We assert the created configuration was as desired.
	assert.DeepEqual(t, wantCfg, cfg)
}
