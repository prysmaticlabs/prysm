package main

import (
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/time/slots"
)

func applyLightClientUpdate(snapshot *LightClientSnapshot, update *LightClientUpdate) {
	snapshotPeriod := slots.ToEpoch(snapshot.Header.Slot) / params.BeaconConfig().EpochsPerSyncCommitteePeriod
	updatePeriod := slots.ToEpoch(update.Header.Slot) / params.BeaconConfig().EpochsPerSyncCommitteePeriod
	if updatePeriod == snapshotPeriod+1 {
		snapshot.CurrentSyncCommittee = snapshot.NextSyncCommittee
	} else {
		snapshot.Header = update.Header
	}
}

func processLightClientUpdate(
	store *Store,
	update *LightClientUpdate,
	currentSlot types.Slot,
	genesisValidatorsRoot [32]byte,
) error {
	if err := validateLightClientUpdate(store.Snapshot, update, genesisValidatorsRoot); err != nil {
		return err
	}
	store.ValidUpdates = append(store.ValidUpdates, update)
	updateTimeout := uint64(params.BeaconConfig().SlotsPerEpoch) * uint64(params.BeaconConfig().EpochsPerSyncCommitteePeriod)
	sumParticipantBits := update.SyncCommitteeBits.Count()
	hasQuorum := sumParticipantBits*3 >= uint64(len(update.SyncCommitteeBits))*2
	if hasQuorum && !isEmptyBlockHeader(update.FinalityHeader) {
		// Apply update if (1) 2/3 quorum is reached and (2) we have a finality proof.
		// Note that (2) means that the current light client design needs finality.
		// It may be changed to re-organizable light client design. See the on-going issue consensus-specs#2182.
		applyLightClientUpdate(store.Snapshot, update)
		store.ValidUpdates = make([]*LightClientUpdate, 0)
	} else if currentSlot > store.Snapshot.Header.Slot.Add(updateTimeout) {
		// Forced best update when the update timeout has elapsed
		// Use the update that has the highest sum of sync committee bits.
		updateWithHighestSumBits := store.ValidUpdates[0]
		highestSumBitsUpdate := updateWithHighestSumBits.SyncCommitteeBits.Count()
		for _, validUpdate := range store.ValidUpdates {
			sumUpdateBits := validUpdate.SyncCommitteeBits.Count()
			if sumUpdateBits > highestSumBitsUpdate {
				highestSumBitsUpdate = sumUpdateBits
				updateWithHighestSumBits = validUpdate
			}
		}
		applyLightClientUpdate(store.Snapshot, updateWithHighestSumBits)
		store.ValidUpdates = make([]*LightClientUpdate, 0)
	}
	return nil
}
