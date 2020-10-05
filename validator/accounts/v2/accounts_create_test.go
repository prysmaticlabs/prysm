package v2

import (
	"context"
	"io/ioutil"
	"os"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/promptutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/prysmaticlabs/prysm/validator/accounts/v2/wallet"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/derived"
	logTest "github.com/sirupsen/logrus/hooks/test"
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
	w, err := wallet.OpenWallet(cliCtx.Context, &wallet.Config{
		WalletDir:      walletDir,
		WalletPassword: password,
	})
	assert.NoError(t, err)

	// We read the keymanager config for the newly created wallet.
	encoded, err := w.ReadKeymanagerConfigFromDisk(ctx)
	assert.NoError(t, err)
	opts, err := derived.UnmarshalOptionsFile(encoded)
	assert.NoError(t, err)

	// We assert the created configuration was as desired.
	assert.DeepEqual(t, derived.DefaultKeymanagerOpts(), opts)

	require.NoError(t, CreateAccountCli(cliCtx))

	keymanager, err := w.InitializeKeymanager(cliCtx.Context, true)
	require.NoError(t, err)
	km, ok := keymanager.(*derived.Keymanager)
	if !ok {
		t.Fatal("not a derived keymanager")
	}
	names, err := km.ValidatingAccountNames(ctx)
	assert.NoError(t, err)
	require.Equal(t, len(names), int(numAccounts))
}

// passwordReader will store data that will be later used to mock Stdin by Test_KeysConsistency_Direct
type passwordReader struct {
	password string
	counter  int // counter equals the maximum number of times method passwordReaderFunc can be called
}

// Instead of forwarding the read request to terminal.ReadPassword(), we simply provide a canned response.
func (p *passwordReader) passwordReaderFunc(file *os.File) ([]byte, error) {
	p.counter--
	if p.counter <= 0 {
		log.Fatalln("Too many password attempts using passwordReaderFunc()")
	}
	return []byte(p.password), nil
}

// Test_KeysConsistency_Direct checks that the password does not change due to account creation in a Direct wallet
func Test_KeysConsistency_Direct(t *testing.T) {
	walletDir, passwordsDir, walletPasswordFile := setupWalletAndPasswordsDir(t)

	// Specify the 'initial'/correct password locally to this file for convenience.
	require.NoError(t, ioutil.WriteFile(walletPasswordFile, []byte("Pa$sW0rD0__Fo0xPr"), os.ModePerm))

	cliCtx := setupWalletCtx(t, &testWalletConfig{
		walletDir:          walletDir,
		passwordsDir:       passwordsDir,
		keymanagerKind:     v2keymanager.Direct,
		walletPasswordFile: walletPasswordFile,
	})

	w, err := CreateAndSaveWalletCli(cliCtx)
	require.NoError(t, err)

	// Create an account using "Pa$sW0rD0__Fo0xPr"
	err = CreateAccount(cliCtx.Context, &CreateAccountConfig{
		Wallet:      w,
		NumAccounts: 1,
	})
	require.NoError(t, err)

	/* The bug this test checks for works like this:  Input wrong password followed by the correct password.
	This causes the wallet's password to change to the (initially) wrong provided password.
	*/

	// Now we change the password to "SecoNDxyzPass__9!@#"
	require.NoError(t, ioutil.WriteFile(walletPasswordFile, []byte("SecoNDxyzPass__9!@#"), os.ModePerm))
	_, err = wallet.OpenWalletOrElseCli(cliCtx, CreateAndSaveWalletCli)
	require.ErrorContains(t, "wrong password for wallet", err)

	require.NoError(t, ioutil.WriteFile(walletPasswordFile, []byte("Pa$sW0rD0__Fo0xPr"), os.ModePerm))
	w, err = wallet.OpenWalletOrElseCli(cliCtx, CreateAndSaveWalletCli)
	require.NoError(t, err)

	/*  The purpose of using a passwordReader object is to store a 'canned' response for when the program
	asks for more passwords.  As we are about to call CreateAccount() with an incorrect password, we expect the
	program to ask for more attempts via Stdin.	 This will provide the correct password.*/
	mockPasswordReader := passwordReader{password: "Pa$sW0rD0__Fo0xPr", counter: 3}
	// Redirect promptutil's PasswordReader to our function which bypasses/mocks Stdin
	promptutil.PasswordReader = mockPasswordReader.passwordReaderFunc

	err = CreateAccount(cliCtx.Context, &CreateAccountConfig{
		Wallet:      w,
		NumAccounts: 1,
	})
	require.NoError(t, err)

	// Now we make sure a bug did not change the password to "SecoNDxyzPass__9!@#"
	logHook := logTest.NewGlobal()
	require.NoError(t, ioutil.WriteFile(walletPasswordFile, []byte("Pa$sW0rD0__Fo0xPr"), os.ModePerm))
	w, err = wallet.OpenWalletOrElseCli(cliCtx, CreateAndSaveWalletCli)
	require.NoError(t, err)
	mockPasswordReader.counter = 3

	err = CreateAccount(cliCtx.Context, &CreateAccountConfig{
		Wallet:      w,
		NumAccounts: 1,
	})
	require.NoError(t, err)
	assert.LogsContain(t, logHook, "Successfully created new validator account")
}
