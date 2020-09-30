package rpc

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/validator/accounts/v2"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	v2 "github.com/prysmaticlabs/prysm/validator/accounts/v2"
	"github.com/prysmaticlabs/prysm/validator/accounts/v2/wallet"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/derived"
)

func TestServer_CreateAccount(t *testing.T) {
	ctx := context.Background()
	localWalletDir := setupWalletDir(t)
	defaultWalletPath = localWalletDir
	strongPass := "29384283xasjasd32%%&*@*#*"
	// We attempt to create the wallet.
	w, err := v2.CreateWalletWithKeymanager(ctx, &v2.CreateWalletConfig{
		WalletCfg: &wallet.Config{
			WalletDir:      defaultWalletPath,
			KeymanagerKind: v2keymanager.Derived,
			WalletPassword: strongPass,
		},
		SkipMnemonicConfirm: true,
	})
	require.NoError(t, err)
	km, err := w.InitializeKeymanager(ctx, true /* skip mnemonic confirm */)
	require.NoError(t, err)
	s := &Server{
		keymanager:        km,
		walletInitialized: true,
		wallet:            w,
	}
	_, err = s.CreateAccount(ctx, &pb.CreateAccountRequest{})
	require.NoError(t, err)
}

func TestServer_ListAccounts(t *testing.T) {
	ctx := context.Background()
	localWalletDir := setupWalletDir(t)
	defaultWalletPath = localWalletDir
	strongPass := "29384283xasjasd32%%&*@*#*"
	// We attempt to create the wallet.
	w, err := v2.CreateWalletWithKeymanager(ctx, &v2.CreateWalletConfig{
		WalletCfg: &wallet.Config{
			WalletDir:      defaultWalletPath,
			KeymanagerKind: v2keymanager.Derived,
			WalletPassword: strongPass,
		},
		SkipMnemonicConfirm: true,
	})
	require.NoError(t, err)
	km, err := w.InitializeKeymanager(ctx, true /* skip mnemonic confirm */)
	require.NoError(t, err)
	s := &Server{
		keymanager:        km,
		walletInitialized: true,
		wallet:            w,
	}
	numAccounts := 5
	keys := make([][]byte, numAccounts)
	for i := 0; i < numAccounts; i++ {
		key, err := km.(*derived.Keymanager).CreateAccount(ctx, false /* log account info */)
		require.NoError(t, err)
		keys[i] = key
	}
	resp, err := s.ListAccounts(ctx, &pb.ListAccountsRequest{})
	require.NoError(t, err)
	require.Equal(t, len(resp.Accounts), numAccounts)
	for i := 0; i < numAccounts; i++ {
		assert.DeepEqual(t, resp.Accounts[i].ValidatingPublicKey, keys[i])
	}
}

func TestServer_BackupAccounts(t *testing.T) {
	ss, pubKeys := createDirectWalletWithAccounts(t, 3)

	// We now attempt to backup all public keys from the wallet.
	res, err := ss.BackupAccounts(context.Background(), &pb.BackupAccountsRequest{
		PublicKeys:     pubKeys,
		BackupPassword: ss.wallet.Password(),
	})
	require.NoError(t, err)
	require.NotNil(t, res.ZipFile)

	// Open a zip archive for reading.
	buf := bytes.NewReader(res.ZipFile)
	r, err := zip.NewReader(buf, int64(len(res.ZipFile)))
	require.NoError(t, err)
	require.Equal(t, len(pubKeys), len(r.File))

	// Iterate through the files in the archive, checking they
	// match the keystores we wanted to backup.
	for i, f := range r.File {
		keystoreFile, err := f.Open()
		require.NoError(t, err)
		encoded, err := ioutil.ReadAll(keystoreFile)
		if err != nil {
			require.NoError(t, keystoreFile.Close())
			t.Fatal(err)
		}
		keystore := &v2keymanager.Keystore{}
		if err := json.Unmarshal(encoded, &keystore); err != nil {
			require.NoError(t, keystoreFile.Close())
			t.Fatal(err)
		}
		assert.Equal(t, keystore.Pubkey, fmt.Sprintf("%x", pubKeys[i]))
		require.NoError(t, keystoreFile.Close())
	}
}

func TestServer_DeleteAccounts_FailedPreconditions_WrongKeymanagerKind(t *testing.T) {
	localWalletDir := setupWalletDir(t)
	defaultWalletPath = localWalletDir
	ctx := context.Background()
	strongPass := "29384283xasjasd32%%&*@*#*"
	w, err := v2.CreateWalletWithKeymanager(ctx, &v2.CreateWalletConfig{
		WalletCfg: &wallet.Config{
			WalletDir:      defaultWalletPath,
			KeymanagerKind: v2keymanager.Derived,
			WalletPassword: strongPass,
		},
		SkipMnemonicConfirm: true,
	})
	require.NoError(t, err)
	require.NoError(t, w.SaveHashedPassword(ctx))
	km, err := w.InitializeKeymanager(ctx, true /* skip mnemonic confirm */)
	require.NoError(t, err)
	ss := &Server{
		wallet:     w,
		keymanager: km,
	}
	_, err = ss.DeleteAccounts(ctx, &pb.DeleteAccountsRequest{
		PublicKeys: make([][]byte, 1),
	})
	assert.ErrorContains(t, "Only Non-HD wallets can delete accounts", err)
}

func TestServer_DeleteAccounts_FailedPreconditions(t *testing.T) {
	ss := &Server{}
	ctx := context.Background()
	_, err := ss.DeleteAccounts(ctx, &pb.DeleteAccountsRequest{})
	assert.ErrorContains(t, "No public keys specified", err)
	_, err = ss.DeleteAccounts(ctx, &pb.DeleteAccountsRequest{
		PublicKeys: make([][]byte, 1),
	})
	assert.ErrorContains(t, "No wallet nor keymanager found", err)
}

func TestServer_DeleteAccounts_OK(t *testing.T) {
	ss, pubKeys := createDirectWalletWithAccounts(t, 3)
	ctx := context.Background()
	keys, err := ss.keymanager.FetchValidatingPublicKeys(ctx)
	require.NoError(t, err)
	require.Equal(t, len(pubKeys), len(keys))

	// Next, we attempt to delete one of the keystores.
	_, err = ss.DeleteAccounts(ctx, &pb.DeleteAccountsRequest{
		PublicKeys: pubKeys[:1], // Delete the 0th public key
	})
	require.NoError(t, err)
	ss.keymanager, err = ss.wallet.InitializeKeymanager(ctx, true /* skip mnemonic confirm */)
	require.NoError(t, err)

	// We expect one of the keys to have been deleted.
	keys, err = ss.keymanager.FetchValidatingPublicKeys(ctx)
	require.NoError(t, err)
	assert.Equal(t, len(pubKeys)-1, len(keys))
}
