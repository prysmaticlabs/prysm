package kv

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	enginev1 "github.com/prysmaticlabs/prysm/v5/proto/engine/v1"
	ethpbv1 "github.com/prysmaticlabs/prysm/v5/proto/eth/v1"
	ethpbv2 "github.com/prysmaticlabs/prysm/v5/proto/eth/v2"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
)

func TestStore_LightClientUpdate_CanSaveRetrieveAltair(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	update := &ethpbv2.LightClientUpdate{
		AttestedHeader: &ethpbv2.LightClientHeaderContainer{
			Header: &ethpbv2.LightClientHeaderContainer_HeaderAltair{
				HeaderAltair: &ethpbv2.LightClientHeader{
					Beacon: &ethpbv1.BeaconBlockHeader{
						Slot:          1,
						ProposerIndex: 1,
						ParentRoot:    []byte{1, 1, 1},
						StateRoot:     []byte{1, 1, 1},
						BodyRoot:      []byte{1, 1, 1},
					},
				},
			},
		},
		NextSyncCommittee: &ethpbv2.SyncCommittee{
			Pubkeys:         nil,
			AggregatePubkey: nil,
		},
		NextSyncCommitteeBranch: nil,
		FinalizedHeader: &ethpbv2.LightClientHeaderContainer{
			Header: &ethpbv2.LightClientHeaderContainer_HeaderAltair{
				HeaderAltair: &ethpbv2.LightClientHeader{
					Beacon: &ethpbv1.BeaconBlockHeader{
						Slot:          1,
						ProposerIndex: 1,
						ParentRoot:    []byte{1, 1, 1},
						StateRoot:     []byte{1, 1, 1},
						BodyRoot:      []byte{1, 1, 1},
					},
				},
			},
		},
		FinalityBranch: nil,
		SyncAggregate:  nil,
		SignatureSlot:  7,
	}
	period := uint64(1)
	err := db.SaveLightClientUpdate(ctx, period, &ethpbv2.LightClientUpdateWithVersion{
		Version: version.Altair,
		Data:    update,
	})
	require.NoError(t, err)

	retrievedUpdate, err := db.LightClientUpdate(ctx, period)
	require.NoError(t, err)
	require.DeepEqual(t, update, retrievedUpdate.Data, "retrieved update does not match saved update")
}

func TestStore_LightClientUpdate_CanSaveRetrieveCapella(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	update := &ethpbv2.LightClientUpdate{
		AttestedHeader: &ethpbv2.LightClientHeaderContainer{
			Header: &ethpbv2.LightClientHeaderContainer_HeaderCapella{
				HeaderCapella: &ethpbv2.LightClientHeaderCapella{
					Beacon: &ethpbv1.BeaconBlockHeader{
						Slot:          1,
						ProposerIndex: 1,
						ParentRoot:    []byte{1, 1, 1},
						StateRoot:     []byte{1, 1, 1},
						BodyRoot:      []byte{1, 1, 1},
					},
					Execution: &enginev1.ExecutionPayloadHeaderCapella{
						FeeRecipient: []byte{1, 2, 3},
					},
					ExecutionBranch: [][]byte{{1, 2, 3}, {4, 5, 6}},
				},
			},
		},
		NextSyncCommittee: &ethpbv2.SyncCommittee{
			Pubkeys:         nil,
			AggregatePubkey: nil,
		},
		NextSyncCommitteeBranch: nil,
		FinalizedHeader: &ethpbv2.LightClientHeaderContainer{
			Header: &ethpbv2.LightClientHeaderContainer_HeaderCapella{
				HeaderCapella: &ethpbv2.LightClientHeaderCapella{
					Beacon: &ethpbv1.BeaconBlockHeader{
						Slot:          1,
						ProposerIndex: 1,
						ParentRoot:    []byte{1, 1, 1},
						StateRoot:     []byte{1, 1, 1},
						BodyRoot:      []byte{1, 1, 1},
					},
					Execution:       nil,
					ExecutionBranch: nil,
				},
			},
		},
		FinalityBranch: nil,
		SyncAggregate:  nil,
		SignatureSlot:  7,
	}
	period := uint64(1)
	err := db.SaveLightClientUpdate(ctx, period, &ethpbv2.LightClientUpdateWithVersion{
		Version: version.Capella,
		Data:    update,
	})
	require.NoError(t, err)

	retrievedUpdate, err := db.LightClientUpdate(ctx, period)
	require.NoError(t, err)
	require.DeepEqual(t, update, retrievedUpdate.Data, "retrieved update does not match saved update")
}

