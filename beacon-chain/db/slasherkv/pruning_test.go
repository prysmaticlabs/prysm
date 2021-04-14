package slasherkv

import (
	"context"
	"fmt"
	"io/ioutil"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	slashertypes "github.com/prysmaticlabs/prysm/beacon-chain/slasher/types"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/sirupsen/logrus"
	logTest "github.com/sirupsen/logrus/hooks/test"
	bolt "go.etcd.io/bbolt"
)

func TestMain(m *testing.M) {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(ioutil.Discard)
	m.Run()
}

func TestStore_PruneProposals_OK(t *testing.T) {
	ctx := context.Background()

	// If the current slot is less than the history length we track in slasher, there
	// is nothing to prune yet, so we expect exiting early.
	t.Run("current_epoch_less_than_history_length", func(t *testing.T) {
		hook := logTest.NewGlobal()
		beaconDB := setupDB(t)
		epochPruningIncrements := types.Epoch(100)
		currentEpoch := types.Epoch(1)
		historyLength := types.Epoch(2)
		err := beaconDB.PruneProposals(ctx, currentEpoch, epochPruningIncrements, historyLength)
		require.NoError(t, err)
		require.LogsContain(t, hook, "Current epoch 1 < history length 2, nothing to prune")
	})

	// If the lowest stored epoch in the database is >= the end epoch of the pruning process,
	// there is nothing to prune, so we also expect exiting early.
	t.Run("lowest_stored_epoch_greater_than_pruning_limit_epoch", func(t *testing.T) {
		hook := logTest.NewGlobal()
		beaconDB := setupDB(t)
		epochPruningIncrements := types.Epoch(100)

		// With a current epoch of 20 and a history length of 10, we should be pruning
		// everything before epoch (20 - 10) = 10.
		currentEpoch := types.Epoch(20)
		historyLength := types.Epoch(10)

		pruningLimitEpoch := currentEpoch - historyLength
		lowestStoredSlot, err := helpers.StartSlot(pruningLimitEpoch)
		require.NoError(t, err)

		err = beaconDB.db.Update(func(tx *bolt.Tx) error {
			bkt := tx.Bucket(proposalRecordsBucket)
			key, err := keyForValidatorProposal(&slashertypes.SignedBlockHeaderWrapper{
				SignedBeaconBlockHeader: &ethpb.SignedBeaconBlockHeader{
					Header: &ethpb.BeaconBlockHeader{
						Slot:          lowestStoredSlot,
						ProposerIndex: 0,
					},
				},
			})
			if err != nil {
				return err
			}
			return bkt.Put(key, []byte("hi"))
		})
		require.NoError(t, err)

		err = beaconDB.PruneProposals(ctx, currentEpoch, epochPruningIncrements, historyLength)
		require.NoError(t, err)
		expectedLog := fmt.Sprintf(
			"Lowest slot %d is >= pruning slot %d, nothing to prune", lowestStoredSlot, lowestStoredSlot,
		)
		require.LogsContain(t, hook, expectedLog)
	})

	// Prune in increments until the cursor reaches the pruning limit epoch, and we expect
	// that all the proposals written are deleted from the database, while those above the
	// pruning limit epoch are kept intact.
	t.Run("prune_in_full_increments_and_verify_deletions", func(t *testing.T) {
		hook := logTest.NewGlobal()
		beaconDB := setupDB(t)

		config := params.BeaconConfig()
		copyConfig := config.Copy()
		copyConfig.SlotsPerEpoch = 2
		params.OverrideBeaconConfig(copyConfig)
		defer params.OverrideBeaconConfig(config)

		epochPruningIncrements := types.Epoch(2)
		historyLength := types.Epoch(10)
		currentEpoch := types.Epoch(20)
		pruningLimitEpoch := currentEpoch - historyLength

		// We create proposals from genesis to the current epoch, with 2 proposals
		// at each slot to ensure the entire pruning logic works correctly.
		slotsPerEpoch := params.BeaconConfig().SlotsPerEpoch
		proposals := make([]*slashertypes.SignedBlockHeaderWrapper, 0, uint64(currentEpoch)*uint64(slotsPerEpoch)*2)
		t.Log("Creating proposals")
		for i := types.Epoch(0); i < currentEpoch; i++ {
			startSlot, err := helpers.StartSlot(i)
			require.NoError(t, err)
			for j := startSlot; j < slotsPerEpoch; j++ {
				prop1 := createProposalWrapper(t, j, 0 /* proposer index */, []byte{0})
				prop2 := createProposalWrapper(t, j, 2 /* proposer index */, []byte{1})
				proposals = append(proposals, prop1, prop2)
			}
		}
		t.Log("Finished proposals")

		t.Log("Saving proposals")
		require.NoError(t, beaconDB.SaveBlockProposals(ctx, proposals))
		t.Log("Saved proposals")

		t.Log("Pruning proposals")
		// We expect pruning completes without an issue and properly logs progress.
		err := beaconDB.PruneProposals(ctx, currentEpoch, epochPruningIncrements, historyLength)
		require.NoError(t, err)
		t.Log("Pruned proposals")

		for i := types.Epoch(0); i < pruningLimitEpoch; i += epochPruningIncrements {
			wantedLog := fmt.Sprintf("Pruned %d/%d epochs worth of proposals", i, pruningLimitEpoch)
			require.LogsContain(t, hook, wantedLog)
		}
	})
}

