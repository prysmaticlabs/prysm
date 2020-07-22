package v2

import (
	"context"
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/prysmaticlabs/prysm/validator/flags"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/derived"
	"github.com/urfave/cli/v2"
)

func TestRecoverDerivedWallet(t *testing.T) {
	testDir := testutil.TempDir()
	walletDir := filepath.Join(testDir, walletDirName)
	passwordsDir := filepath.Join(testDir, passwordDirName)
	exportDir := filepath.Join(testDir, exportDirName)
	defer func() {
		assert.NoError(t, os.RemoveAll(walletDir))
		assert.NoError(t, os.RemoveAll(passwordsDir))
		assert.NoError(t, os.RemoveAll(exportDir))
	}()

	passwordFilePath := filepath.Join(testDir, passwordFileName)
	require.NoError(t, ioutil.WriteFile(passwordFilePath, []byte(password), os.ModePerm))
	mnemonicFilePath := filepath.Join(testDir, mnemonicFileName)
	require.NoError(t, ioutil.WriteFile(mnemonicFilePath, []byte(mnemonic), os.ModePerm))

	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	set.String(flags.WalletDirFlag.Name, walletDir, "")
	set.String(flags.WalletPasswordsDirFlag.Name, passwordsDir, "")
	set.String(flags.PasswordFileFlag.Name, passwordFilePath, "")
	set.String(flags.MnemonicFileFlag.Name, mnemonicFilePath, "")
	assert.NoError(t, set.Set(flags.WalletDirFlag.Name, walletDir))
	assert.NoError(t, set.Set(flags.WalletPasswordsDirFlag.Name, passwordsDir))
	assert.NoError(t, set.Set(flags.PasswordFileFlag.Name, passwordFilePath))
	assert.NoError(t, set.Set(flags.MnemonicFileFlag.Name, mnemonicFilePath))
	cliCtx := cli.NewContext(&app, set, nil)

	if err := recoverDerivedWallet(cliCtx, walletDir); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	wallet, err := OpenWallet(ctx, &WalletConfig{
		WalletDir:         walletDir,
		PasswordsDir:      passwordsDir,
		KeymanagerKind:    v2keymanager.Derived,
		CanUnlockAccounts: false,
	})
	assert.NoError(t, err)

	encoded, err := wallet.ReadKeymanagerConfigFromDisk(ctx)
	assert.NoError(t, err)
	cfg, err := derived.UnmarshalConfigFile(encoded)
	assert.NoError(t, err)

	// We assert the created configuration was as desired.
	wantCfg := derived.DefaultConfig()
	assert.DeepEqual(t, wantCfg, cfg)
}