func TestStore_LightClientUpdate_CanSaveRetrieveDeneb(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	update := &ethpbv2.LightClientUpdate{
		AttestedHeader: &ethpbv2.LightClientHeaderContainer{
			Header: &ethpbv2.LightClientHeaderContainer_HeaderDeneb{
				HeaderDeneb: &ethpbv2.LightClientHeaderDeneb{
					Beacon: &ethpbv1.BeaconBlockHeader{
						Slot:          1,
						ProposerIndex: 1,
						ParentRoot:    []byte{1, 1, 1},
						StateRoot:     []byte{1, 1, 1},
						BodyRoot:      []byte{1, 1, 1},
					},
					Execution: &enginev1.ExecutionPayloadHeaderDeneb{
						FeeRecipient: []byte{1, 2, 3},
					},
					ExecutionBranch: [][]byte{{1, 2, 3}, {4, 5, 6}},
				},
			},
		},
		NextSyncCommittee: &ethpbv2.SyncCommittee{
			Pubkeys:         nil,
			AggregatePubkey: nil,
		},
		NextSyncCommitteeBranch: nil,
		FinalizedHeader: &ethpbv2.LightClientHeaderContainer{
			Header: &ethpbv2.LightClientHeaderContainer_HeaderDeneb{
				HeaderDeneb: &ethpbv2.LightClientHeaderDeneb{
					Beacon: &ethpbv1.BeaconBlockHeader{
						Slot:          1,
						ProposerIndex: 1,
						ParentRoot:    []byte{1, 1, 1},
						StateRoot:     []byte{1, 1, 1},
						BodyRoot:      []byte{1, 1, 1},
					},
					Execution:       nil,
					ExecutionBranch: nil,
				},
			},
		},
		FinalityBranch: nil,
		SyncAggregate:  nil,
		SignatureSlot:  7,
	}
	period := uint64(1)
	err := db.SaveLightClientUpdate(ctx, period, &ethpbv2.LightClientUpdateWithVersion{
		Version: version.Deneb,
		Data:    update,
	})
	require.NoError(t, err)

	retrievedUpdate, err := db.LightClientUpdate(ctx, period)
	require.NoError(t, err)
	require.DeepEqual(t, update, retrievedUpdate.Data, "retrieved update does not match saved update")
}

func TestStore_LightClientUpdates_canRetrieveRange(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	updates := []*ethpbv2.LightClientUpdateWithVersion{
		{
			Version: 1,
			Data: &ethpbv2.LightClientUpdate{
				AttestedHeader:          nil,
				NextSyncCommittee:       nil,
				NextSyncCommitteeBranch: nil,
				FinalizedHeader:         nil,
				FinalityBranch:          nil,
				SyncAggregate:           nil,
				SignatureSlot:           7,
			},
		},
		{
			Version: 1,
			Data: &ethpbv2.LightClientUpdate{
				AttestedHeader:          nil,
				NextSyncCommittee:       nil,
				NextSyncCommitteeBranch: nil,
				FinalizedHeader:         nil,
				FinalityBranch:          nil,
				SyncAggregate:           nil,
				SignatureSlot:           8,
			},
		},
		{
			Version: 1,
			Data: &ethpbv2.LightClientUpdate{
				AttestedHeader:          nil,
				NextSyncCommittee:       nil,
				NextSyncCommitteeBranch: nil,
				FinalizedHeader:         nil,
				FinalityBranch:          nil,
				SyncAggregate:           nil,
				SignatureSlot:           9,
			},
		},
	}

	for i, update := range updates {
		err := db.SaveLightClientUpdate(ctx, uint64(i+1), update)
		require.NoError(t, err)
	}

	// Retrieve the updates
	retrievedUpdatesMap, err := db.LightClientUpdates(ctx, 1, 3)
	require.NoError(t, err)
	require.Equal(t, len(updates), len(retrievedUpdatesMap), "retrieved updates do not match saved updates")
	for i, update := range updates {
		require.Equal(t, update.Data.SignatureSlot, retrievedUpdatesMap[uint64(i+1)].Data.SignatureSlot, "retrieved update does not match saved update")
	}

}

