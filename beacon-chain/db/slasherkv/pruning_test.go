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

func TestStore_PruneProposals(t *testing.T) {
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
		for i := types.Epoch(0); i < currentEpoch; i++ {
			startSlot, err := helpers.StartSlot(i)
			require.NoError(t, err)
			endSlot, err := helpers.StartSlot(i + 1)
			require.NoError(t, err)
			for j := startSlot; j < endSlot; j++ {
				prop1 := createProposalWrapper(t, j, 0 /* proposer index */, []byte{0})
				prop2 := createProposalWrapper(t, j, 2 /* proposer index */, []byte{1})
				proposals = append(proposals, prop1, prop2)
			}
		}

		require.NoError(t, beaconDB.SaveBlockProposals(ctx, proposals))

		// We expect pruning completes without an issue and properly logs progress.
		err := beaconDB.PruneProposals(ctx, currentEpoch, epochPruningIncrements, historyLength)
		require.NoError(t, err)

		for i := types.Epoch(0); i < pruningLimitEpoch; i++ {
			wantedLog := fmt.Sprintf("Pruned %d/%d epochs worth of proposals", i, pruningLimitEpoch-1)
			require.LogsContain(t, hook, wantedLog)
		}

		// Everything before epoch 10 should be deleted.
		for i := types.Epoch(0); i < pruningLimitEpoch; i++ {
			err = beaconDB.db.View(func(tx *bolt.Tx) error {
				bkt := tx.Bucket(proposalRecordsBucket)
				startSlot, err := helpers.StartSlot(i)
				require.NoError(t, err)
				endSlot, err := helpers.StartSlot(i + 1)
				for j := startSlot; j < endSlot; j++ {
					prop1 := createProposalWrapper(t, j, 0 /* proposer index */, []byte{0})
					prop1Key, err := keyForValidatorProposal(prop1)
					if err != nil {
						return err
					}
					prop2 := createProposalWrapper(t, j, 2 /* proposer index */, []byte{1})
					prop2Key, err := keyForValidatorProposal(prop2)
					if err != nil {
						return err
					}
					if bkt.Get(prop1Key) != nil {
						return fmt.Errorf("proposal still exists for epoch %d, validator 0", j)
					}
					if bkt.Get(prop2Key) != nil {
						return fmt.Errorf("proposal still exists for slot %d, validator 1", j)
					}
				}
				return nil
			})
			require.NoError(t, err)
		}
	})
}

func TestStore_PruneAttestations_OK(t *testing.T) {
	ctx := context.Background()

	// If the current slot is less than the history length we track in slasher, there
	// is nothing to prune yet, so we expect exiting early.
	t.Run("current_epoch_less_than_history_length", func(t *testing.T) {
		hook := logTest.NewGlobal()
		beaconDB := setupDB(t)
		epochPruningIncrements := types.Epoch(100)
		currentEpoch := types.Epoch(1)
		historyLength := types.Epoch(2)
		err := beaconDB.PruneAttestations(ctx, currentEpoch, epochPruningIncrements, historyLength)
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
		lowestStoredEpoch := pruningLimitEpoch

		err := beaconDB.db.Update(func(tx *bolt.Tx) error {
			bkt := tx.Bucket(attestationDataRootsBucket)
			encIdx := encodeValidatorIndex(types.ValidatorIndex(0))
			encodedTargetEpoch := encodeTargetEpoch(lowestStoredEpoch, historyLength)
			key := append(encodedTargetEpoch, encIdx...)
			fmt.Println("Putting", lowestStoredEpoch, key)
			return bkt.Put(key, []byte("hi"))
		})
		require.NoError(t, err)

		err = beaconDB.PruneAttestations(ctx, currentEpoch, epochPruningIncrements, historyLength)
		require.NoError(t, err)
		expectedLog := fmt.Sprintf(
			"Lowest epoch %d is >= pruning epoch %d, nothing to prune", lowestStoredEpoch, lowestStoredEpoch,
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
		for i := types.Epoch(0); i < currentEpoch; i++ {
			startSlot, err := helpers.StartSlot(i)
			require.NoError(t, err)
			endSlot, err := helpers.StartSlot(i + 1)
			require.NoError(t, err)
			for j := startSlot; j < endSlot; j++ {
				prop1 := createProposalWrapper(t, j, 0 /* proposer index */, []byte{0})
				prop2 := createProposalWrapper(t, j, 2 /* proposer index */, []byte{1})
				proposals = append(proposals, prop1, prop2)
			}
		}

		require.NoError(t, beaconDB.SaveBlockProposals(ctx, proposals))

		// We expect pruning completes without an issue and properly logs progress.
		err := beaconDB.PruneProposals(ctx, currentEpoch, epochPruningIncrements, historyLength)
		require.NoError(t, err)

		for i := types.Epoch(0); i < pruningLimitEpoch; i++ {
			wantedLog := fmt.Sprintf("Pruned %d/%d epochs worth of proposals", i, pruningLimitEpoch-1)
			require.LogsContain(t, hook, wantedLog)
		}

		// Everything before epoch 10 should be deleted.
		for i := types.Epoch(0); i < pruningLimitEpoch; i++ {
			err = beaconDB.db.View(func(tx *bolt.Tx) error {
				bkt := tx.Bucket(proposalRecordsBucket)
				startSlot, err := helpers.StartSlot(i)
				require.NoError(t, err)
				endSlot, err := helpers.StartSlot(i + 1)
				for j := startSlot; j < endSlot; j++ {
					prop1 := createProposalWrapper(t, j, 0 /* proposer index */, []byte{0})
					prop1Key, err := keyForValidatorProposal(prop1)
					if err != nil {
						return err
					}
					prop2 := createProposalWrapper(t, j, 2 /* proposer index */, []byte{1})
					prop2Key, err := keyForValidatorProposal(prop2)
					if err != nil {
						return err
					}
					if bkt.Get(prop1Key) != nil {
						return fmt.Errorf("proposal still exists for epoch %d, validator 0", j)
					}
					if bkt.Get(prop2Key) != nil {
						return fmt.Errorf("proposal still exists for slot %d, validator 1", j)
					}
				}
				return nil
			})
			require.NoError(t, err)
		}
	})
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
