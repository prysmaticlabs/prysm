// Package validators contains libraries to shuffle validators
// and retrieve active validator indices from a given slot
// or an attestation. It also provides helper functions to locate
// validator based on pubic key.
package validators

import (
	"fmt"
	"sync"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

type validatorStore struct {
	sync.RWMutex
	// activatedValidators is a mapping that tracks validator activation epoch to validators index.
	activatedValidators map[uint64][]uint64
	// exitedValidators is a mapping that tracks validator exit epoch to validators index.
	exitedValidators map[uint64][]uint64
}

//VStore validator map for quick
var VStore = validatorStore{
	activatedValidators: make(map[uint64][]uint64),
	exitedValidators:    make(map[uint64][]uint64),
}

// InitiateValidatorExit takes in validator index and updates
// validator with correct voluntary exit parameters.
//
// Spec pseudocode definition:
//  def initiate_validator_exit(state: BeaconState, index: ValidatorIndex) -> None:
//    """
//    Initiate the exit of the validator with index ``index``.
//    """
//    # Return if validator already initiated exit
//    validator = state.validators[index]
//    if validator.exit_epoch != FAR_FUTURE_EPOCH:
//        return
//
//    # Compute exit queue epoch
//    exit_epochs = [v.exit_epoch for v in state.validators if v.exit_epoch != FAR_FUTURE_EPOCH]
//    exit_queue_epoch = max(exit_epochs + [compute_activation_exit_epoch(get_current_epoch(state))])
//    exit_queue_churn = len([v for v in state.validators if v.exit_epoch == exit_queue_epoch])
//    if exit_queue_churn >= get_validator_churn_limit(state):
//        exit_queue_epoch += Epoch(1)
//
//    # Set validator exit epoch and withdrawable epoch
//    validator.exit_epoch = exit_queue_epoch
//    validator.withdrawable_epoch = Epoch(validator.exit_epoch + MIN_VALIDATOR_WITHDRAWABILITY_DELAY)
func InitiateValidatorExit(state *pb.BeaconState, idx uint64) (*pb.BeaconState, error) {
	validator := state.Validators[idx]
	if validator.ExitEpoch != params.BeaconConfig().FarFutureEpoch {
		return state, nil
	}
	exitEpochs := []uint64{}
	for _, val := range state.Validators {
		if val.ExitEpoch != params.BeaconConfig().FarFutureEpoch {
			exitEpochs = append(exitEpochs, val.ExitEpoch)
		}
	}
	exitEpochs = append(exitEpochs, helpers.DelayedActivationExitEpoch(helpers.CurrentEpoch(state)))

	// Obtain the exit queue epoch as the maximum number in the exit epochs array.
	exitQueueEpoch := uint64(0)
	for _, i := range exitEpochs {
		if exitQueueEpoch < i {
			exitQueueEpoch = i
		}
	}

	// We use the exit queue churn to determine if we have passed a churn limit.
	exitQueueChurn := 0
	for _, val := range state.Validators {
		if val.ExitEpoch == exitQueueEpoch {
			exitQueueChurn++
		}
	}
	churn, err := helpers.ValidatorChurnLimit(state)
	if err != nil {
		return nil, fmt.Errorf("could not get churn limit: %v", err)
	}

	if uint64(exitQueueChurn) >= churn {
		exitQueueEpoch++
	}
	state.Validators[idx].ExitEpoch = exitQueueEpoch
	state.Validators[idx].WithdrawableEpoch = exitQueueEpoch + params.BeaconConfig().MinValidatorWithdrawabilityDelay
	return state, nil
}

// ExitValidator takes in validator index and does house
// keeping work to exit validator with entry exit delay.
//
// Spec pseudocode definition:
//  def exit_validator(state: BeaconState, index: ValidatorIndex) -> None:
//    """
//    Exit the validator of the given ``index``.
//    Note that this function mutates ``state``.
//    """
//    validator = state.validator_registry[index]
//
//    # The following updates only occur if not previous exited
//    if validator.exit_epoch <= get_entry_exit_effect_epoch(get_current_epoch(state)):
//        return
//
//    validator.exit_epoch = get_entry_exit_effect_epoch(get_current_epoch(state))
func ExitValidator(state *pb.BeaconState, idx uint64) *pb.BeaconState {
	validator := state.Validators[idx]

	if validator.ExitEpoch != params.BeaconConfig().FarFutureEpoch {
		return state
	}
	validator.ExitEpoch = helpers.DelayedActivationExitEpoch(helpers.CurrentEpoch(state))
	return state
}

// SlashValidator slashes the malicious validator's balance and awards
// the whistleblower's balance.
//
// Spec pseudocode definition:
//  def slash_validator(state: BeaconState,
//                    slashed_index: ValidatorIndex,
//                    whistleblower_index: ValidatorIndex=None) -> None:
//    """
//    Slash the validator with index ``slashed_index``.
//    """
//    epoch = get_current_epoch(state)
//    initiate_validator_exit(state, slashed_index)
//    validator = state.validators[slashed_index]
//    validator.slashed = True
//    validator.withdrawable_epoch = max(validator.withdrawable_epoch, Epoch(epoch + EPOCHS_PER_SLASHINGS_VECTOR))
//    state.slashings[epoch % EPOCHS_PER_SLASHINGS_VECTOR] += validator.effective_balance
//    decrease_balance(state, slashed_index, validator.effective_balance // MIN_SLASHING_PENALTY_QUOTIENT)
//
//    # Apply proposer and whistleblower rewards
//    proposer_index = get_beacon_proposer_index(state)
//    if whistleblower_index is None:
//    whistleblower_reward = Gwei(validator.effective_balance // WHISTLEBLOWER_REWARD_QUOTIENT)
//    proposer_reward = Gwei(whistleblower_reward // PROPOSER_REWARD_QUOTIENT)
//    increase_balance(state, proposer_index, proposer_reward)
//    increase_balance(state, whistleblower_index, whistleblower_reward - proposer_reward)
func SlashValidator(state *pb.BeaconState, slashedIdx uint64, whistleBlowerIdx uint64) (*pb.BeaconState, error) {
	state, err := InitiateValidatorExit(state, slashedIdx)
	if err != nil {
		return nil, fmt.Errorf("could not initiate validator %d exit: %v", slashedIdx, err)
	}
	currentEpoch := helpers.CurrentEpoch(state)
	validator := state.Validators[slashedIdx]
	validator.Slashed = true
	maxWithdrawableEpoch := mathutil.Max(validator.WithdrawableEpoch, currentEpoch+params.BeaconConfig().EpochsPerSlashingsVector)
	validator.WithdrawableEpoch = maxWithdrawableEpoch
	state.Slashings[currentEpoch%params.BeaconConfig().EpochsPerSlashingsVector] += validator.EffectiveBalance
	helpers.DecreaseBalance(state, slashedIdx, validator.EffectiveBalance/params.BeaconConfig().MinSlashingPenaltyQuotient)

	proposerIdx, err := helpers.BeaconProposerIndex(state)
	if err != nil {
		return nil, fmt.Errorf("could not get proposer idx: %v", err)
	}

	if whistleBlowerIdx == 0 {
		whistleBlowerIdx = proposerIdx
	}
	whistleblowerReward := validator.EffectiveBalance / params.BeaconConfig().WhistleBlowerRewardQuotient
	proposerReward := whistleblowerReward / params.BeaconConfig().ProposerRewardQuotient
	state = helpers.IncreaseBalance(state, proposerIdx, proposerReward)
	state = helpers.IncreaseBalance(state, whistleBlowerIdx, whistleblowerReward-proposerReward)
	return state, nil
}

// InitializeValidatorStore sets the current active validators from the current
// state.
func InitializeValidatorStore(bState *pb.BeaconState) error {
	VStore.Lock()
	defer VStore.Unlock()

	currentEpoch := helpers.CurrentEpoch(bState)
	activeValidatorIndices, err := helpers.ActiveValidatorIndices(bState, currentEpoch)
	if err != nil {
		return err
	}
	VStore.activatedValidators[currentEpoch] = activeValidatorIndices
	return nil
}

// InsertActivatedVal locks the validator store, inserts the activated validator
// indices, then unlocks the store again. This method may be used by
// external services in testing to populate the validator store.
func InsertActivatedVal(epoch uint64, validators []uint64) {
	VStore.Lock()
	defer VStore.Unlock()
	VStore.activatedValidators[epoch] = validators
}

// InsertActivatedIndices locks the validator store, inserts the activated validator
// indices corresponding to their activation epochs.
func InsertActivatedIndices(epoch uint64, indices []uint64) {
	VStore.Lock()
	defer VStore.Unlock()
	VStore.activatedValidators[epoch] = append(VStore.activatedValidators[epoch], indices...)
}

// InsertExitedVal locks the validator store, inserts the exited validator
// indices, then unlocks the store again. This method may be used by
// external services in testing to remove the validator store.
func InsertExitedVal(epoch uint64, validators []uint64) {
	VStore.Lock()
	defer VStore.Unlock()
	VStore.exitedValidators[epoch] = validators
}

// ActivatedValFromEpoch locks the validator store, retrieves the activated validator
// indices of a given epoch, then unlocks the store again.
func ActivatedValFromEpoch(epoch uint64) []uint64 {
	VStore.RLock()
	defer VStore.RUnlock()
	if _, exists := VStore.activatedValidators[epoch]; !exists {
		return nil
	}
	return VStore.activatedValidators[epoch]
}

// ExitedValFromEpoch locks the validator store, retrieves the exited validator
// indices of a given epoch, then unlocks the store again.
func ExitedValFromEpoch(epoch uint64) []uint64 {
	VStore.RLock()
	defer VStore.RUnlock()
	if _, exists := VStore.exitedValidators[epoch]; !exists {
		return nil
	}
	return VStore.exitedValidators[epoch]
}

// DeleteActivatedVal locks the validator store, delete the activated validator
// indices of a given epoch, then unlocks the store again.
func DeleteActivatedVal(epoch uint64) {
	VStore.Lock()
	defer VStore.Unlock()
	delete(VStore.activatedValidators, epoch)
}

// DeleteExitedVal locks the validator store, delete the exited validator
// indices of a given epoch, then unlocks the store again.
func DeleteExitedVal(epoch uint64) {
	VStore.Lock()
	defer VStore.Unlock()
	delete(VStore.exitedValidators, epoch)
}
