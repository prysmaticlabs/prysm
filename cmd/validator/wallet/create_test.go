package wallet

import (
	"flag"
	"io"
	"testing"

	"github.com/pkg/errors"
	cmdacc "github.com/prysmaticlabs/prysm/v3/cmd/validator/accounts"
	"github.com/prysmaticlabs/prysm/v3/cmd/validator/flags"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/validator/accounts"
	"github.com/prysmaticlabs/prysm/v3/validator/accounts/wallet"
	"github.com/prysmaticlabs/prysm/v3/validator/keymanager"
	"github.com/prysmaticlabs/prysm/v3/validator/keymanager/local"
	"github.com/sirupsen/logrus"
	logTest "github.com/sirupsen/logrus/hooks/test"
	"github.com/urfave/cli/v2"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(io.Discard)
}

func TestCreateOrOpenWallet(t *testing.T) {
	hook := logTest.NewGlobal()
	walletDir, passwordsDir, walletPasswordFile := setupWalletAndPasswordsDir(t)
	cliCtx := setupWalletCtx(t, &testWalletConfig{
		walletDir:          walletDir,
		passwordsDir:       passwordsDir,
		keymanagerKind:     keymanager.Local,
		walletPasswordFile: walletPasswordFile,
	})
	createLocalWallet := func(cliCtx *cli.Context) (*wallet.Wallet, error) {
		cfg, err := cmdacc.ExtractWalletDirPassword(cliCtx)
		if err != nil {
			return nil, err
		}
		w := wallet.New(&wallet.Config{
			KeymanagerKind: keymanager.Local,
			WalletDir:      cfg.Dir,
			WalletPassword: cfg.Password,
		})
		if err = accounts.CreateLocalKeymanagerWallet(cliCtx.Context, w); err != nil {
			return nil, errors.Wrap(err, "could not create keymanager")
		}
		log.WithField("wallet-path", cfg.Dir).Info(
			"Successfully created new wallet",
		)
		return w, nil
	}
	createdWallet, err := wallet.OpenWalletOrElseCli(cliCtx, createLocalWallet)
	require.NoError(t, err)
	require.LogsContain(t, hook, "Successfully created new wallet")

	openedWallet, err := wallet.OpenWalletOrElseCli(cliCtx, createLocalWallet)
	require.NoError(t, err)
	assert.Equal(t, createdWallet.KeymanagerKind(), openedWallet.KeymanagerKind())
	assert.Equal(t, createdWallet.AccountsDir(), openedWallet.AccountsDir())
}

func TestCreateWallet_Local(t *testing.T) {
	walletDir, passwordsDir, walletPasswordFile := setupWalletAndPasswordsDir(t)
	cliCtx := setupWalletCtx(t, &testWalletConfig{
		walletDir:          walletDir,
		passwordsDir:       passwordsDir,
		keymanagerKind:     keymanager.Local,
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
	_, err = w.ReadFileAtPath(cliCtx.Context, local.AccountsPath, local.AccountsKeystoreFileName)
	require.NoError(t, err)
}

func TestCreateWallet_Derived(t *testing.T) {
	walletDir, passwordsDir, passwordFile := setupWalletAndPasswordsDir(t)
	cliCtx := setupWalletCtx(t, &testWalletConfig{
		walletDir:          walletDir,
		passwordsDir:       passwordsDir,
		walletPasswordFile: passwordFile,
		keymanagerKind:     keymanager.Derived,
		numAccounts:        1,
	})

	// We attempt to create the wallet.
	_, err := CreateAndSaveWalletCli(cliCtx)
	require.NoError(t, err)

	// We attempt to open the newly created wallet.
	_, err = wallet.OpenWallet(cliCtx.Context, &wallet.Config{
		WalletDir: walletDir,
	})
	assert.NoError(t, err)
}

// TestCreateWallet_WalletAlreadyExists checks for expected error if trying to create a wallet when there is one already.
func TestCreateWallet_WalletAlreadyExists(t *testing.T) {
	walletDir, passwordsDir, passwordFile := setupWalletAndPasswordsDir(t)
	cliCtx := setupWalletCtx(t, &testWalletConfig{
		walletDir:          walletDir,
		passwordsDir:       passwordsDir,
		walletPasswordFile: passwordFile,
		keymanagerKind:     keymanager.Derived,
		numAccounts:        1,
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
		keymanagerKind:     keymanager.Local,
	})

	// We attempt to create another wallet of different type at the same location. We expect an error.
	_, err = CreateAndSaveWalletCli(cliCtx)
	require.ErrorContains(t, "already exists", err)
}

func TestInputKeymanagerKind(t *testing.T) {
	tests := []struct {
		name    string
		args    string
		want    keymanager.Kind
		wantErr bool
	}{
		{
			name:    "local returns local kind",
			args:    "local",
			want:    keymanager.Local,
			wantErr: false,
		},
		{
			name:    "direct returns local kind",
			args:    "direct",
			want:    keymanager.Local,
			wantErr: false,
		},
		{
			name:    "imported returns local kind",
			args:    "imported",
			want:    keymanager.Local,
			wantErr: false,
		},
		{
			name:    "derived returns derived kind",
			args:    "derived",
			want:    keymanager.Derived,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := cli.App{}
			set := flag.NewFlagSet("test", 0)
			set.String(flags.KeymanagerKindFlag.Name, tt.args, "")
			assert.NoError(t, set.Set(flags.KeymanagerKindFlag.Name, tt.args))
			cliCtx := cli.NewContext(&app, set, nil)
			got, err := inputKeymanagerKind(cliCtx)
			assert.NoError(t, err)
			assert.Equal(t, tt.want.String(), got.String())
		})
	}
}