func TestStore_LightClientUpdate_EndPeriodSmallerThanStartPeriod(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	updates := []*ethpbv2.LightClientUpdateWithVersion{
		{
			Version: 1,
			Data: &ethpbv2.LightClientUpdate{
				AttestedHeader:          nil,
				NextSyncCommittee:       nil,
				NextSyncCommitteeBranch: nil,
				FinalizedHeader:         nil,
				FinalityBranch:          nil,
				SyncAggregate:           nil,
				SignatureSlot:           7,
			},
		},
		{
			Version: 1,
			Data: &ethpbv2.LightClientUpdate{
				AttestedHeader:          nil,
				NextSyncCommittee:       nil,
				NextSyncCommitteeBranch: nil,
				FinalizedHeader:         nil,
				FinalityBranch:          nil,
				SyncAggregate:           nil,
				SignatureSlot:           8,
			},
		},
		{
			Version: 1,
			Data: &ethpbv2.LightClientUpdate{
				AttestedHeader:          nil,
				NextSyncCommittee:       nil,
				NextSyncCommitteeBranch: nil,
				FinalizedHeader:         nil,
				FinalityBranch:          nil,
				SyncAggregate:           nil,
				SignatureSlot:           9,
			},
		},
	}

	for i, update := range updates {
		err := db.SaveLightClientUpdate(ctx, uint64(i+1), update)
		require.NoError(t, err)
	}

	// Retrieve the updates
	retrievedUpdates, err := db.LightClientUpdates(ctx, 3, 1)
	require.NotNil(t, err)
	require.Equal(t, err.Error(), "start period 3 is greater than end period 1")
	require.IsNil(t, retrievedUpdates)

}

func TestStore_LightClientUpdate_EndPeriodEqualToStartPeriod(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	updates := []*ethpbv2.LightClientUpdateWithVersion{
		{
			Version: 1,
			Data: &ethpbv2.LightClientUpdate{
				AttestedHeader:          nil,
				NextSyncCommittee:       nil,
				NextSyncCommitteeBranch: nil,
				FinalizedHeader:         nil,
				FinalityBranch:          nil,
				SyncAggregate:           nil,
				SignatureSlot:           7,
			},
		},
		{
			Version: 1,
			Data: &ethpbv2.LightClientUpdate{
				AttestedHeader:          nil,
				NextSyncCommittee:       nil,
				NextSyncCommitteeBranch: nil,
				FinalizedHeader:         nil,
				FinalityBranch:          nil,
				SyncAggregate:           nil,
				SignatureSlot:           8,
			},
		},
		{
			Version: 1,
			Data: &ethpbv2.LightClientUpdate{
				AttestedHeader:          nil,
				NextSyncCommittee:       nil,
				NextSyncCommitteeBranch: nil,
				FinalizedHeader:         nil,
				FinalityBranch:          nil,
				SyncAggregate:           nil,
				SignatureSlot:           9,
			},
		},
	}

	for i, update := range updates {
		err := db.SaveLightClientUpdate(ctx, uint64(i+1), update)
		require.NoError(t, err)
	}

	// Retrieve the updates
	retrievedUpdates, err := db.LightClientUpdates(ctx, 2, 2)
	require.NoError(t, err)
	require.Equal(t, 1, len(retrievedUpdates))
	require.Equal(t, updates[1].Data.SignatureSlot, retrievedUpdates[2].Data.SignatureSlot, "retrieved update does not match saved update")
}

