package accounts

import (
	"context"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/promptutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/prysmaticlabs/prysm/validator/accounts/wallet"
	"github.com/prysmaticlabs/prysm/validator/keymanager"
	"github.com/prysmaticlabs/prysm/validator/keymanager/derived"
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
		keymanagerKind:      keymanager.Derived,
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
func (p *passwordReader) passwordReaderFunc(_ *os.File) ([]byte, error) {
	p.counter--
	if p.counter <= 0 {
		log.Fatalln("Too many password attempts using passwordReaderFunc()")
	}
	return []byte(p.password), nil
}

// Test_KeysConsistency_Imported checks that the password does not change due to account creation in a Imported wallet
func Test_KeysConsistency_Imported(t *testing.T) {
	walletDir, passwordsDir, walletPasswordFile := setupWalletAndPasswordsDir(t)

	// Specify the 'initial'/correct password locally to this file for convenience.
	require.NoError(t, ioutil.WriteFile(walletPasswordFile, []byte("Pa$sW0rD0__Fo0xPr"), os.ModePerm))

	cliCtx := setupWalletCtx(t, &testWalletConfig{
		walletDir:          walletDir,
		passwordsDir:       passwordsDir,
		keymanagerKind:     keymanager.Imported,
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

func TestDepositDataJSON(t *testing.T) {
	// Use a real deposit data JSON fixture generated by the eth2 deposit cli
	fixture := make(map[string]string)
	fixture["pubkey"] = "a611f309b4a24853e0b04bd70e35fbac887e099b9f81c2fac2bb2cde9f6f58bd37d947be552ec515b1f45d406f61de27"
	fixture["withdrawal_credentials"] = "003561705197f621bfaa59add59ee066e6f2fe356201d00c610ed5d6cd7fcb83"
	fixture["amount"] = "32000000000"
	fixture["signature"] = "b0a27f2e7684fc1aa6403e2e76dcbcf29568ba02e9076e61b4c926bccec25ec636a1fdc8d08457cf23a1715ea9ee4fe20b030820e2fcf6dee07a3ce5e6ec65a824027f4cb01c143db74b34f5ca54f7e011d84fe89ce55b0e75f39003e2c9afe9"
	fixture["deposit_message_root"] = "12c267fdc80fb07b47770f8fcf5e25ed2280df391d7de224cc6486e925b7d7f9"
	fixture["deposit_data_root"] = "3b3c62bcff04d0249209c79a76cea98520932609986c11cb4ff62a4f54b76548"
	fixture["fork_version"] = fmt.Sprintf("%x", params.BeaconConfig().GenesisForkVersion)

	pubKey, err := hex.DecodeString(fixture["pubkey"])
	require.NoError(t, err)
	credentials, err := hex.DecodeString(fixture["withdrawal_credentials"])
	require.NoError(t, err)
	sig, err := hex.DecodeString(fixture["signature"])
	require.NoError(t, err)
	depositData := &ethpb.Deposit_Data{
		PublicKey:             pubKey,
		WithdrawalCredentials: credentials,
		Amount:                32000000000,
		Signature:             sig,
	}
	got, err := DepositDataJSON(depositData)
	require.NoError(t, err)
	assert.DeepEqual(t, fixture, got)
}
