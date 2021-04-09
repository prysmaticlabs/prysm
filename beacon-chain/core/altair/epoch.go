package altair

import (
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	iface "github.com/prysmaticlabs/prysm/beacon-chain/state/interface"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// ProcessSyncClientCommitteeUpdates processes sync client committee updates for the beacon state.
//
// Spec code:
// def process_sync_committee_updates(state: BeaconState) -> None:
//    next_epoch = get_current_epoch(state) + Epoch(1)
//    if next_epoch % EPOCHS_PER_SYNC_COMMITTEE_PERIOD == 0:
//        state.current_sync_committee = state.next_sync_committee
//        state.next_sync_committee = get_sync_committee(state, next_epoch + EPOCHS_PER_SYNC_COMMITTEE_PERIOD)
func ProcessSyncCommitteeUpdates(beaconState iface.BeaconStateAltair) (iface.BeaconStateAltair, error) {
	nextEpoch := helpers.NextEpoch(beaconState)
	if nextEpoch%params.BeaconConfig().EpochsPerSyncCommitteePeriod == 0 {
		currentSyncCommittee, err := beaconState.NextSyncCommittee()
		if err != nil {
			return nil, err
		}
		if err := beaconState.SetCurrentSyncCommittee(currentSyncCommittee); err != nil {
			return nil, err
		}
		nextCommittee, err := SyncCommittee(beaconState, helpers.CurrentEpoch(beaconState)+params.BeaconConfig().EpochsPerSyncCommitteePeriod)
		if err != nil {
			return nil, err
		}
		if err := beaconState.SetNextSyncCommittee(nextCommittee); err != nil {
			return nil, err
		}
	}
	return beaconState, nil
}

// ProcessParticipationFlagUpdates processes participation flag updates by rotating current to previous.
//
// Spec code:
// def process_participation_flag_updates(state: BeaconState) -> None:
//    state.previous_epoch_participation = state.current_epoch_participation
//    state.current_epoch_participation = [ParticipationFlags(0b0000_0000) for _ in range(len(state.validators))]
func ProcessParticipationFlagUpdates(beaconState iface.BeaconStateAltair) (iface.BeaconStateAltair, error) {
	c, err := beaconState.CurrentEpochParticipation()
	if err != nil {
		return nil, err
	}
	if err := beaconState.SetPreviousParticipationBits(c); err != nil {
		return nil, err
	}
	if err := beaconState.SetCurrentParticipationBits(make([]byte, beaconState.NumValidators())); err != nil {
		return nil, err
	}
	return beaconState, nil
}
