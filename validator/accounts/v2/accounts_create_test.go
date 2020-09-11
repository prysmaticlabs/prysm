package v2

import (
	"context"
	"io/ioutil"
	"os"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/promptutil"
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

	// Now change the password provided to the wrong one
	require.NoError(t, ioutil.WriteFile(walletPasswordFile, []byte("SecoNDxyzPass__9!@#"), os.ModePerm))

	wallet, err = OpenWalletOrElseCli(cliCtx, CreateAndSaveWalletCli)
	require.NoError(t, err)

	// Create a file that will hold input we would otherwise type
	//replaceStdin, err := ioutil.TempFile("", "temp")
	//require.NoError(t, err)
	//defer func() {
	//	fileName := replaceStdin.Name()
	//	err = replaceStdin.Close()
	//	require.NoError(t, err)
	//	err = os.Remove(fileName)
	//	require.NoError(t, err)
	//}()
	// Write in the correct password
	//_, err = replaceStdin.Write([]byte("Pa$sW0rD0__Fo0xPr"))
	//require.NoError(t, err)
	//_, err = replaceStdin.Seek(0, 0)
	//require.NoError(t, err)

	// @@@FILL IN THIS COMMENT@@@
	mockPasswordReader := passwordReader{password: "Pa$sW0rD0__Fo0xPr"}
	promptutil.PasswordReader = mockPasswordReader.passwordReaderFunc

	err = CreateAccount(cliCtx.Context, &CreateAccountConfig{
		Wallet:      wallet,
		NumAccounts: 1,
	})
	require.NoError(t, err)

}

type passwordReader struct{
	password string
}

func (p *passwordReader) passwordReaderFunc(file *os.File) ([]byte, error){
	return []byte(p.password), nil
}