func TestStore_PruneProposals_Logs(t *testing.T) {
	ctx := context.Background()
	proposalsInDB := []*slashertypes.SignedBlockHeaderWrapper{
		createProposalWrapper(t, types.Slot(2), 0, []byte{1}),
		createProposalWrapper(t, types.Slot(8), 0, []byte{1}),
		createProposalWrapper(t, params.BeaconConfig().SlotsPerEpoch*3, 0, []byte{1}),
	}
	afterPruning := []*slashertypes.SignedBlockHeaderWrapper{
		createProposalWrapper(t, params.BeaconConfig().SlotsPerEpoch*3, 0, []byte{1}),
	}
	epoch := types.Epoch(2)
	beaconDB := setupDB(t)
	require.NoError(t, beaconDB.SaveBlockProposals(ctx, proposalsInDB))

	epochPruningIncrements := types.Epoch(100)
	err := beaconDB.PruneProposals(ctx, epoch, epochPruningIncrements, defaultHistoryLength)

	// Second time checking same proposals but all with different signing root should
	// return all double proposals.
	slashable := make([]*slashertypes.SignedBlockHeaderWrapper, len(afterPruning))
	for i := 0; i < len(afterPruning); i++ {
		slashable[i] = createProposalWrapper(t, afterPruning[i].SignedBeaconBlockHeader.Header.Slot, afterPruning[i].SignedBeaconBlockHeader.Header.ProposerIndex, []byte{2})
	}

	doubleProposals, err := beaconDB.CheckDoubleBlockProposals(ctx, slashable)
	require.NoError(t, err)
	require.Equal(t, len(afterPruning), len(doubleProposals))
	for i, existing := range doubleProposals {
		require.DeepSSZEqual(t, existing.Header_1, afterPruning[i].SignedBeaconBlockHeader)
	}
}