func TestStore_LightClientUpdate_StartPeriodBeforeFirstUpdate(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	updates := []*ethpbv2.LightClientUpdateWithVersion{
		{
			Version: 1,
			Data: &ethpbv2.LightClientUpdate{
				AttestedHeader:          nil,
				NextSyncCommittee:       nil,
				NextSyncCommitteeBranch: nil,
				FinalizedHeader:         nil,
				FinalityBranch:          nil,
				SyncAggregate:           nil,
				SignatureSlot:           7,
			},
		},
		{
			Version: 1,
			Data: &ethpbv2.LightClientUpdate{
				AttestedHeader:          nil,
				NextSyncCommittee:       nil,
				NextSyncCommitteeBranch: nil,
				FinalizedHeader:         nil,
				FinalityBranch:          nil,
				SyncAggregate:           nil,
				SignatureSlot:           8,
			},
		},
		{
			Version: 1,
			Data: &ethpbv2.LightClientUpdate{
				AttestedHeader:          nil,
				NextSyncCommittee:       nil,
				NextSyncCommitteeBranch: nil,
				FinalizedHeader:         nil,
				FinalityBranch:          nil,
				SyncAggregate:           nil,
				SignatureSlot:           9,
			},
		},
	}

	for i, update := range updates {
		err := db.SaveLightClientUpdate(ctx, uint64(i+2), update)
		require.NoError(t, err)
	}

	// Retrieve the updates
	retrievedUpdates, err := db.LightClientUpdates(ctx, 0, 4)
	require.NoError(t, err)
	require.Equal(t, 3, len(retrievedUpdates))
	for i, update := range updates {
		require.Equal(t, update.Data.SignatureSlot, retrievedUpdates[uint64(i+2)].Data.SignatureSlot, "retrieved update does not match saved update")
	}
}

func TestStore_LightClientUpdate_EndPeriodAfterLastUpdate(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	updates := []*ethpbv2.LightClientUpdateWithVersion{
		{
			Version: 1,
			Data: &ethpbv2.LightClientUpdate{
				AttestedHeader:          nil,
				NextSyncCommittee:       nil,
				NextSyncCommitteeBranch: nil,
				FinalizedHeader:         nil,
				FinalityBranch:          nil,
				SyncAggregate:           nil,
				SignatureSlot:           7,
			},
		},
		{
			Version: 1,
			Data: &ethpbv2.LightClientUpdate{
				AttestedHeader:          nil,
				NextSyncCommittee:       nil,
				NextSyncCommitteeBranch: nil,
				FinalizedHeader:         nil,
				FinalityBranch:          nil,
				SyncAggregate:           nil,
				SignatureSlot:           8,
			},
		},
		{
			Version: 1,
			Data: &ethpbv2.LightClientUpdate{
				AttestedHeader:          nil,
				NextSyncCommittee:       nil,
				NextSyncCommitteeBranch: nil,
				FinalizedHeader:         nil,
				FinalityBranch:          nil,
				SyncAggregate:           nil,
				SignatureSlot:           9,
			},
		},
	}

	for i, update := range updates {
		err := db.SaveLightClientUpdate(ctx, uint64(i+1), update)
		require.NoError(t, err)
	}

	// Retrieve the updates
	retrievedUpdates, err := db.LightClientUpdates(ctx, 1, 6)
	require.NoError(t, err)
	require.Equal(t, 3, len(retrievedUpdates))
	for i, update := range updates {
		require.Equal(t, update.Data.SignatureSlot, retrievedUpdates[uint64(i+1)].Data.SignatureSlot, "retrieved update does not match saved update")
	}
}

func TestStore_LightClientUpdate_PartialUpdates(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	updates := []*ethpbv2.LightClientUpdateWithVersion{
		{
			Version: 1,
			Data: &ethpbv2.LightClientUpdate{
				AttestedHeader:          nil,
				NextSyncCommittee:       nil,
				NextSyncCommitteeBranch: nil,
				FinalizedHeader:         nil,
				FinalityBranch:          nil,
				SyncAggregate:           nil,
				SignatureSlot:           7,
			},
		},
		{
			Version: 1,
			Data: &ethpbv2.LightClientUpdate{
				AttestedHeader:          nil,
				NextSyncCommittee:       nil,
				NextSyncCommitteeBranch: nil,
				FinalizedHeader:         nil,
				FinalityBranch:          nil,
				SyncAggregate:           nil,
				SignatureSlot:           8,
			},
		},
		{
			Version: 1,
			Data: &ethpbv2.LightClientUpdate{
				AttestedHeader:          nil,
				NextSyncCommittee:       nil,
				NextSyncCommitteeBranch: nil,
				FinalizedHeader:         nil,
				FinalityBranch:          nil,
				SyncAggregate:           nil,
				SignatureSlot:           9,
			},
		},
	}

	for i, update := range updates {
		err := db.SaveLightClientUpdate(ctx, uint64(i+1), update)
		require.NoError(t, err)
	}

	// Retrieve the updates
	retrievedUpdates, err := db.LightClientUpdates(ctx, 1, 2)
	require.NoError(t, err)
	require.Equal(t, 2, len(retrievedUpdates))
	for i, update := range updates[:2] {
		require.Equal(t, update.Data.SignatureSlot, retrievedUpdates[uint64(i+1)].Data.SignatureSlot, "retrieved update does not match saved update")
	}
}

