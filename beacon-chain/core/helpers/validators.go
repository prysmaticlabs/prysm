package helpers

import (
	"fmt"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// IsActiveValidator returns the boolean value on whether the validator
// is active or not.
//
// Spec pseudocode definition:
//   def is_active_validator(validator: Validator, epoch: Epoch) -> bool:
//    """
//    Check if ``validator`` is active.
//    """
//    return validator.activation_epoch <= epoch < validator.exit_epoch
func IsActiveValidator(validator *pb.Validator, epoch uint64) bool {
	return validator.ActivationEpoch <= epoch &&
		epoch < validator.ExitEpoch
}

// ActiveValidatorIndices filters out active validators based on validator status
// and returns their indices in a list.
//
// Spec pseudocode definition:
//   def get_active_validator_indices(validators: List[Validator], epoch: Epoch) -> List[ValidatorIndex]:
//    """
//    Get indices of active validators from ``validators``.
//    """
//    return [i for i, v in enumerate(validators) if is_active_validator(v, epoch)]
func ActiveValidatorIndices(validators []*pb.Validator, epoch uint64) []uint64 {
	indices := make([]uint64, 0, len(validators))
	for i, v := range validators {
		if IsActiveValidator(v, epoch) {
			indices = append(indices, uint64(i))
		}

	}
	return indices
}

// EntryExitEffectEpoch takes in epoch number and returns when
// the validator is eligible for activation and exit.
//
// Spec pseudocode definition:
// def get_entry_exit_effect_epoch(epoch: Epoch) -> Epoch:
//    """
//    An entry or exit triggered in the ``epoch`` given by the input takes effect at
//    the epoch given by the output.
//    """
//    return epoch + 1 + ACTIVATION_EXIT_DELAY
func EntryExitEffectEpoch(epoch uint64) uint64 {
	return epoch + 1 + params.BeaconConfig().ActivationExitDelay
}

// BeaconProposerIndex returns the index of the proposer of the block at a
// given slot.
//
// Spec pseudocode definition:
//  def get_beacon_proposer_index(state: BeaconState,slot: int) -> int:
//    """
//    Returns the beacon proposer index for the ``slot``.
//    """
//    first_committee, _ = get_crosslink_committees_at_slot(state, slot)[0]
//    return first_committee[slot % len(first_committee)]
func BeaconProposerIndex(state *pb.BeaconState, slot uint64) (uint64, error) {
	// RegistryChange is false because BeaconProposerIndex is only written
	// to be able to get proposers from current and previous epoch following
	// ETH2.0 beacon chain spec.
	committeeArray, err := CrosslinkCommitteesAtSlot(state, slot, false /* registryChange */)
	if err != nil {
		return 0, err
	}
	firstCommittee := committeeArray[0].Committee

	if len(firstCommittee) == 0 {
		return 0, fmt.Errorf("empty first committee at slot %d",
			slot-params.BeaconConfig().GenesisSlot)
	}

	return firstCommittee[slot%uint64(len(firstCommittee))], nil
}
