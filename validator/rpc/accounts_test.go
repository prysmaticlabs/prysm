package rpc

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	pb "github.com/prysmaticlabs/prysm/proto/validator/accounts/v2"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/prysmaticlabs/prysm/validator/accounts"
	"github.com/prysmaticlabs/prysm/validator/accounts/wallet"
	"github.com/prysmaticlabs/prysm/validator/flags"
	"github.com/prysmaticlabs/prysm/validator/keymanager"
	"github.com/prysmaticlabs/prysm/validator/keymanager/derived"
)

var defaultWalletPath = filepath.Join(flags.DefaultValidatorDir(), flags.WalletDefaultDirName)
var _ accountCreator = (*mockAccountCreator)(nil)

type mockAccountCreator struct {
	data   *ethpb.Deposit_Data
	pubKey []byte
}

func (m *mockAccountCreator) CreateAccount(ctx context.Context) ([]byte, *ethpb.Deposit_Data, error) {
	return m.pubKey, m.data, nil
}

func TestServer_CreateAccount(t *testing.T) {
	ctx := context.Background()
	localWalletDir := setupWalletDir(t)
	defaultWalletPath = localWalletDir
	strongPass := "29384283xasjasd32%%&*@*#*"
	// We attempt to create the wallet.
	w, err := accounts.CreateWalletWithKeymanager(ctx, &accounts.CreateWalletConfig{
		WalletCfg: &wallet.Config{
			WalletDir:      defaultWalletPath,
			KeymanagerKind: keymanager.Derived,
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
	w, err := accounts.CreateWalletWithKeymanager(ctx, &accounts.CreateWalletConfig{
		WalletCfg: &wallet.Config{
			WalletDir:      defaultWalletPath,
			KeymanagerKind: keymanager.Derived,
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
	numAccounts := 50
	keys := make([][]byte, numAccounts)
	for i := 0; i < numAccounts; i++ {
		key, _, err := km.(*derived.Keymanager).CreateAccount(ctx)
		require.NoError(t, err)
		keys[i] = key
	}
	resp, err := s.ListAccounts(ctx, &pb.ListAccountsRequest{
		PageSize: int32(numAccounts),
	})
	require.NoError(t, err)
	require.Equal(t, len(resp.Accounts), numAccounts)
	for i := 0; i < numAccounts; i++ {
		assert.DeepEqual(t, resp.Accounts[i].ValidatingPublicKey, keys[i])
	}

	tests := []struct {
		req *pb.ListAccountsRequest
		res *pb.ListAccountsResponse
	}{
		{
			req: &pb.ListAccountsRequest{
				PageSize: 5,
			},
			res: &pb.ListAccountsResponse{
				Accounts:      resp.Accounts[0:5],
				NextPageToken: "1",
				TotalSize:     int32(numAccounts),
			},
		},
		{
			req: &pb.ListAccountsRequest{
				PageSize:  5,
				PageToken: "1",
			},
			res: &pb.ListAccountsResponse{
				Accounts:      resp.Accounts[5:10],
				NextPageToken: "2",
				TotalSize:     int32(numAccounts),
			},
		},
	}
	for _, test := range tests {
		res, err := s.ListAccounts(context.Background(), test.req)
		require.NoError(t, err)
		assert.DeepEqual(t, res, test.res)
	}
}

func Test_createAccountWithDepositData(t *testing.T) {
	ctx := context.Background()
	pubKey := bls.RandKey().PublicKey().Marshal()
	m := &mockAccountCreator{
		data: &ethpb.Deposit_Data{
			PublicKey:             pubKey,
			WithdrawalCredentials: make([]byte, 32),
			Amount:                params.BeaconConfig().MaxEffectiveBalance,
			Signature:             make([]byte, 96),
		},
		pubKey: pubKey,
	}
	rawResp, err := createAccountWithDepositData(ctx, m)
	require.NoError(t, err)
	assert.DeepEqual(
		t, rawResp.Data["pubkey"], fmt.Sprintf("%x", pubKey),
	)
	assert.DeepEqual(
		t, rawResp.Data["withdrawal_credentials"], fmt.Sprintf("%x", make([]byte, 32)),
	)
	assert.DeepEqual(
		t, rawResp.Data["amount"], fmt.Sprintf("%d", params.BeaconConfig().MaxEffectiveBalance),
	)
	assert.DeepEqual(
		t, rawResp.Data["signature"], fmt.Sprintf("%x", make([]byte, 96)),
	)
	assert.DeepEqual(
		t, rawResp.Data["fork_version"], fmt.Sprintf("%x", params.BeaconConfig().GenesisForkVersion),
	)
}

func TestServer_BackupAccounts(t *testing.T) {
	ss, pubKeys := createImportedWalletWithAccounts(t, 3)

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
		keystore := &keymanager.Keystore{}
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
	w, err := accounts.CreateWalletWithKeymanager(ctx, &accounts.CreateWalletConfig{
		WalletCfg: &wallet.Config{
			WalletDir:      defaultWalletPath,
			KeymanagerKind: keymanager.Derived,
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
	assert.ErrorContains(t, "Only imported wallets can delete accounts", err)
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
	ss, pubKeys := createImportedWalletWithAccounts(t, 3)
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