func TestStore_LightClientUpdate_MissingPeriods_SimpleData(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	updates := []*ethpbv2.LightClientUpdateWithVersion{
		{
			Version: 1,
			Data: &ethpbv2.LightClientUpdate{
				AttestedHeader:          nil,
				NextSyncCommittee:       nil,
				NextSyncCommitteeBranch: nil,
				FinalizedHeader:         nil,
				FinalityBranch:          nil,
				SyncAggregate:           nil,
				SignatureSlot:           7,
			},
		},
		{
			Version: 1,
			Data: &ethpbv2.LightClientUpdate{
				AttestedHeader:          nil,
				NextSyncCommittee:       nil,
				NextSyncCommitteeBranch: nil,
				FinalizedHeader:         nil,
				FinalityBranch:          nil,
				SyncAggregate:           nil,
				SignatureSlot:           8,
			},
		},
		{
			Version: 1,
			Data: &ethpbv2.LightClientUpdate{
				AttestedHeader:          nil,
				NextSyncCommittee:       nil,
				NextSyncCommitteeBranch: nil,
				FinalizedHeader:         nil,
				FinalityBranch:          nil,
				SyncAggregate:           nil,
				SignatureSlot:           11,
			},
		},
		{
			Version: 1,
			Data: &ethpbv2.LightClientUpdate{
				AttestedHeader:          nil,
				NextSyncCommittee:       nil,
				NextSyncCommitteeBranch: nil,
				FinalizedHeader:         nil,
				FinalityBranch:          nil,
				SyncAggregate:           nil,
				SignatureSlot:           12,
			},
		},
	}

	for _, update := range updates {
		err := db.SaveLightClientUpdate(ctx, uint64(update.Data.SignatureSlot), update)
		require.NoError(t, err)
	}

	// Retrieve the updates
	retrievedUpdates, err := db.LightClientUpdates(ctx, 7, 12)
	require.NoError(t, err)
	require.Equal(t, 4, len(retrievedUpdates))
	for _, update := range updates {
		require.Equal(t, update.Data.SignatureSlot, retrievedUpdates[uint64(update.Data.SignatureSlot)].Data.SignatureSlot, "retrieved update does not match saved update")
	}

	// Retrieve the updates from the middle
	retrievedUpdates, err = db.LightClientUpdates(ctx, 8, 12)
	require.NoError(t, err)
	require.Equal(t, 3, len(retrievedUpdates))
	require.Equal(t, updates[1].Data.SignatureSlot, retrievedUpdates[8].Data.SignatureSlot, "retrieved update does not match saved update")
	require.Equal(t, updates[2].Data.SignatureSlot, retrievedUpdates[11].Data.SignatureSlot, "retrieved update does not match saved update")
	require.Equal(t, updates[3].Data.SignatureSlot, retrievedUpdates[12].Data.SignatureSlot, "retrieved update does not match saved update")

	// Retrieve the updates from after the missing period
	retrievedUpdates, err = db.LightClientUpdates(ctx, 11, 12)
	require.NoError(t, err)
	require.Equal(t, 2, len(retrievedUpdates))
	require.Equal(t, updates[2].Data.SignatureSlot, retrievedUpdates[11].Data.SignatureSlot, "retrieved update does not match saved update")
	require.Equal(t, updates[3].Data.SignatureSlot, retrievedUpdates[12].Data.SignatureSlot, "retrieved update does not match saved update")

	//retrieve the updates from before the missing period to after the missing period
	retrievedUpdates, err = db.LightClientUpdates(ctx, 3, 15)
	require.NoError(t, err)
	require.Equal(t, 4, len(retrievedUpdates))
	require.Equal(t, updates[0].Data.SignatureSlot, retrievedUpdates[7].Data.SignatureSlot, "retrieved update does not match saved update")
	require.Equal(t, updates[1].Data.SignatureSlot, retrievedUpdates[8].Data.SignatureSlot, "retrieved update does not match saved update")
	require.Equal(t, updates[2].Data.SignatureSlot, retrievedUpdates[11].Data.SignatureSlot, "retrieved update does not match saved update")
	require.Equal(t, updates[3].Data.SignatureSlot, retrievedUpdates[12].Data.SignatureSlot, "retrieved update does not match saved update")
}

