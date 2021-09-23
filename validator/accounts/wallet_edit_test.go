package accounts

import (
	"flag"
	"testing"

	"github.com/prysmaticlabs/prysm/cmd/validator/flags"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/validator/accounts/wallet"
	"github.com/prysmaticlabs/prysm/validator/keymanager"
	"github.com/prysmaticlabs/prysm/validator/keymanager/remote"
	"github.com/urfave/cli/v2"
)

func TestEditWalletConfiguration(t *testing.T) {
	walletDir, _, passwordFile := setupWalletAndPasswordsDir(t)
	cliCtx := setupWalletCtx(t, &testWalletConfig{
		walletDir:      walletDir,
		keymanagerKind: keymanager.Remote,
	})
	wallet, err := CreateWalletWithKeymanager(cliCtx.Context, &CreateWalletConfig{
		WalletCfg: &wallet.Config{
			WalletDir:      walletDir,
			KeymanagerKind: keymanager.Remote,
			WalletPassword: "Passwordz0320$",
		},
	})
	require.NoError(t, err)

	originalCfg := &remote.KeymanagerOpts{
		RemoteCertificate: &remote.CertificateConfig{
			RequireTls:     true,
			ClientCertPath: "/tmp/a.crt",
			ClientKeyPath:  "/tmp/b.key",
			CACertPath:     "/tmp/c.crt",
		},
		RemoteAddr: "my.server.com:4000",
	}
	encodedCfg, err := remote.MarshalOptionsFile(cliCtx.Context, originalCfg)
	assert.NoError(t, err)
	assert.NoError(t, wallet.WriteKeymanagerConfigToDisk(cliCtx.Context, encodedCfg))

	wantCfg := &remote.KeymanagerOpts{
		RemoteCertificate: &remote.CertificateConfig{
			RequireTls:     true,
			ClientCertPath: "/tmp/client.crt",
			ClientKeyPath:  "/tmp/client.key",
			CACertPath:     "/tmp/ca.crt",
		},
		RemoteAddr: "host.example.com:4000",
	}
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	set.String(flags.WalletDirFlag.Name, walletDir, "")
	set.String(flags.WalletPasswordFileFlag.Name, passwordFile, "")
	set.String(flags.GrpcRemoteAddressFlag.Name, wantCfg.RemoteAddr, "")
	set.String(flags.RemoteSignerCertPathFlag.Name, wantCfg.RemoteCertificate.ClientCertPath, "")
	set.String(flags.RemoteSignerKeyPathFlag.Name, wantCfg.RemoteCertificate.ClientKeyPath, "")
	set.String(flags.RemoteSignerCACertPathFlag.Name, wantCfg.RemoteCertificate.CACertPath, "")
	assert.NoError(t, set.Set(flags.WalletDirFlag.Name, walletDir))
	assert.NoError(t, set.Set(flags.WalletPasswordFileFlag.Name, passwordFile))
	assert.NoError(t, set.Set(flags.GrpcRemoteAddressFlag.Name, wantCfg.RemoteAddr))
	assert.NoError(t, set.Set(flags.RemoteSignerCertPathFlag.Name, wantCfg.RemoteCertificate.ClientCertPath))
	assert.NoError(t, set.Set(flags.RemoteSignerKeyPathFlag.Name, wantCfg.RemoteCertificate.ClientKeyPath))
	assert.NoError(t, set.Set(flags.RemoteSignerCACertPathFlag.Name, wantCfg.RemoteCertificate.CACertPath))
	cliCtx = cli.NewContext(&app, set, nil)

	err = EditWalletConfigurationCli(cliCtx)
	require.NoError(t, err)
	encoded, err := wallet.ReadKeymanagerConfigFromDisk(cliCtx.Context)
	require.NoError(t, err)

	cfg, err := remote.UnmarshalOptionsFile(encoded)
	assert.NoError(t, err)
	assert.DeepEqual(t, wantCfg, cfg)
}
