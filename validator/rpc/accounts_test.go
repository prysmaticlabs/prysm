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
	"time"

	"github.com/golang/mock/gomock"
	"github.com/prysmaticlabs/prysm/cmd/validator/flags"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	pb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/validator-client"
	"github.com/prysmaticlabs/prysm/testing/assert"
	mock2 "github.com/prysmaticlabs/prysm/testing/mock"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/validator/accounts"
	"github.com/prysmaticlabs/prysm/validator/accounts/iface"
	mock "github.com/prysmaticlabs/prysm/validator/accounts/testing"
	"github.com/prysmaticlabs/prysm/validator/accounts/wallet"
	"github.com/prysmaticlabs/prysm/validator/client"
	"github.com/prysmaticlabs/prysm/validator/keymanager"
	"github.com/prysmaticlabs/prysm/validator/keymanager/derived"
	constant "github.com/prysmaticlabs/prysm/validator/testing"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	defaultWalletPath = filepath.Join(flags.DefaultValidatorDir(), flags.WalletDefaultDirName)
)

func TestServer_ListAccounts(t *testing.T) {
	ctx := context.Background()
	localWalletDir := setupWalletDir(t)
	defaultWalletPath = localWalletDir
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
	km, err := w.InitializeKeymanager(ctx, iface.InitKeymanagerConfig{ListenForChanges: false})
	require.NoError(t, err)
	vs, err := client.NewValidatorService(ctx, &client.Config{
		Wallet: w,
		Validator: &mock.MockValidator{
			Km: km,
		},
	})
	require.NoError(t, err)
	s := &Server{
		walletInitialized: true,
		wallet:            w,
		validatorService:  vs,
	}
	numAccounts := 50
	dr, ok := km.(*derived.Keymanager)
	require.Equal(t, true, ok)
	err = dr.RecoverAccountsFromMnemonic(ctx, constant.TestMnemonic, "", numAccounts)
	require.NoError(t, err)
	resp, err := s.ListAccounts(ctx, &pb.ListAccountsRequest{
		PageSize: int32(numAccounts),
	})
	require.NoError(t, err)
	require.Equal(t, len(resp.Accounts), numAccounts)

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

func TestServer_BackupAccounts(t *testing.T) {
	ctx := context.Background()
	localWalletDir := setupWalletDir(t)
	defaultWalletPath = localWalletDir
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
	km, err := w.InitializeKeymanager(ctx, iface.InitKeymanagerConfig{ListenForChanges: false})
	require.NoError(t, err)
	vs, err := client.NewValidatorService(ctx, &client.Config{
		Wallet: w,
		Validator: &mock.MockValidator{
			Km: km,
		},
	})
	require.NoError(t, err)
	s := &Server{
		walletInitialized: true,
		wallet:            w,
		validatorService:  vs,
	}
	numAccounts := 50
	dr, ok := km.(*derived.Keymanager)
	require.Equal(t, true, ok)
	err = dr.RecoverAccountsFromMnemonic(ctx, constant.TestMnemonic, "", numAccounts)
	require.NoError(t, err)
	resp, err := s.ListAccounts(ctx, &pb.ListAccountsRequest{
		PageSize: int32(numAccounts),
	})
	require.NoError(t, err)
	require.Equal(t, len(resp.Accounts), numAccounts)

	pubKeys := make([][]byte, numAccounts)
	for i, aa := range resp.Accounts {
		pubKeys[i] = aa.ValidatingPublicKey
	}
	// We now attempt to backup all public keys from the wallet.
	res, err := s.BackupAccounts(context.Background(), &pb.BackupAccountsRequest{
		PublicKeys:     pubKeys,
		BackupPassword: s.wallet.Password(),
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

func TestServer_DeleteAccounts_FailedPreconditions_DerivedWallet(t *testing.T) {
	ctx := context.Background()
	localWalletDir := setupWalletDir(t)
	defaultWalletPath = localWalletDir
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
	km, err := w.InitializeKeymanager(ctx, iface.InitKeymanagerConfig{ListenForChanges: false})
	require.NoError(t, err)
	vs, err := client.NewValidatorService(ctx, &client.Config{
		Wallet: w,
		Validator: &mock.MockValidator{
			Km: km,
		},
	})
	require.NoError(t, err)
	s := &Server{
		walletInitialized: true,
		wallet:            w,
		validatorService:  vs,
	}
	numAccounts := 5
	dr, ok := km.(*derived.Keymanager)
	require.Equal(t, true, ok)
	err = dr.RecoverAccountsFromMnemonic(ctx, constant.TestMnemonic, "", numAccounts)
	require.NoError(t, err)

	_, err = s.DeleteAccounts(ctx, &pb.DeleteAccountsRequest{
		PublicKeysToDelete: nil,
	})
	assert.ErrorContains(t, "No public keys specified to delete", err)
	ikm, err := s.validatorService.Keymanager()
	require.NoError(t, err)
	keys, err := ikm.FetchValidatingPublicKeys(ctx)
	require.NoError(t, err)
	_, err = s.DeleteAccounts(ctx, &pb.DeleteAccountsRequest{
		PublicKeysToDelete: bytesutil.FromBytes48Array(keys),
	})
	require.NoError(t, err)
}

func TestServer_DeleteAccounts_FailedPreconditions_NoWallet(t *testing.T) {
	ctx := context.Background()
	localWalletDir := setupWalletDir(t)
	defaultWalletPath = localWalletDir
	w, err := accounts.CreateWalletWithKeymanager(ctx, &accounts.CreateWalletConfig{
		WalletCfg: &wallet.Config{
			WalletDir:      defaultWalletPath,
			KeymanagerKind: keymanager.Derived,
			WalletPassword: strongPass,
		},
		SkipMnemonicConfirm: true,
	})
	require.NoError(t, err)
	km, err := w.InitializeKeymanager(ctx, iface.InitKeymanagerConfig{ListenForChanges: false})
	require.NoError(t, err)
	vs, err := client.NewValidatorService(ctx, &client.Config{
		Wallet: w,
		Validator: &mock.MockValidator{
			Km: km,
		},
	})
	require.NoError(t, err)
	s := &Server{
		validatorService: vs,
	}
	_, err = s.DeleteAccounts(ctx, &pb.DeleteAccountsRequest{})
	assert.ErrorContains(t, "No public keys specified to delete", err)
	_, err = s.DeleteAccounts(ctx, &pb.DeleteAccountsRequest{
		PublicKeysToDelete: make([][]byte, 1),
	})
	assert.ErrorContains(t, "No wallet found", err)
}

func TestServer_DeleteAccounts_OK_ImportedWallet(t *testing.T) {
	s, pubKeys := createImportedWalletWithAccounts(t, 3)
	ctx := context.Background()
	ikm, err := s.validatorService.Keymanager()
	require.NoError(t, err)
	keys, err := ikm.FetchValidatingPublicKeys(ctx)
	require.NoError(t, err)
	require.Equal(t, len(pubKeys), len(keys))

	// Next, we attempt to delete one of the keystores.
	_, err = s.DeleteAccounts(ctx, &pb.DeleteAccountsRequest{
		PublicKeysToDelete: pubKeys[:1], // Delete the 0th public key
	})
	require.NoError(t, err)
	km, err := s.wallet.InitializeKeymanager(ctx, iface.InitKeymanagerConfig{ListenForChanges: false})
	require.NoError(t, err)
	// We expect one of the keys to have been deleted.
	keys, err = km.FetchValidatingPublicKeys(ctx)
	require.NoError(t, err)
	assert.Equal(t, len(pubKeys)-1, len(keys))
}

func TestServer_VoluntaryExit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	ctx := context.Background()
	mockValidatorClient := mock2.NewMockBeaconNodeValidatorClient(ctrl)
	mockNodeClient := mock2.NewMockNodeClient(ctrl)

	mockValidatorClient.EXPECT().
		ValidatorIndex(gomock.Any(), gomock.Any()).
		Return(&ethpb.ValidatorIndexResponse{Index: 0}, nil)

	mockValidatorClient.EXPECT().
		ValidatorIndex(gomock.Any(), gomock.Any()).
		Return(&ethpb.ValidatorIndexResponse{Index: 1}, nil)

	// Any time in the past will suffice
	genesisTime := &timestamppb.Timestamp{
		Seconds: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
	}

	mockNodeClient.EXPECT().
		GetGenesis(gomock.Any(), gomock.Any()).
		Times(2).
		Return(&ethpb.Genesis{GenesisTime: genesisTime}, nil)

	mockValidatorClient.EXPECT().
		DomainData(gomock.Any(), gomock.Any()).
		Times(2).
		Return(&ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}, nil)

	mockValidatorClient.EXPECT().
		ProposeExit(gomock.Any(), gomock.AssignableToTypeOf(&ethpb.SignedVoluntaryExit{})).
		Times(2).
		Return(&ethpb.ProposeExitResponse{}, nil)

	localWalletDir := setupWalletDir(t)
	defaultWalletPath = localWalletDir
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
	km, err := w.InitializeKeymanager(ctx, iface.InitKeymanagerConfig{ListenForChanges: false})
	require.NoError(t, err)
	require.NoError(t, err)
	vs, err := client.NewValidatorService(ctx, &client.Config{
		Wallet: w,
		Validator: &mock.MockValidator{
			Km: km,
		},
	})
	require.NoError(t, err)
	s := &Server{
		walletInitialized:         true,
		wallet:                    w,
		beaconNodeClient:          mockNodeClient,
		beaconNodeValidatorClient: mockValidatorClient,
		validatorService:          vs,
	}
	numAccounts := 2
	dr, ok := km.(*derived.Keymanager)
	require.Equal(t, true, ok)
	err = dr.RecoverAccountsFromMnemonic(ctx, constant.TestMnemonic, "", numAccounts)
	require.NoError(t, err)
	pubKeys, err := dr.FetchValidatingPublicKeys(ctx)
	require.NoError(t, err)

	rawPubKeys := make([][]byte, len(pubKeys))
	for i, key := range pubKeys {
		rawPubKeys[i] = key[:]
	}
	res, err := s.VoluntaryExit(ctx, &pb.VoluntaryExitRequest{
		PublicKeys: rawPubKeys,
	})
	require.NoError(t, err)
	require.DeepEqual(t, rawPubKeys, res.ExitedKeys)
}
