package v2

import (
	"context"
	"flag"
	"os"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/validator/flags"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/remote"
	"github.com/urfave/cli/v2"
)

func TestEditWalletConfiguration(t *testing.T) {
	walletDir := testutil.TempDir() + "/wallet"
	defer func() {
		assert.NoError(t, os.RemoveAll(walletDir))
	}()
	ctx := context.Background()
	originalCfg := &remote.Config{
		RemoteCertificate: &remote.CertificateConfig{
			ClientCertPath: "/tmp/a.crt",
			ClientKeyPath:  "/tmp/b.key",
			CACertPath:     "/tmp/c.crt",
		},
		RemoteAddr: "my.server.com:4000",
	}
	encodedCfg, err := remote.MarshalConfigFile(ctx, originalCfg)
	assert.NoError(t, err)
	walletConfig := &WalletConfig{
		WalletDir:      walletDir,
		KeymanagerKind: v2keymanager.Remote,
	}
	wallet, err := NewWallet(ctx, walletConfig)
	assert.NoError(t, err)
	assert.NoError(t, wallet.WriteKeymanagerConfigToDisk(ctx, encodedCfg))

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
	set.String(flags.WalletDirFlag.Name, walletDir, "")
	set.String(flags.GrpcRemoteAddressFlag.Name, wantCfg.RemoteAddr, "")
	set.String(flags.RemoteSignerCertPathFlag.Name, wantCfg.RemoteCertificate.ClientCertPath, "")
	set.String(flags.RemoteSignerKeyPathFlag.Name, wantCfg.RemoteCertificate.ClientKeyPath, "")
	set.String(flags.RemoteSignerCACertPathFlag.Name, wantCfg.RemoteCertificate.CACertPath, "")
	assert.NoError(t, set.Set(flags.WalletDirFlag.Name, walletDir))
	assert.NoError(t, set.Set(flags.GrpcRemoteAddressFlag.Name, wantCfg.RemoteAddr))
	assert.NoError(t, set.Set(flags.RemoteSignerCertPathFlag.Name, wantCfg.RemoteCertificate.ClientCertPath))
	assert.NoError(t, set.Set(flags.RemoteSignerKeyPathFlag.Name, wantCfg.RemoteCertificate.ClientKeyPath))
	assert.NoError(t, set.Set(flags.RemoteSignerCACertPathFlag.Name, wantCfg.RemoteCertificate.CACertPath))
	cliCtx := cli.NewContext(&app, set, nil)

	assert.NoError(t, EditWalletConfiguration(cliCtx))
	encoded, err := wallet.ReadKeymanagerConfigFromDisk(ctx)
	assert.NoError(t, err)

	cfg, err := remote.UnmarshalConfigFile(encoded)
	assert.NoError(t, err)
	assert.DeepEqual(t, wantCfg, cfg)
}
