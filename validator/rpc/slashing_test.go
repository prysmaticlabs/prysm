package rpc

import (
	"context"
	"encoding/json"
	"math/rand"
	"path/filepath"
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/validator/accounts/v2"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/prysmaticlabs/prysm/shared/fileutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/prysmaticlabs/prysm/validator/accounts"
	"github.com/prysmaticlabs/prysm/validator/accounts/wallet"
	"github.com/prysmaticlabs/prysm/validator/db/kv"
	"github.com/prysmaticlabs/prysm/validator/keymanager"
	mocks "github.com/prysmaticlabs/prysm/validator/testing"
)

const jsonExportFileName = "slashing_protection.json"


func TestImportSlashingProtection_Preconditions(t *testing.T) {
	ctx := context.Background()
	localWalletDir := setupWalletDir(t)
	defaultWalletPath = localWalletDir

	// Empty JSON.
	req := &pb.ImportSlashingProtectionRequest{
		SlashingProtectionJSON: "",
	}
	s := &Server{
		walletDir: defaultWalletPath,
	}

	// No validator DB provided.
	_, err := s.ImportSlashingProtection(ctx, req)
	require.ErrorContains(t, "err finding validator database at path ", err)

	// Create Wallet and add to server for more realistic testing.
	w, err := accounts.CreateWalletWithKeymanager(ctx, &accounts.CreateWalletConfig{
		WalletCfg: &wallet.Config{
			WalletDir:      defaultWalletPath,
			KeymanagerKind: keymanager.Imported,
			WalletPassword: strongPass,
		},
		SkipMnemonicConfirm: true,
	})
	require.NoError(t, err)
	s.wallet = w

	numValidators := 10
	// Create public keys for the mock validator DB
	pubKeys, err := mocks.CreateRandomPubKeys(numValidators)
	require.NoError(t, err)

	// We create a validator database.
	validatorDB, err := kv.NewKVStore(ctx, defaultWalletPath, &kv.Config{
		PubKeys: pubKeys,
	})
	require.NoError(t, err)
	s.valDB = validatorDB
	require.NoError(t, validatorDB.Close())

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
	req.SlashingProtectionJSON = string(encoded)

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
	require.ErrorContains(t, "err finding validator database at path ", err)

	// Create Wallet and add to server for more realistic testing.
	w, err := accounts.CreateWalletWithKeymanager(ctx, &accounts.CreateWalletConfig{
		WalletCfg: &wallet.Config{
			WalletDir:      defaultWalletPath,
			KeymanagerKind: keymanager.Imported,
			WalletPassword: strongPass,
		},
		SkipMnemonicConfirm: true,
	})
	require.NoError(t, err)
	s.wallet = w

	numValidators := 10
	// Create public keys for the mock validator DB
	pubKeys, err := mocks.CreateRandomPubKeys(numValidators)
	require.NoError(t, err)

	// We create a validator database.
	validatorDB, err := kv.NewKVStore(ctx, defaultWalletPath, &kv.Config{
		PubKeys: pubKeys,
	})
	require.NoError(t, err)
	s.valDB = validatorDB
	require.NoError(t, validatorDB.Close())

	// Create mock Slashing history
	attestingHistory, proposalHistory := mocks.MockAttestingAndProposalHistories(numValidators)
	require.NoError(t, err)
	mockJSON, err := mocks.MockSlashingProtectionJSON(pubKeys, attestingHistory, proposalHistory)
	require.NoError(t, err)

	// We JSON encode the protection file and save it to disk as a JSON file.
	encoded, err := json.Marshal(mockJSON)
	require.NoError(t, err)

	// Scramble the JSON
	scamEncoded := make([]byte,len(encoded))
	perm := rand.Perm(len(encoded))
	for i,v := range perm{
		scamEncoded[v]=encoded[i]
	}
	protectionFilePath := filepath.Join(defaultWalletPath, jsonExportFileName)
	err = fileutil.WriteFile(protectionFilePath, scamEncoded)
	require.NoError(t, err)
	log.Infof("%s",string(scamEncoded))
	_, err = s.ExportSlashingProtection(ctx, &empty.Empty{})
	require.NoError(t, err)

}

