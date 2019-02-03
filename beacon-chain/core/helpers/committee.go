package helpers

import (
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// EpochCommitteeCount returns the number of crosslink committees of an epoch.
//
// Spec pseudocode definition:
//   def get_epoch_committee_count(active_validator_count: int) -> int:
//    """
//    Return the number of committees in one epoch.
//    """
//    return max(
//        1,
//        min(
//            SHARD_COUNT // EPOCH_LENGTH,
//            active_validator_count // EPOCH_LENGTH // TARGET_COMMITTEE_SIZE,
//        )
//    ) * EPOCH_LENGTH
func EpochCommitteeCount(activeValidatorCount uint64) uint64 {
	var minCommitteePerSlot = uint64(1)
	var maxCommitteePerSlot = config.ShardCount / config.EpochLength
	var currCommitteePerSlot = activeValidatorCount / config.EpochLength / config.TargetCommitteeSize
	if currCommitteePerSlot > maxCommitteePerSlot {
		return maxCommitteePerSlot * config.EpochLength
	}
	if currCommitteePerSlot < 1 {
		return minCommitteePerSlot * config.EpochLength
	}
	return currCommitteePerSlot * config.EpochLength
}

// CurrentEpochCommitteeCount returns the number of crosslink committees per epoch
// of the current epoch.
// Ex: Returns 100 means there's 8 committees assigned to current epoch.
//
// Spec pseudocode definition:
//   def get_current_epoch_committee_count(state: BeaconState) -> int:
//    """
//    Return the number of committees in the current epoch of the given ``state``.
//    """
//    current_active_validators = get_active_validator_indices(
//        state.validator_registry,
//        state.current_calculation_epoch,
//    )
//    return get_epoch_committee_count(len(current_active_validators)
func CurrentEpochCommitteeCount(state *pb.BeaconState) uint64 {
	currActiveValidatorIndices := ActiveValidatorIndices(
		state.ValidatorRegistry, state.CurrentEpochCalculationSlot)
	return EpochCommitteeCount(uint64(len(currActiveValidatorIndices)))
}

// PrevEpochCommitteeCount returns the number of committees per slot
// of the previous epoch.
//
// Spec pseudocode definition:
//   def get_previous_epoch_committee_count(state: BeaconState) -> int:
//    """
//    Return the number of committees in the previous epoch of the given ``state``.
//    """
//    previous_active_validators = get_active_validator_indices(
//        state.validator_registry,
//        state.previous_calculation_epoch,
//    )
//    return get_epoch_committee_count(len(previous_active_validators))
func PrevEpochCommitteeCount(state *pb.BeaconState) uint64 {
	prevActiveValidatorIndices := ActiveValidatorIndices(
		state.ValidatorRegistry, state.PreviousEpochCalculationSlot)
	return EpochCommitteeCount(uint64(len(prevActiveValidatorIndices)))
}

// NextEpochCommitteeCount returns the number of committees per slot
// of the next epoch.
//
// Spec pseudocode definition:
//   def get_next_epoch_committee_count(state: BeaconState) -> int:
//    """
//    Return the number of committees in the next epoch of the given ``state``.
//    """
//    next_active_validators = get_active_validator_indices(
//        state.validator_registry,
//        get_current_epoch(state) + 1,
//    )
//    return get_epoch_committee_count(len(next_active_validators))
func NextEpochCommitteeCount(state *pb.BeaconState) uint64 {
	prevActiveValidatorIndices := ActiveValidatorIndices(
		state.ValidatorRegistry, CurrentEpoch(state)+1)
	return EpochCommitteeCount(uint64(len(prevActiveValidatorIndices)))
}
