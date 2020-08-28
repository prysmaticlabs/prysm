package v2

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	validatorpb "github.com/prysmaticlabs/prysm/proto/validator/accounts/v2"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/derived"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/direct"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/remote"
)

type mockRemoteKeymanager struct {
	publicKeys [][48]byte
	opts       *remote.KeymanagerOpts
}

func (m *mockRemoteKeymanager) FetchValidatingPublicKeys(ctx context.Context) ([][48]byte, error) {
	return m.publicKeys, nil
}

func (m *mockRemoteKeymanager) Sign(context.Context, *validatorpb.SignRequest) (bls.Signature, error) {
	return nil, nil
}

func TestListAccounts_DirectKeymanager(t *testing.T) {
	walletDir, passwordsDir, walletPasswordFile := setupWalletAndPasswordsDir(t)
	cliCtx := setupWalletCtx(t, &testWalletConfig{
		walletDir:          walletDir,
		passwordsDir:       passwordsDir,
		keymanagerKind:     v2keymanager.Direct,
		walletPasswordFile: walletPasswordFile,
	})
	wallet, err := CreateWalletWithKeymanager(cliCtx.Context, &CreateWalletConfig{
		WalletCfg: &WalletConfig{
			WalletDir:      walletDir,
			KeymanagerKind: v2keymanager.Direct,
			WalletPassword: "Passwordz0320$",
		},
	})
	require.NoError(t, err)
	keymanager, err := direct.NewKeymanager(
		cliCtx.Context,
		&direct.SetupConfig{
			Wallet: wallet,
			Opts:   direct.DefaultKeymanagerOpts(),
		},
	)
	require.NoError(t, err)

	numAccounts := 5
	for i := 0; i < numAccounts; i++ {
		_, err := keymanager.CreateAccount(cliCtx.Context)
		require.NoError(t, err)
	}
	rescueStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	// We call the list direct keymanager accounts function.
	require.NoError(t, listDirectKeymanagerAccounts(true /* show deposit data */, keymanager))

	require.NoError(t, w.Close())
	out, err := ioutil.ReadAll(r)
	require.NoError(t, err)
	os.Stdout = rescueStdout

	// Assert the keymanager kind is printed to stdout.
	stringOutput := string(out)
	if !strings.Contains(stringOutput, "non-HD") {
		t.Error("Did not find keymanager kind in output")
	}

	accountNames, err := keymanager.ValidatingAccountNames()
	require.NoError(t, err)
	pubKeys, err := keymanager.FetchValidatingPublicKeys(cliCtx.Context)
	require.NoError(t, err)

	for i := 0; i < numAccounts; i++ {
		accountName := accountNames[i]
		// Assert the account name is printed to stdout.
		if !strings.Contains(stringOutput, accountName) {
			t.Errorf("Did not find account %s in output", accountName)
		}
		key := pubKeys[i]

		// Assert every public key is printed to stdout.
		if !strings.Contains(stringOutput, fmt.Sprintf("%#x", key)) {
			t.Errorf("Did not find pubkey %#x in output", key)
		}
	}
}