func TestStore_LightClientUpdate_EmptyDB(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	// Retrieve the updates
	retrievedUpdates, err := db.LightClientUpdates(ctx, 1, 3)
	require.IsNil(t, err)
	require.Equal(t, 0, len(retrievedUpdates))
}

func TestStore_LightClientUpdate_MissingPeriodsAtTheEnd_SimpleData(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	for i := 1; i < 4; i++ {
		update := &ethpbv2.LightClientUpdateWithVersion{
			Version: 1,
			Data: &ethpbv2.LightClientUpdate{
				AttestedHeader:          nil,
				NextSyncCommittee:       nil,
				NextSyncCommitteeBranch: nil,
				FinalizedHeader:         nil,
				FinalityBranch:          nil,
				SyncAggregate:           nil,
				SignatureSlot:           primitives.Slot(uint64(i)),
			},
		}
		err := db.SaveLightClientUpdate(ctx, uint64(i), update)
		require.NoError(t, err)
	}
	for i := 7; i < 10; i++ {
		update := &ethpbv2.LightClientUpdateWithVersion{
			Version: 1,
			Data: &ethpbv2.LightClientUpdate{
				AttestedHeader:          nil,
				NextSyncCommittee:       nil,
				NextSyncCommitteeBranch: nil,
				FinalizedHeader:         nil,
				FinalityBranch:          nil,
				SyncAggregate:           nil,
				SignatureSlot:           primitives.Slot(uint64(i)),
			},
		}
		err := db.SaveLightClientUpdate(ctx, uint64(i), update)
		require.NoError(t, err)
	}

	// Retrieve the updates from 1 to 5
	retrievedUpdates, err := db.LightClientUpdates(ctx, 1, 5)
	require.NoError(t, err)
	require.Equal(t, 3, len(retrievedUpdates))
	require.Equal(t, primitives.Slot(1), retrievedUpdates[1].Data.SignatureSlot, "retrieved update does not match saved update")
	require.Equal(t, primitives.Slot(2), retrievedUpdates[2].Data.SignatureSlot, "retrieved update does not match saved update")
	require.Equal(t, primitives.Slot(3), retrievedUpdates[3].Data.SignatureSlot, "retrieved update does not match saved update")

}

func setupLightClientTestDB(t *testing.T) (*Store, context.Context) {
	db := setupDB(t)
	ctx := context.Background()

	for i := 10; i < 101; i++ { // 10 to 100
		update := &ethpbv2.LightClientUpdateWithVersion{
			Version: 1,
			Data: &ethpbv2.LightClientUpdate{
				AttestedHeader:          nil,
				NextSyncCommittee:       nil,
				NextSyncCommitteeBranch: nil,
				FinalizedHeader:         nil,
				FinalityBranch:          nil,
				SyncAggregate:           nil,
				SignatureSlot:           primitives.Slot(uint64(i)),
			},
		}
		err := db.SaveLightClientUpdate(ctx, uint64(i), update)
		require.NoError(t, err)
	}

	for i := 110; i < 201; i++ { // 110 to 200
		update := &ethpbv2.LightClientUpdateWithVersion{
			Version: 1,
			Data: &ethpbv2.LightClientUpdate{
				AttestedHeader:          nil,
				NextSyncCommittee:       nil,
				NextSyncCommitteeBranch: nil,
				FinalizedHeader:         nil,
				FinalityBranch:          nil,
				SyncAggregate:           nil,
				SignatureSlot:           primitives.Slot(uint64(i)),
			},
		}
		err := db.SaveLightClientUpdate(ctx, uint64(i), update)
		require.NoError(t, err)
	}

	return db, ctx
}

