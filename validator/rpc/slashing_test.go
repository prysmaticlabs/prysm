package rpc

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/golang/protobuf/ptypes/empty"
	pb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1/validator-client"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/validator/accounts"
	"github.com/prysmaticlabs/prysm/v3/validator/accounts/wallet"
	"github.com/prysmaticlabs/prysm/v3/validator/db/kv"
	"github.com/prysmaticlabs/prysm/v3/validator/keymanager"
	"github.com/prysmaticlabs/prysm/v3/validator/slashing-protection-history/format"
	mocks "github.com/prysmaticlabs/prysm/v3/validator/testing"
)

func TestImportSlashingProtection_Preconditions(t *testing.T) {
	ctx := context.Background()
	localWalletDir := setupWalletDir(t)
	defaultWalletPath = localWalletDir

	// Empty JSON.
	req := &pb.ImportSlashingProtectionRequest{
		SlashingProtectionJson: "",
	}
	s := &Server{
		walletDir: defaultWalletPath,
	}

	// No validator DB provided.
	_, err := s.ImportSlashingProtection(ctx, req)
	require.ErrorContains(t, "err finding validator database at path", err)

	// Create Wallet and add to server for more realistic testing.
	w, err := accounts.CreateWalletWithKeymanager(ctx, &accounts.CreateWalletConfig{
		WalletCfg: &wallet.Config{
			WalletDir:      defaultWalletPath,
			KeymanagerKind: keymanager.Local,
			WalletPassword: strongPass,
		},
		SkipMnemonicConfirm: true,
	})
	require.NoError(t, err)
	s.wallet = w

	numValidators := 1
	// Create public keys for the mock validator DB.
	pubKeys, err := mocks.CreateRandomPubKeys(numValidators)
	require.NoError(t, err)

	// Create a validator database.
	validatorDB, err := kv.NewKVStore(ctx, defaultWalletPath, &kv.Config{
		PubKeys: pubKeys,
	})
	require.NoError(t, err)
	s.valDB = validatorDB

	// Have to close it after import is done otherwise it complains db is not open.
	defer func() {
		require.NoError(t, validatorDB.Close())
	}()

	// Test empty JSON.
	_, err = s.ImportSlashingProtection(ctx, req)
	require.ErrorContains(t, "empty slashing_protection json specified", err)

	// Generate mock slashing history.
	attestingHistory := make([][]*kv.AttestationRecord, 0)
	proposalHistory := make([]kv.ProposalHistoryForPubkey, len(pubKeys))
	for i := 0; i < len(pubKeys); i++ {
		proposalHistory[i].Proposals = make([]kv.Proposal, 0)
	}
	mockJSON, err := mocks.MockSlashingProtectionJSON(pubKeys, attestingHistory, proposalHistory)
	require.NoError(t, err)

	// JSON encode the protection JSON and save it in rpc req.
	encoded, err := json.Marshal(mockJSON)
	require.NoError(t, err)
	req.SlashingProtectionJson = string(encoded)

	_, err = s.ImportSlashingProtection(ctx, req)
	require.NoError(t, err)
}

func TestExportSlashingProtection_Preconditions(t *testing.T) {
	ctx := context.Background()
	localWalletDir := setupWalletDir(t)
	defaultWalletPath = localWalletDir

	s := &Server{
		walletDir: defaultWalletPath,
	}
	// No validator DB provided.
	_, err := s.ExportSlashingProtection(ctx, &empty.Empty{})
	require.ErrorContains(t, "err finding validator database at path", err)

	numValidators := 10
	// Create public keys for the mock validator DB.
	pubKeys, err := mocks.CreateRandomPubKeys(numValidators)
	require.NoError(t, err)

	// We create a validator database.
	validatorDB, err := kv.NewKVStore(ctx, defaultWalletPath, &kv.Config{
		PubKeys: pubKeys,
	})
	require.NoError(t, err)
	s.valDB = validatorDB

	// Have to close it after export is done otherwise it complains db is not open.
	defer func() {
		require.NoError(t, validatorDB.Close())
	}()
	genesisValidatorsRoot := [32]byte{1}
	err = validatorDB.SaveGenesisValidatorsRoot(ctx, genesisValidatorsRoot[:])
	require.NoError(t, err)

	_, err = s.ExportSlashingProtection(ctx, &empty.Empty{})
	require.NoError(t, err)
}

func TestImportExportSlashingProtection_RoundTrip(t *testing.T) {
	ctx := context.Background()
	localWalletDir := setupWalletDir(t)
	defaultWalletPath = localWalletDir

	s := &Server{
		walletDir: defaultWalletPath,
	}

	numValidators := 10
	// Create public keys for the mock validator DB.
	pubKeys, err := mocks.CreateRandomPubKeys(numValidators)
	require.NoError(t, err)

	// Create a validator database.
	validatorDB, err := kv.NewKVStore(ctx, defaultWalletPath, &kv.Config{
		PubKeys: pubKeys,
	})
	require.NoError(t, err)
	s.valDB = validatorDB

	// Have to close it after import is done otherwise it complains db is not open.
	defer func() {
		require.NoError(t, validatorDB.Close())
	}()

	// Generate mock slashing history.
	attestingHistory := make([][]*kv.AttestationRecord, 0)
	proposalHistory := make([]kv.ProposalHistoryForPubkey, len(pubKeys))
	for i := 0; i < len(pubKeys); i++ {
		proposalHistory[i].Proposals = make([]kv.Proposal, 0)
	}
	mockJSON, err := mocks.MockSlashingProtectionJSON(pubKeys, attestingHistory, proposalHistory)
	require.NoError(t, err)

	// JSON encode the protection JSON and save it in rpc req.
	encoded, err := json.Marshal(mockJSON)
	require.NoError(t, err)
	req := &pb.ImportSlashingProtectionRequest{
		SlashingProtectionJson: string(encoded),
	}

	_, err = s.ImportSlashingProtection(ctx, req)
	require.NoError(t, err)

	reqE, err := s.ExportSlashingProtection(ctx, &empty.Empty{})
	require.NoError(t, err)

	// Attempt to read the exported data and convert from string to EIP-3076.
	enc := []byte(reqE.File)

	receivedJSON := &format.EIPSlashingProtectionFormat{}
	err = json.Unmarshal(enc, receivedJSON)
	require.NoError(t, err)

	require.DeepEqual(t, mockJSON.Metadata, receivedJSON.Metadata)
}
