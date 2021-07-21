package slasherkv

import (
	"context"
	"fmt"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	slashertypes "github.com/prysmaticlabs/prysm/beacon-chain/slasher/types"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	logTest "github.com/sirupsen/logrus/hooks/test"
	bolt "go.etcd.io/bbolt"
)

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
			wantedLog := fmt.Sprintf("Pruned %d/%d epochs", i, pruningLimitEpoch-1)
			require.LogsContain(t, hook, wantedLog)
		}

		// Everything before epoch 10 should be deleted.
		for i := types.Epoch(0); i < pruningLimitEpoch; i++ {
			err = beaconDB.db.View(func(tx *bolt.Tx) error {
				bkt := tx.Bucket(proposalRecordsBucket)
				startSlot, err := helpers.StartSlot(i)
				require.NoError(t, err)
				endSlot, err := helpers.StartSlot(i + 1)
				require.NoError(t, err)
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
			encodedTargetEpoch := encodeTargetEpoch(lowestStoredEpoch)
			key := append(encodedTargetEpoch, encIdx...)
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
	// that all the attestations written are deleted from the database, while those above the
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

		// We create attestations from genesis to the current epoch, with 2 attestations
		// at each slot to ensure the entire pruning logic works correctly.
		slotsPerEpoch := params.BeaconConfig().SlotsPerEpoch
		attestations := make([]*slashertypes.IndexedAttestationWrapper, 0, uint64(currentEpoch)*uint64(slotsPerEpoch)*2)
		for i := types.Epoch(0); i < currentEpoch; i++ {
			startSlot, err := helpers.StartSlot(i)
			require.NoError(t, err)
			endSlot, err := helpers.StartSlot(i + 1)
			require.NoError(t, err)
			for j := startSlot; j < endSlot; j++ {
				attester1 := uint64(j + 10)
				attester2 := uint64(j + 11)
				target := i
				var source types.Epoch
				if i > 0 {
					source = target - 1
				}
				att1 := createAttestationWrapper(source, target, []uint64{attester1}, []byte{0})
				att2 := createAttestationWrapper(source, target, []uint64{attester2}, []byte{1})
				attestations = append(attestations, att1, att2)
			}
		}

		require.NoError(t, beaconDB.SaveAttestationRecordsForValidators(ctx, attestations))

		// We expect pruning completes without an issue and properly logs progress.
		err := beaconDB.PruneAttestations(ctx, currentEpoch, epochPruningIncrements, historyLength)
		require.NoError(t, err)

		for i := types.Epoch(0); i < pruningLimitEpoch; i++ {
			wantedLog := fmt.Sprintf("Pruned %d/%d epochs", i, pruningLimitEpoch-1)
			require.LogsContain(t, hook, wantedLog)
		}

		// Everything before epoch 10 should be deleted.
		for i := types.Epoch(0); i < pruningLimitEpoch; i++ {
			err = beaconDB.db.View(func(tx *bolt.Tx) error {
				bkt := tx.Bucket(attestationDataRootsBucket)
				startSlot, err := helpers.StartSlot(i)
				require.NoError(t, err)
				endSlot, err := helpers.StartSlot(i + 1)
				require.NoError(t, err)
				for j := startSlot; j < endSlot; j++ {
					attester1 := types.ValidatorIndex(j + 10)
					attester2 := types.ValidatorIndex(j + 11)
					key1 := append(encodeTargetEpoch(i), encodeValidatorIndex(attester1)...)
					key2 := append(encodeTargetEpoch(i), encodeValidatorIndex(attester2)...)
					if bkt.Get(key1) != nil {
						return fmt.Errorf("still exists for epoch %d, validator %d", i, attester1)
					}
					if bkt.Get(key2) != nil {
						return fmt.Errorf("still exists for slot %d, validator %d", i, attester2)
					}
				}
				return nil
			})
			require.NoError(t, err)
		}
	})
}