func TestStore_LightClientUpdate_MissingPeriodsInTheMiddleDistributed(t *testing.T) {
	db, ctx := setupLightClientTestDB(t)

	// Retrieve the updates - should fail because of missing periods in the middle
	retrievedUpdates, err := db.LightClientUpdates(ctx, 1, 300)
	require.NoError(t, err)
	require.Equal(t, 91*2, len(retrievedUpdates))
	for i := 10; i < 101; i++ {
		require.Equal(t, primitives.Slot(uint64(i)), retrievedUpdates[uint64(i)].Data.SignatureSlot, "retrieved update does not match saved update")
	}
	for i := 110; i < 201; i++ {
		require.Equal(t, primitives.Slot(uint64(i)), retrievedUpdates[uint64(i)].Data.SignatureSlot, "retrieved update does not match saved update")
	}

}

func TestStore_LightClientUpdate_RetrieveValidRangeFromStart(t *testing.T) {
	db, ctx := setupLightClientTestDB(t)

	// retrieve 1 to 100 - should work because all periods are present after the firstPeriodInDB > startPeriod
	retrievedUpdates, err := db.LightClientUpdates(ctx, 1, 100)
	require.NoError(t, err)
	require.Equal(t, 91, len(retrievedUpdates))
	for i := 10; i < 101; i++ {
		require.Equal(t, primitives.Slot(uint64(i)), retrievedUpdates[uint64(i)].Data.SignatureSlot, "retrieved update does not match saved update")
	}
}

func TestStore_LightClientUpdate_RetrieveValidRangeInTheMiddle(t *testing.T) {
	db, ctx := setupLightClientTestDB(t)

	// retrieve 110 to 200 - should work because all periods are present
	retrievedUpdates, err := db.LightClientUpdates(ctx, 110, 200)
	require.NoError(t, err)
	require.Equal(t, 91, len(retrievedUpdates))
	for i := 110; i < 201; i++ {
		require.Equal(t, primitives.Slot(uint64(i)), retrievedUpdates[uint64(i)].Data.SignatureSlot, "retrieved update does not match saved update")
	}
}

func TestStore_LightClientUpdate_MissingPeriodInTheMiddleConcentrated(t *testing.T) {
	db, ctx := setupLightClientTestDB(t)

	// retrieve 100 to 200
	retrievedUpdates, err := db.LightClientUpdates(ctx, 100, 200)
	require.NoError(t, err)
	require.Equal(t, 92, len(retrievedUpdates))
	require.Equal(t, primitives.Slot(100), retrievedUpdates[100].Data.SignatureSlot, "retrieved update does not match saved update")
	for i := 110; i < 201; i++ {
		require.Equal(t, primitives.Slot(uint64(i)), retrievedUpdates[uint64(i)].Data.SignatureSlot, "retrieved update does not match saved update")
	}
}

func TestStore_LightClientUpdate_MissingPeriodsAtTheEnd(t *testing.T) {
	db, ctx := setupLightClientTestDB(t)

	// retrieve 10 to 109
	retrievedUpdates, err := db.LightClientUpdates(ctx, 10, 109)
	require.NoError(t, err)
	require.Equal(t, 91, len(retrievedUpdates))
	for i := 10; i < 101; i++ {
		require.Equal(t, primitives.Slot(uint64(i)), retrievedUpdates[uint64(i)].Data.SignatureSlot, "retrieved update does not match saved update")
	}
}

func TestStore_LightClientUpdate_MissingPeriodsAtTheBeginning(t *testing.T) {
	db, ctx := setupLightClientTestDB(t)

	// retrieve 105 to 200
	retrievedUpdates, err := db.LightClientUpdates(ctx, 105, 200)
	require.NoError(t, err)
	require.Equal(t, 91, len(retrievedUpdates))
	for i := 110; i < 201; i++ {
		require.Equal(t, primitives.Slot(uint64(i)), retrievedUpdates[uint64(i)].Data.SignatureSlot, "retrieved update does not match saved update")
	}
}

func TestStore_LightClientUpdate_StartPeriodGreaterThanLastPeriod(t *testing.T) {
	db, ctx := setupLightClientTestDB(t)

	// retrieve 300 to 400
	retrievedUpdates, err := db.LightClientUpdates(ctx, 300, 400)
	require.NoError(t, err)
	require.Equal(t, 0, len(retrievedUpdates))

}