func TestStore_PruneProposals(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name          string
		proposalsInDB []*slashertypes.SignedBlockHeaderWrapper
		afterPruning  []*slashertypes.SignedBlockHeaderWrapper
		epoch         types.Epoch
		historyLength types.Epoch
		wantErr       bool
	}{
		{
			name: "should delete all proposals under epoch 2",
			proposalsInDB: []*slashertypes.SignedBlockHeaderWrapper{
				createProposalWrapper(t, types.Slot(2), 0, []byte{1}),
				createProposalWrapper(t, types.Slot(8), 0, []byte{1}),
				createProposalWrapper(t, params.BeaconConfig().SlotsPerEpoch*3, 0, []byte{1}),
			},
			afterPruning: []*slashertypes.SignedBlockHeaderWrapper{
				createProposalWrapper(t, params.BeaconConfig().SlotsPerEpoch*3, 0, []byte{1}),
			},
			epoch: 2,
		},
		{
			name: "should delete all proposals under epoch 5 - historyLength 3",
			proposalsInDB: []*slashertypes.SignedBlockHeaderWrapper{
				createProposalWrapper(t, types.Slot(2), 0, []byte{1}),
				createProposalWrapper(t, types.Slot(8), 0, []byte{1}),
				createProposalWrapper(t, params.BeaconConfig().SlotsPerEpoch*3, 0, []byte{1}),
				createProposalWrapper(t, params.BeaconConfig().SlotsPerEpoch*4, 0, []byte{1}),
				createProposalWrapper(t, params.BeaconConfig().SlotsPerEpoch*5, 0, []byte{1}),
			},
			afterPruning: []*slashertypes.SignedBlockHeaderWrapper{
				createProposalWrapper(t, params.BeaconConfig().SlotsPerEpoch*3, 0, []byte{1}),
				createProposalWrapper(t, params.BeaconConfig().SlotsPerEpoch*4, 0, []byte{1}),
				createProposalWrapper(t, params.BeaconConfig().SlotsPerEpoch*5, 0, []byte{1}),
			},
			historyLength: 3,
			epoch:         5,
		},
		{
			name: "should delete all proposals under epoch 4",
			proposalsInDB: []*slashertypes.SignedBlockHeaderWrapper{
				createProposalWrapper(t, params.BeaconConfig().SlotsPerEpoch*0, 0, []byte{1}),
				createProposalWrapper(t, params.BeaconConfig().SlotsPerEpoch*1, 0, []byte{1}),
				createProposalWrapper(t, params.BeaconConfig().SlotsPerEpoch*2, 0, []byte{1}),
				createProposalWrapper(t, params.BeaconConfig().SlotsPerEpoch*3, 0, []byte{1}),
			},
			afterPruning: []*slashertypes.SignedBlockHeaderWrapper{},
			epoch:        4,
		},
		{
			name: "no proposal to delete under epoch 1",
			proposalsInDB: []*slashertypes.SignedBlockHeaderWrapper{
				createProposalWrapper(t, params.BeaconConfig().SlotsPerEpoch*2, 0, []byte{1}),
				createProposalWrapper(t, params.BeaconConfig().SlotsPerEpoch*3, 0, []byte{1}),
			},
			afterPruning: []*slashertypes.SignedBlockHeaderWrapper{
				createProposalWrapper(t, params.BeaconConfig().SlotsPerEpoch*2, 0, []byte{1}),
				createProposalWrapper(t, params.BeaconConfig().SlotsPerEpoch*3, 0, []byte{1}),
			},
			epoch: 5,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			epochPruningIncrements := types.Epoch(100)
			beaconDB := setupDB(t)
			require.NoError(t, beaconDB.SaveBlockProposals(ctx, tt.proposalsInDB))
			if err := beaconDB.PruneProposals(ctx, tt.epoch, epochPruningIncrements, tt.historyLength); (err != nil) != tt.wantErr {
				t.Errorf("PruneProposals() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Second time checking same proposals but all with different signing root should
			// return all double proposals.
			slashable := make([]*slashertypes.SignedBlockHeaderWrapper, len(tt.afterPruning))
			for i := 0; i < len(tt.afterPruning); i++ {
				slashable[i] = createProposalWrapper(t, tt.afterPruning[i].SignedBeaconBlockHeader.Header.Slot, tt.afterPruning[i].SignedBeaconBlockHeader.Header.ProposerIndex, []byte{2})
			}

			doubleProposals, err := beaconDB.CheckDoubleBlockProposals(ctx, slashable)
			require.NoError(t, err)
			require.Equal(t, len(tt.afterPruning), len(doubleProposals))
			for i, existing := range doubleProposals {
				require.DeepSSZEqual(t, existing.Header_1, tt.afterPruning[i].SignedBeaconBlockHeader)
			}
		})
	}
}

