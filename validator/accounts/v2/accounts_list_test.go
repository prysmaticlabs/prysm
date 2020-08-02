package v2

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"testing"

	validatorpb "github.com/prysmaticlabs/prysm/proto/validator/accounts/v2"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/petnames"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/derived"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/direct"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/remote"
)

type mockKeymanager struct {
	publicKeys [][48]byte
}

func (m *mockKeymanager) FetchValidatingPublicKeys(ctx context.Context) ([][48]byte, error) {
	return m.publicKeys, nil
}

func (m *mockKeymanager) Sign(context.Context, *validatorpb.SignRequest) (bls.Signature, error) {
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
	wallet, err := NewWallet(cliCtx, v2keymanager.Direct)
	require.NoError(t, err)
	require.NoError(t, wallet.SaveWallet())
	ctx := context.Background()
	keymanager, err := direct.NewKeymanager(
		ctx,
		wallet,
		direct.DefaultConfig(),
	)
	require.NoError(t, err)

	numAccounts := 5
	for i := 0; i < numAccounts; i++ {
		_, err := keymanager.CreateAccount(ctx, "hello world")
		require.NoError(t, err)
	}
	rescueStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	// We call the list direct keymanager accounts function.
	require.NoError(t, listDirectKeymanagerAccounts(true /* show deposit data */, wallet, keymanager))

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
	pubKeys, err := keymanager.FetchValidatingPublicKeys(ctx)
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
	wallet, err := NewWallet(cliCtx, v2keymanager.Derived)
	require.NoError(t, err)
	require.NoError(t, wallet.SaveWallet())
	ctx := context.Background()

	seedConfig, err := derived.InitializeWalletSeedFile(ctx, password, true /* skip confirm */)
	require.NoError(t, err)

	// Create a new wallet seed file and write it to disk.
	seedConfigFile, err := derived.MarshalEncryptedSeedFile(ctx, seedConfig)
	require.NoError(t, err)
	require.NoError(t, wallet.WriteFileAtPath(ctx, "", derived.EncryptedSeedFileName, seedConfigFile))

	keymanager, err := derived.NewKeymanager(
		ctx,
		wallet,
		derived.DefaultConfig(),
		true, /* skip confirm */
		password,
	)
	require.NoError(t, err)

	numAccounts := 5
	depositDataForAccounts := make([][]byte, numAccounts)
	for i := 0; i < numAccounts; i++ {
		_, err := keymanager.CreateAccount(ctx, false /*logAccountInfo*/)
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
	require.NoError(t, listDerivedKeymanagerAccounts(true /* show deposit data */, wallet, keymanager))

	require.NoError(t, w.Close())
	out, err := ioutil.ReadAll(r)
	require.NoError(t, err)
	os.Stdout = rescueStdout

	// Assert the keymanager kind is printed to stdout.
	stringOutput := string(out)
	if !strings.Contains(stringOutput, wallet.KeymanagerKind().String()) {
		t.Error("Did not find Keymanager kind in output")
	}

	accountNames, err := keymanager.ValidatingAccountNames(ctx)
	require.NoError(t, err)
	pubKeys, err := keymanager.FetchValidatingPublicKeys(ctx)
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

func TestListAccounts_RemoteKeymanager(t *testing.T) {
	walletDir, _, _ := setupWalletAndPasswordsDir(t)
	cliCtx := setupWalletCtx(t, &testWalletConfig{
		walletDir:      walletDir,
		keymanagerKind: v2keymanager.Remote,
	})
	wallet, err := NewWallet(cliCtx, v2keymanager.Remote)
	require.NoError(t, err)
	require.NoError(t, wallet.SaveWallet())

	rescueStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	numAccounts := 3
	pubKeys := make([][48]byte, numAccounts)
	for i := 0; i < numAccounts; i++ {
		key := make([]byte, 48)
		copy(key, strconv.Itoa(i))
		pubKeys[i] = bytesutil.ToBytes48(key)
	}
	km := &mockKeymanager{
		publicKeys: pubKeys,
	}
	// We call the list remote keymanager accounts function.
	cfg := &remote.Config{
		RemoteCertificate: &remote.CertificateConfig{
			ClientCertPath: "/tmp/client.crt",
			ClientKeyPath:  "/tmp/client.key",
			CACertPath:     "/tmp/ca.crt",
		},
		RemoteAddr: "localhost:4000",
	}
	require.NoError(t, listRemoteKeymanagerAccounts(wallet, km, cfg))

	require.NoError(t, w.Close())
	out, err := ioutil.ReadAll(r)
	require.NoError(t, err)
	os.Stdout = rescueStdout

	// Assert the keymanager kind is printed to stdout.
	stringOutput := string(out)
	if !strings.Contains(stringOutput, wallet.KeymanagerKind().String()) {
		t.Error("Did not find keymanager kind in output")
	}

	// Assert the keymanager configuration is printed to stdout.
	if !strings.Contains(stringOutput, cfg.String()) {
		t.Error("Did not find remote config in output")
	}

	// Assert the wallet accounts path is in stdout.
	if !strings.Contains(stringOutput, wallet.accountsPath) {
		t.Errorf("Did not find accounts path %s in output", wallet.accountsPath)
	}

	for i := 0; i < numAccounts; i++ {
		accountName := petnames.DeterministicName(pubKeys[i][:], "-")
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