func TestListAccounts_DerivedKeymanager(t *testing.T) {
	walletDir, passwordsDir, passwordFilePath := setupWalletAndPasswordsDir(t)
	cliCtx := setupWalletCtx(t, &testWalletConfig{
		walletDir:          walletDir,
		passwordsDir:       passwordsDir,
		keymanagerKind:     v2keymanager.Derived,
		walletPasswordFile: passwordFilePath,
	})
	wallet, err := CreateWalletWithKeymanager(cliCtx.Context, &CreateWalletConfig{
		WalletCfg: &WalletConfig{
			WalletDir:      walletDir,
			KeymanagerKind: v2keymanager.Derived,
			WalletPassword: "Passwordz0320$",
		},
	})
	require.NoError(t, err)

	keymanager, err := derived.NewKeymanager(
		cliCtx.Context,
		&derived.SetupConfig{
			Opts:                derived.DefaultKeymanagerOpts(),
			Wallet:              wallet,
			SkipMnemonicConfirm: true,
		},
	)
	require.NoError(t, err)

	numAccounts := 5
	depositDataForAccounts := make([][]byte, numAccounts)
	for i := 0; i < numAccounts; i++ {
		_, err := keymanager.CreateAccount(cliCtx.Context, false /*logAccountInfo*/)
		require.NoError(t, err)
		enc, err := keymanager.DepositDataForAccount(uint64(i))
		require.NoError(t, err)
		depositDataForAccounts[i] = enc
	}

	rescueStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	// We call the list direct keymanager accounts function.
	require.NoError(t, listDerivedKeymanagerAccounts(true /* show deposit data */, keymanager))

	require.NoError(t, w.Close())
	out, err := ioutil.ReadAll(r)
	require.NoError(t, err)
	os.Stdout = rescueStdout

	// Assert the keymanager kind is printed to stdout.
	stringOutput := string(out)
	if !strings.Contains(stringOutput, wallet.KeymanagerKind().String()) {
		t.Error("Did not find Keymanager kind in output")
	}

	accountNames, err := keymanager.ValidatingAccountNames(cliCtx.Context)
	require.NoError(t, err)
	pubKeys, err := keymanager.FetchValidatingPublicKeys(cliCtx.Context)
	require.NoError(t, err)

	for i := 0; i < numAccounts; i++ {
		accountName := accountNames[i]
		// Assert the account name is printed to stdout.
		if !strings.Contains(stringOutput, accountName) {
			t.Errorf("Did not find account %s in output", accountName)
		}
		key := pubKeys[i]
		depositData := depositDataForAccounts[i]

		// Assert every public key is printed to stdout.
		if !strings.Contains(stringOutput, fmt.Sprintf("%#x", key)) {
			t.Errorf("Did not find pubkey %#x in output", key)
		}

		// Assert the deposit data for the account is printed to stdout.
		if !strings.Contains(stringOutput, fmt.Sprintf("%#x", depositData)) {
			t.Errorf("Did not find deposit data %#x in output", depositData)
		}
	}
}

//func TestListAccounts_RemoteKeymanager(t *testing.T) {
//	walletDir, _, _ := setupWalletAndPasswordsDir(t)
//	cliCtx := setupWalletCtx(t, &testWalletConfig{
//		walletDir:      walletDir,
//		keymanagerKind: v2keymanager.Remote,
//	})
//	wallet, err := CreateWallet(cliCtx.Context, &WalletConfig{
//		WalletDir:      walletDir,
//		KeymanagerKind: v2keymanager.Remote,
//	})
//	require.NoError(t, err)
//	require.NoError(t, wallet.SaveWallet())
//
//	rescueStdout := os.Stdout
//	r, w, err := os.Pipe()
//	require.NoError(t, err)
//	os.Stdout = w
//
//	numAccounts := 3
//	pubKeys := make([][48]byte, numAccounts)
//	for i := 0; i < numAccounts; i++ {
//		key := make([]byte, 48)
//		copy(key, strconv.Itoa(i))
//		pubKeys[i] = bytesutil.ToBytes48(key)
//	}
//	km := &mockRemoteKeymanager{
//		publicKeys: pubKeys,
//		opts: &remote.KeymanagerOpts{
//			RemoteCertificate: &remote.CertificateConfig{
//				ClientCertPath: "/tmp/client.crt",
//				ClientKeyPath:  "/tmp/client.key",
//				CACertPath:     "/tmp/ca.crt",
//			},
//			RemoteAddr: "localhost:4000",
//		},
//	}
//	// We call the list remote keymanager accounts function.
//	require.NoError(t, listRemoteKeymanagerAccounts(km))
//
//	require.NoError(t, w.Close())
//	out, err := ioutil.ReadAll(r)
//	require.NoError(t, err)
//	os.Stdout = rescueStdout
//
//	// Assert the keymanager kind is printed to stdout.
//	stringOutput := string(out)
//	if !strings.Contains(stringOutput, wallet.KeymanagerKind().String()) {
//		t.Error("Did not find keymanager kind in output")
//	}
//
//	// Assert the keymanager configuration is printed to stdout.
//	if !strings.Contains(stringOutput, cfg.String()) {
//		t.Error("Did not find remote config in output")
//	}
//
//	// Assert the wallet accounts path is in stdout.
//	if !strings.Contains(stringOutput, wallet.accountsPath) {
//		t.Errorf("Did not find accounts path %s in output", wallet.accountsPath)
//	}
//
//	for i := 0; i < numAccounts; i++ {
//		accountName := petnames.DeterministicName(pubKeys[i][:], "-")
//		// Assert the account name is printed to stdout.
//		if !strings.Contains(stringOutput, accountName) {
//			t.Errorf("Did not find account %s in output", accountName)
//		}
//		key := pubKeys[i]
//
//		// Assert every public key is printed to stdout.
//		if !strings.Contains(stringOutput, fmt.Sprintf("%#x", key)) {
//			t.Errorf("Did not find pubkey %#x in output", key)
//		}
//	}
//}