func TestStore_PruneAttestations(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name             string
		attestationsInDB []*slashertypes.IndexedAttestationWrapper
		afterPruning     []*slashertypes.IndexedAttestationWrapper
		epoch            types.Epoch
		historyLength    types.Epoch
		wantErr          bool
	}{
		{
			name: "should delete all attestations under epoch 2",
			attestationsInDB: []*slashertypes.IndexedAttestationWrapper{
				createAttestationWrapper(0, 0, []uint64{1}, []byte{1}),
				createAttestationWrapper(0, 1, []uint64{1}, []byte{1}),
				createAttestationWrapper(0, 2, []uint64{1}, []byte{1}),
			},
			afterPruning: []*slashertypes.IndexedAttestationWrapper{
				createAttestationWrapper(0, 1, []uint64{1}, []byte{1}),
				createAttestationWrapper(0, 2, []uint64{1}, []byte{1}),
			},
			epoch: 5000,
		},
		{
			name: "should delete all attestations under epoch 6 - 2 historyLength",
			attestationsInDB: []*slashertypes.IndexedAttestationWrapper{
				createAttestationWrapper(0, 1, []uint64{1}, []byte{1}),
				createAttestationWrapper(0, 2, []uint64{1}, []byte{1}),
				createAttestationWrapper(0, 3, []uint64{1}, []byte{1}),
				createAttestationWrapper(0, 4, []uint64{1}, []byte{1}),
				createAttestationWrapper(0, 5, []uint64{1}, []byte{1}),
				createAttestationWrapper(0, 6, []uint64{1}, []byte{1}),
			},
			afterPruning: []*slashertypes.IndexedAttestationWrapper{
				createAttestationWrapper(0, 4, []uint64{1}, []byte{1}),
				createAttestationWrapper(0, 5, []uint64{1}, []byte{1}),
				createAttestationWrapper(0, 6, []uint64{1}, []byte{1}),
			},
			historyLength: 2,
			epoch:         4,
		},
		{
			name: "should delete all attestations under epoch 4",
			attestationsInDB: []*slashertypes.IndexedAttestationWrapper{
				createAttestationWrapper(0, 1, []uint64{1}, []byte{1}),
				createAttestationWrapper(0, 2, []uint64{1}, []byte{1}),
				createAttestationWrapper(0, 3, []uint64{1}, []byte{1}),
			},
			afterPruning: []*slashertypes.IndexedAttestationWrapper{},
			epoch:        4,
		},
		{
			name: "no attestations to delete under epoch 1",
			attestationsInDB: []*slashertypes.IndexedAttestationWrapper{
				createAttestationWrapper(0, 1, []uint64{1}, []byte{1}),
			},
			afterPruning: []*slashertypes.IndexedAttestationWrapper{
				createAttestationWrapper(0, 1, []uint64{1}, []byte{1}),
			},
			epoch: 5,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			beaconDB := setupDB(t)
			require.NoError(t, beaconDB.SaveAttestationRecordsForValidators(ctx, tt.attestationsInDB, defaultHistoryLength))
			pruningEpochIncrements := types.Epoch(100)
			if err := beaconDB.PruneProposals(ctx, tt.epoch, pruningEpochIncrements, tt.historyLength); (err != nil) != tt.wantErr {
				t.Errorf("PruneProposals() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Second time checking same proposals but all with different signing root should
			// return all double proposals.
			slashable := make([]*slashertypes.IndexedAttestationWrapper, len(tt.afterPruning))
			for i := 0; i < len(tt.afterPruning); i++ {
				slashable[i] = createAttestationWrapper(
					tt.afterPruning[i].IndexedAttestation.Data.Source.Epoch,
					tt.afterPruning[i].IndexedAttestation.Data.Target.Epoch,
					tt.afterPruning[i].IndexedAttestation.AttestingIndices,
					[]byte{2},
				)
			}

			doubleProposals, err := beaconDB.CheckAttesterDoubleVotes(ctx, slashable, defaultHistoryLength)
			require.NoError(t, err)
			require.Equal(t, len(tt.afterPruning), len(doubleProposals))
			for i, existing := range doubleProposals {
				require.DeepEqual(t, existing.PrevAttestationWrapper.SigningRoot, tt.afterPruning[i].SigningRoot)
			}
		})
	}
}
