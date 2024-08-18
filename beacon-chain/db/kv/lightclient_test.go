package kv

import (
	"context"
	ethpbv2 "github.com/prysmaticlabs/prysm/v5/proto/eth/v2"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	"testing"
)

func TestStore_LightclientUpdate_CanSaveRetrieve(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	update := &ethpbv2.LightClientUpdate{
		AttestedHeader:          nil,
		NextSyncCommittee:       nil,
		NextSyncCommitteeBranch: nil,
		FinalizedHeader:         nil,
		FinalityBranch:          nil,
		SyncAggregate:           nil,
		SignatureSlot:           7,
	}

	period := uint64(1)
	err := db.SaveLightClientUpdate(ctx, period, &ethpbv2.LightClientUpdateWithVersion{
		Version: 1,
		Data:    update,
	})
	require.NoError(t, err)

	// Retrieve the update
	retrievedUpdate, err := db.LightClientUpdate(ctx, period)
	require.NoError(t, err)
	require.Equal(t, update.SignatureSlot, retrievedUpdate.Data.SignatureSlot, "retrieved update does not match saved update")

}
