package wallet

import (
	"flag"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/prysmaticlabs/prysm/v3/cmd/validator/flags"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/validator/accounts"
	"github.com/prysmaticlabs/prysm/v3/validator/keymanager"
	"github.com/prysmaticlabs/prysm/v3/validator/keymanager/remote"
	"github.com/urfave/cli/v2"
)

const (
	passwordFileName = "password.txt"
	password         = "OhWOWthisisatest42!$"
)

// TODO(mikeneuder): Figure out how to shared these functions with
// `cmd/validator/accounts/delete_test.go`. https://pastebin.com/2n2VB7Ez is
// the error I couldn't get around.
func setupWalletAndPasswordsDir(t testing.TB) (string, string, string) {
	walletDir := filepath.Join(t.TempDir(), "wallet")
	passwordsDir := filepath.Join(t.TempDir(), "passwords")
	passwordFileDir := filepath.Join(t.TempDir(), "passwordFile")
	require.NoError(t, os.MkdirAll(passwordFileDir, params.BeaconIoConfig().ReadWriteExecutePermissions))
	passwordFilePath := filepath.Join(passwordFileDir, passwordFileName)
	require.NoError(t, os.WriteFile(passwordFilePath, []byte(password), os.ModePerm))
	return walletDir, passwordsDir, passwordFilePath
}

type testWalletConfig struct {
	exitAll                 bool
	skipDepositConfirm      bool
	keymanagerKind          keymanager.Kind
	numAccounts             int64
	grpcHeaders             string
	privateKeyFile          string
	accountPasswordFile     string
	walletPasswordFile      string
	backupPasswordFile      string
	backupPublicKeys        string
	voluntaryExitPublicKeys string
	deletePublicKeys        string
	keysDir                 string
	backupDir               string
	walletDir               string
	passwordsDir            string
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
	set.Bool(flags.SkipMnemonic25thWordCheckFlag.Name, true, "")
	set.Bool(flags.ExitAllFlag.Name, cfg.exitAll, "")
	set.String(flags.GrpcHeadersFlag.Name, cfg.grpcHeaders, "")

	if cfg.privateKeyFile != "" {
		set.String(flags.ImportPrivateKeyFileFlag.Name, cfg.privateKeyFile, "")
		assert.NoError(tb, set.Set(flags.ImportPrivateKeyFileFlag.Name, cfg.privateKeyFile))
	}
	assert.NoError(tb, set.Set(flags.WalletDirFlag.Name, cfg.walletDir))
	assert.NoError(tb, set.Set(flags.SkipMnemonic25thWordCheckFlag.Name, "true"))
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
	assert.NoError(tb, set.Set(flags.ExitAllFlag.Name, strconv.FormatBool(cfg.exitAll)))
	assert.NoError(tb, set.Set(flags.GrpcHeadersFlag.Name, cfg.grpcHeaders))
	return cli.NewContext(&app, set, nil)
}

func TestEditWalletConfiguration(t *testing.T) {
	walletDir, _, passwordFile := setupWalletAndPasswordsDir(t)
	cliCtx := setupWalletCtx(t, &testWalletConfig{
		walletDir:      walletDir,
		keymanagerKind: keymanager.Remote,
	})
	opts := []accounts.Option{
		accounts.WithWalletDir(walletDir),
		accounts.WithKeymanagerType(keymanager.Remote),
		accounts.WithWalletPassword("Passwordz0320$"),
	}
	acc, err := accounts.NewCLIManager(opts...)
	require.NoError(t, err)
	w, err := acc.WalletCreate(cliCtx.Context)
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
	assert.NoError(t, w.WriteKeymanagerConfigToDisk(cliCtx.Context, encodedCfg))

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

	err = remoteWalletEdit(cliCtx)
	require.NoError(t, err)
	encoded, err := w.ReadKeymanagerConfigFromDisk(cliCtx.Context)
	require.NoError(t, err)

	cfg, err := remote.UnmarshalOptionsFile(encoded)
	assert.NoError(t, err)
	assert.DeepEqual(t, wantCfg, cfg)
}
