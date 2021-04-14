package slasherkv

import (
	"context"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	slashertypes "github.com/prysmaticlabs/prysm/beacon-chain/slasher/types"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestStore_PruneProposals_Logs(t *testing.T) {
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			beaconDB := setupDB(t)
			require.NoError(t, beaconDB.SaveBlockProposals(ctx, tt.proposalsInDB))
			if err := beaconDB.PruneProposals(ctx, tt.epoch, tt.historyLength); (err != nil) != tt.wantErr {
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
			beaconDB := setupDB(t)
			require.NoError(t, beaconDB.SaveBlockProposals(ctx, tt.proposalsInDB))
			if err := beaconDB.PruneProposals(ctx, tt.epoch, tt.historyLength); (err != nil) != tt.wantErr {
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
			if err := beaconDB.PruneProposals(ctx, tt.epoch, tt.historyLength); (err != nil) != tt.wantErr {
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
