package rpc

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	pb "github.com/prysmaticlabs/prysm/proto/validator/accounts/v2"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/prysmaticlabs/prysm/validator/accounts"
	"github.com/prysmaticlabs/prysm/validator/accounts/iface"
	"github.com/prysmaticlabs/prysm/validator/accounts/wallet"
	"github.com/prysmaticlabs/prysm/validator/flags"
	"github.com/prysmaticlabs/prysm/validator/keymanager"
	"github.com/prysmaticlabs/prysm/validator/keymanager/derived"
)

var defaultWalletPath = filepath.Join(flags.DefaultValidatorDir(), flags.WalletDefaultDirName)

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
	km, err := w.InitializeKeymanager(ctx)
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
	km, err := w.InitializeKeymanager(ctx)
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
	secretKey, err := bls.RandKey()
	require.NoError(t, err)
	pubKey := secretKey.PublicKey().Marshal()
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
