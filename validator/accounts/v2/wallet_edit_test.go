package v2

import (
	"context"
	"flag"
	"os"
	"reflect"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/remote"
	"github.com/urfave/cli/v2"
)

func TestEditWalletConfiguration(t *testing.T) {
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	walletDir := testutil.TempDir() + "/wallet"
	defer func() {
		if err := os.RemoveAll(walletDir); err != nil {
			t.Log(err)
		}
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
	if err != nil {
		t.Fatal(err)
	}
	walletConfig := &WalletConfig{
		WalletDir:      walletDir,
		KeymanagerKind: v2keymanager.Remote,
	}
	wallet, err := NewWallet(ctx, walletConfig)
	if err != nil {
		t.Fatal(err)
	}
	if err := wallet.WriteKeymanagerConfigToDisk(ctx, encodedCfg); err != nil {
		t.Fatal(err)
	}

	wantCfg := &remote.Config{
		RemoteCertificate: &remote.CertificateConfig{
			ClientCertPath: "/tmp/client.crt",
			ClientKeyPath:  "/tmp/client.key",
			CACertPath:     "/tmp/ca.crt",
		},
		RemoteAddr: "host.example.com:4000",
	}
	set.String("wallet-dir", walletDir, "")
	set.String("grpc-remote-address", wantCfg.RemoteAddr, "")
	set.String("remote-signer-crt-path", wantCfg.RemoteCertificate.ClientCertPath, "")
	set.String("remote-signer-key-path", wantCfg.RemoteCertificate.ClientKeyPath, "")
	set.String("remote-signer-ca-crt-path", wantCfg.RemoteCertificate.CACertPath, "")
	cliCtx := cli.NewContext(&app, set, nil)
	if err := EditWalletConfiguration(cliCtx); err != nil {
		t.Fatal(err)
	}
	encoded, err := wallet.ReadKeymanagerConfigFromDisk(ctx)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := remote.UnmarshalConfigFile(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(wantCfg, cfg) {
		t.Errorf("Wanted %v, received %v", wantCfg, cfg)
	}
}
