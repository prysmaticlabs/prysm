package v2

import (
	"context"
	"io/ioutil"
	"os"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/derived"
)

func TestCreateAccount_Derived(t *testing.T) {
	walletDir, passwordsDir, passwordFile := setupWalletAndPasswordsDir(t)
	numAccounts := int64(5)
	cliCtx := setupWalletCtx(t, &testWalletConfig{
		walletDir:           walletDir,
		passwordsDir:        passwordsDir,
		walletPasswordFile:  passwordFile,
		accountPasswordFile: passwordFile,
		keymanagerKind:      v2keymanager.Derived,
		numAccounts:         numAccounts,
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
	opts, err := derived.UnmarshalOptionsFile(encoded)
	assert.NoError(t, err)

	// We assert the created configuration was as desired.
	assert.DeepEqual(t, derived.DefaultKeymanagerOpts(), opts)

	require.NoError(t, CreateAccountCli(cliCtx))

	keymanager, err := wallet.InitializeKeymanager(cliCtx.Context, true)
	require.NoError(t, err)
	km, ok := keymanager.(*derived.Keymanager)
	if !ok {
		t.Fatal("not a derived keymanager")
	}
	names, err := km.ValidatingAccountNames(ctx)
	assert.NoError(t, err)
	require.Equal(t, len(names), int(numAccounts))
}

// Test_KeysConsistency_Direct checks that the password does not change due to account creation in a Direct wallet
func Test_KeysConsistency_Direct(t *testing.T) {
	walletDir, passwordsDir, walletPasswordFile := setupWalletAndPasswordsDir(t)

	//Specify the 'initial'/correct password here
	err := ioutil.WriteFile(walletPasswordFile, []byte("Pa$sW0rD0__Fo0xPr"), os.ModePerm)
	require.NoError(t, err)

	cliCtx := setupWalletCtx(t, &testWalletConfig{
		walletDir:          walletDir,
		passwordsDir:       passwordsDir,
		keymanagerKind:     v2keymanager.Direct,
		walletPasswordFile: walletPasswordFile,
	})

	wallet, err := CreateAndSaveWalletCli(cliCtx)
	require.NoError(t, err)

	err = CreateAccount(cliCtx.Context, &CreateAccountConfig{
		Wallet:      wallet,
		NumAccounts: 1,
	})
	require.NoError(t, err)

	// Now change the password provided
	require.NoError(t, ioutil.WriteFile(walletPasswordFile, []byte("SecoNDxyzPass__9!@#"), os.ModePerm))

	wallet, err = OpenWalletOrElseCli(cliCtx, CreateAndSaveWalletCli)
	require.NoError(t, err)

	// The bug this regression test is for arises when first there is a wrong password followed by the correct one
	// If the first password is incorrect, the program will ask for more via standard input

	//tmpFile, err := ioutil.TempFile("", "testFile")
	//tmpFile.Write([]byte("Pa$sW0rD0__Fo0xPr"))
	//require.NoError(t, err)
	//defer os.Remove(tmpFile.Name())
	//_, err = tmpFile.Seek(0, 0)
	//require.NoError(t, err)
	//
	//savedStdin := os.Stdin
	//defer func() {
	//	os.Stdin = savedStdin
	//}()
	//os.Stdin = tmpFile

	err = CreateAccount(cliCtx.Context, &CreateAccountConfig{
		Wallet:      wallet,
		NumAccounts: 1,
	})
	require.NoError(t, err)

}
