package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/golang/protobuf/ptypes/empty"
	ethpbservice "github.com/prysmaticlabs/prysm/proto/eth/service"
	validatorpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/validator-client"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/validator/accounts"
	"github.com/prysmaticlabs/prysm/validator/accounts/iface"
	"github.com/prysmaticlabs/prysm/validator/accounts/wallet"
	"github.com/prysmaticlabs/prysm/validator/db/kv"
	"github.com/prysmaticlabs/prysm/validator/keymanager"
	"github.com/prysmaticlabs/prysm/validator/keymanager/derived"
	mocks "github.com/prysmaticlabs/prysm/validator/testing"
)

func TestServer_ListKeystores(t *testing.T) {
	t.Run("wallet not ready", func(t *testing.T) {
		s := Server{}
		_, err := s.ListKeystores(context.Background(), &empty.Empty{})
		require.ErrorContains(t, "Wallet not ready", err)
	})

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

	s := &Server{
		keymanager:        km,
		walletInitialized: true,
		wallet:            w,
	}

	t.Run("no keystores found", func(t *testing.T) {
		resp, err := s.ListKeystores(context.Background(), &empty.Empty{})
		require.NoError(t, err)
		require.Equal(t, true, len(resp.Keystores) == 0)
	})

	numAccounts := 50
	dr, ok := km.(*derived.Keymanager)
	require.Equal(t, true, ok)
	err = dr.RecoverAccountsFromMnemonic(ctx, mocks.TestMnemonic, "", numAccounts)
	require.NoError(t, err)
	expectedKeys, err := dr.FetchValidatingPublicKeys(ctx)
	require.NoError(t, err)

	t.Run("returns proper data with existing keystores", func(t *testing.T) {
		resp, err := s.ListKeystores(context.Background(), &empty.Empty{})
		require.NoError(t, err)
		require.Equal(t, numAccounts, len(resp.Keystores))
		for i := 0; i < numAccounts; i++ {
			require.DeepEqual(t, expectedKeys[i][:], resp.Keystores[i].ValidatingPubkey)
			require.Equal(
				t,
				fmt.Sprintf(derived.ValidatingKeyDerivationPathTemplate, i),
				resp.Keystores[i].DerivationPath,
			)
		}
	})
}

func TestServer_ImportKeystores(t *testing.T) {
	t.Run("wallet not ready", func(t *testing.T) {
		s := Server{}
		_, err := s.ImportKeystoresStandard(context.Background(), nil)
		require.ErrorContains(t, "Wallet not ready", err)
	})

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

	s := &Server{
		keymanager:        km,
		walletInitialized: true,
		wallet:            w,
	}

	t.Run("prevents importing if faulty keystore in request", func(t *testing.T) {
		_, err := s.ImportKeystoresStandard(context.Background(), &ethpbservice.ImportKeystoresRequest{
			Keystores: []string{"hi"},
			Passwords: []string{"hi"},
		})
		require.NoError(t, err)
	})
	t.Run("returns proper statuses for keystores in request", func(t *testing.T) {

	})
}

func TestServer_DeleteKeystores(t *testing.T) {
	ctx := context.Background()
	t.Run("wallet not ready", func(t *testing.T) {
		s := Server{}
		_, err := s.DeleteKeystores(context.Background(), nil)
		require.ErrorContains(t, "Wallet not ready", err)
	})
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

	s := &Server{
		keymanager:        km,
		walletInitialized: true,
		wallet:            w,
	}
	numAccounts := 50
	dr, ok := km.(*derived.Keymanager)
	require.Equal(t, true, ok)
	err = dr.RecoverAccountsFromMnemonic(ctx, mocks.TestMnemonic, "", numAccounts)
	require.NoError(t, err)

	publicKeys, err := km.FetchValidatingPublicKeys(ctx)
	require.NoError(t, err)
	require.Equal(t, numAccounts, len(publicKeys))

	// Create a validator database.
	validatorDB, err := kv.NewKVStore(ctx, defaultWalletPath, &kv.Config{
		PubKeys: publicKeys,
	})
	require.NoError(t, err)
	s.valDB = validatorDB

	// Have to close it after import is done otherwise it complains db is not open.
	defer func() {
		require.NoError(t, validatorDB.Close())
	}()

	// Generate mock slashing history.
	attestingHistory := make([][]*kv.AttestationRecord, 0)
	proposalHistory := make([]kv.ProposalHistoryForPubkey, len(publicKeys))
	for i := 0; i < len(publicKeys); i++ {
		proposalHistory[i].Proposals = make([]kv.Proposal, 0)
	}
	mockJSON, err := mocks.MockSlashingProtectionJSON(publicKeys, attestingHistory, proposalHistory)
	require.NoError(t, err)

	// JSON encode the protection JSON and save it.
	encoded, err := json.Marshal(mockJSON)
	require.NoError(t, err)

	_, err = s.ImportSlashingProtection(ctx, &validatorpb.ImportSlashingProtectionRequest{
		SlashingProtectionJson: string(encoded),
	})
	require.NoError(t, err)
	rawPubKeys := make([][]byte, numAccounts)
	for i := 0; i < numAccounts; i++ {
		rawPubKeys[i] = publicKeys[i][:]
	}

	// Deletes properly and returns slashing protection history.
	resp, err := s.DeleteKeystores(ctx, &ethpbservice.DeleteKeystoresRequest{
		PublicKeys: rawPubKeys,
	})
	require.NoError(t, err)
	require.Equal(t, numAccounts, len(resp.Statuses))
	for _, status := range resp.Statuses {
		require.Equal(t, ethpbservice.DeletedKeystoreStatus_DELETED, status.Status)
	}
	publicKeys, err = km.FetchValidatingPublicKeys(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, len(publicKeys))
	require.Equal(t, numAccounts, len(mockJSON.Data))

	// Returns slashing protection history if already deleted.
	resp, err = s.DeleteKeystores(ctx, &ethpbservice.DeleteKeystoresRequest{
		PublicKeys: rawPubKeys,
	})
	require.NoError(t, err)
	require.Equal(t, numAccounts, len(resp.Statuses))
	for _, status := range resp.Statuses {
		require.Equal(t, ethpbservice.DeletedKeystoreStatus_NOT_FOUND, status.Status)
	}
	require.Equal(t, numAccounts, len(mockJSON.Data))
}
