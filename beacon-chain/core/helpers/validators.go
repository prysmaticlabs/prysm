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
//  def is_active_validator(validator: Validator, epoch: Epoch) -> bool:
//    """
//    Check if ``validator`` is active.
//    """
//    return validator.activation_epoch <= epoch < validator.exit_epoch
func IsActiveValidator(validator *pb.Validator, epoch uint64) bool {
	return validator.ActivationEpoch <= epoch &&
		epoch < validator.ExitEpoch
}

// IsSlashableValidator returns the boolean value on whether the validator
// is slashable or not.
//
// Spec pseudocode definition:
//  def is_slashable_validator(validator: Validator, epoch: Epoch) -> bool:
//    """
//    Check if ``validator`` is slashable.
//    """
//    return (
//        validator.activation_epoch <= epoch < validator.withdrawable_epoch and
//        validator.slashed is False
// 		)
func IsSlashableValidator(validator *pb.Validator, epoch uint64) bool {
	active := validator.ActivationEpoch <= epoch
	beforeWithdrawable := epoch < validator.WithdrawableEpoch
	return beforeWithdrawable && active && !validator.Slashed
}

// ActiveValidatorIndices filters out active validators based on validator status
// and returns their indices in a list.
//
// Spec pseudocode definition:
//  def get_active_validator_indices(state: BeaconState, epoch: Epoch) -> List[ValidatorIndex]:
//    """
//    Get active validator indices at ``epoch``.
//    """
//    return [i for i, v in enumerate(state.validator_registry) if is_active_validator(v, epoch)]
func ActiveValidatorIndices(state *pb.BeaconState, epoch uint64) []uint64 {
	indices := make([]uint64, 0, len(state.ValidatorRegistry))
	for i, v := range state.ValidatorRegistry {
		if IsActiveValidator(v, epoch) {
			indices = append(indices, uint64(i))
		}
	}
	return indices
}

// DelayedActivationExitEpoch takes in epoch number and returns when
// the validator is eligible for activation and exit.
//
// Spec pseudocode definition:
//  def get_delayed_activation_exit_epoch(epoch: Epoch) -> Epoch:
//    """
//    Return the epoch at which an activation or exit triggered in ``epoch`` takes effect.
//    """
//    return epoch + 1 + ACTIVATION_EXIT_DELAY
func DelayedActivationExitEpoch(epoch uint64) uint64 {
	return epoch + 1 + params.BeaconConfig().ActivationExitDelay
}

// ChurnLimit returns the number of validators that are allowed to
// enter and exit validator pool for an epoch.
//
// Spec pseudocode definition:
// def get_churn_limit(state: BeaconState) -> int:
//    return max(
//        MIN_PER_EPOCH_CHURN_LIMIT,
//        len(get_active_validator_indices(state, get_current_epoch(state))) // CHURN_LIMIT_QUOTIENT
//    )
func ChurnLimit(state *pb.BeaconState) uint64 {
	validatorCount := uint64(len(ActiveValidatorIndices(state.ValidatorRegistry, CurrentEpoch(state))))
	if validatorCount/params.BeaconConfig().ChurnLimitQuotient > params.BeaconConfig().MinPerEpochChurnLimit {
		return validatorCount / params.BeaconConfig().ChurnLimitQuotient
	}
	return params.BeaconConfig().MinPerEpochChurnLimit
}

// BeaconProposerIndex returns the index of the proposer of the block at a
// given slot.
// TODO(2307): Update BeaconProposerIndex to v0.6
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
	committeeArray, err := CrosslinkCommitteesAtSlot(state, slot)
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
