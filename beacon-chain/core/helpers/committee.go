package helpers

import (
	"encoding/binary"
	"fmt"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bitutil"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
)

// committeeCountPerSlot returns the number of crosslink committees of one slot.
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
		return maxCommitteePerSlot
	}
	if currCommitteePerSlot < 1 {
		return minCommitteePerSlot
	}
	return currCommitteePerSlot * config.EpochLength
}

// CurrentEpochCommitteeCount returns the number of crosslink committees per epoch
// of the current epoch.
// Ex: Returns 8 means there's 8 committees assigned to current epoch.
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

// prevCommitteesCountPerSlot returns the number of committees per slot
// of the previous epoch.
// Ex: Returns 16 means there's 16 committees assigned to one slot in previous epoch.
//
// Spec pseudocode definition:
//   def get_previous_epoch_committee_count_per_slot(state: BeaconState) -> int:
//         previous_active_validators =
// 			get_active_validator_indices(validators, state.previous_epoch_calculation_slot)
//        return get_committees_per_slot(len(previous_active_validators))
func prevCommitteesCountPerSlot(state *pb.BeaconState) uint64 {
	prevActiveValidatorIndices := ActiveValidatorIndices(
		state.ValidatorRegistry, state.PreviousEpochCalculationSlot)
	return committeeCountPerSlot(uint64(len(prevActiveValidatorIndices)))
}
