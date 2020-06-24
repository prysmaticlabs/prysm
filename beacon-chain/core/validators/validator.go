// Package validators contains libraries to shuffle validators
// and retrieve active validator indices from a given slot
// or an attestation. It also provides helper functions to locate
// validator based on pubic key.
package validators

import (
	"fmt"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

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
func InitiateValidatorExit(state *stateTrie.BeaconState, idx uint64) (*stateTrie.BeaconState, error) {
	vals := state.Validators()
	if idx >= uint64(len(vals)) {
		return nil, fmt.Errorf("validator idx %d is higher then validator count %d", idx, len(vals))
	}
	validator := vals[idx]
	if validator.ExitEpoch != params.BeaconConfig().FarFutureEpoch {
		return state, nil
	}
	exitEpochs := []uint64{}
	for _, val := range vals {
		if val.ExitEpoch != params.BeaconConfig().FarFutureEpoch {
			exitEpochs = append(exitEpochs, val.ExitEpoch)
		}
	}
	exitEpochs = append(exitEpochs, helpers.ActivationExitEpoch(helpers.CurrentEpoch(state)))

	// Obtain the exit queue epoch as the maximum number in the exit epochs array.
	exitQueueEpoch := uint64(0)
	for _, i := range exitEpochs {
		if exitQueueEpoch < i {
			exitQueueEpoch = i
		}
	}

	// We use the exit queue churn to determine if we have passed a churn limit.
	exitQueueChurn := 0
	for _, val := range vals {
		if val.ExitEpoch == exitQueueEpoch {
			exitQueueChurn++
		}
	}
	activeValidatorCount, err := helpers.ActiveValidatorCount(state, helpers.CurrentEpoch(state))
	if err != nil {
		return nil, errors.Wrap(err, "could not get active validator count")
	}
	churn, err := helpers.ValidatorChurnLimit(activeValidatorCount)
	if err != nil {
		return nil, errors.Wrap(err, "could not get churn limit")
	}

	if uint64(exitQueueChurn) >= churn {
		exitQueueEpoch++
	}
	validator.ExitEpoch = exitQueueEpoch
	validator.WithdrawableEpoch = exitQueueEpoch + params.BeaconConfig().MinValidatorWithdrawabilityDelay
	if err := state.UpdateValidatorAtIndex(idx, validator); err != nil {
		return nil, err
	}
	return state, nil
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
//        whistleblower_index = proposer_index
//    whistleblower_reward = Gwei(validator.effective_balance // WHISTLEBLOWER_REWARD_QUOTIENT)
//    proposer_reward = Gwei(whistleblower_reward // PROPOSER_REWARD_QUOTIENT)
//    increase_balance(state, proposer_index, proposer_reward)
//    increase_balance(state, whistleblower_index, Gwei(whistleblower_reward - proposer_reward))
func SlashValidator(state *stateTrie.BeaconState, slashedIdx uint64) (*stateTrie.BeaconState, error) {
	state, err := InitiateValidatorExit(state, slashedIdx)
	if err != nil {
		return nil, errors.Wrapf(err, "could not initiate validator %d exit", slashedIdx)
	}
	currentEpoch := helpers.SlotToEpoch(state.Slot())
	validator, err := state.ValidatorAtIndex(slashedIdx)
	if err != nil {
		return nil, err
	}
	validator.Slashed = true
	maxWithdrawableEpoch := mathutil.Max(validator.WithdrawableEpoch, currentEpoch+params.BeaconConfig().EpochsPerSlashingsVector)
	validator.WithdrawableEpoch = maxWithdrawableEpoch

	if err := state.UpdateValidatorAtIndex(slashedIdx, validator); err != nil {
		return nil, err
	}

	// The slashing amount is represented by epochs per slashing vector. The validator's effective balance is then applied to that amount.
	slashings := state.Slashings()
	currentSlashing := slashings[currentEpoch%params.BeaconConfig().EpochsPerSlashingsVector]
	if err := state.UpdateSlashingsAtIndex(
		currentEpoch%params.BeaconConfig().EpochsPerSlashingsVector,
		currentSlashing+validator.EffectiveBalance,
	); err != nil {
		return nil, err
	}
	if err := helpers.DecreaseBalance(state, slashedIdx, validator.EffectiveBalance/params.BeaconConfig().MinSlashingPenaltyQuotient); err != nil {
		return nil, err
	}

	proposerIdx, err := helpers.BeaconProposerIndex(state)
	if err != nil {
		return nil, errors.Wrap(err, "could not get proposer idx")
	}
	// In phase 0, the proposer is the whistleblower.
	whistleBlowerIdx := proposerIdx
	whistleblowerReward := validator.EffectiveBalance / params.BeaconConfig().WhistleBlowerRewardQuotient
	proposerReward := whistleblowerReward / params.BeaconConfig().ProposerRewardQuotient
	err = helpers.IncreaseBalance(state, proposerIdx, proposerReward)
	if err != nil {
		return nil, err
	}
	err = helpers.IncreaseBalance(state, whistleBlowerIdx, whistleblowerReward-proposerReward)
	if err != nil {
		return nil, err
	}
	return state, nil
}

// ActivatedValidatorIndices determines the indices activated during the given epoch.
func ActivatedValidatorIndices(epoch uint64, validators []*ethpb.Validator) []uint64 {
	activations := make([]uint64, 0)
	for i := 0; i < len(validators); i++ {
		val := validators[i]
		if val.ActivationEpoch <= epoch && epoch < val.ExitEpoch {
			activations = append(activations, uint64(i))
		}
	}
	return activations
}

// SlashedValidatorIndices determines the indices slashed during the given epoch.
func SlashedValidatorIndices(epoch uint64, validators []*ethpb.Validator) []uint64 {
	slashed := make([]uint64, 0)
	for i := 0; i < len(validators); i++ {
		val := validators[i]
		maxWithdrawableEpoch := mathutil.Max(val.WithdrawableEpoch, epoch+params.BeaconConfig().EpochsPerSlashingsVector)
		if val.WithdrawableEpoch == maxWithdrawableEpoch && val.Slashed {
			slashed = append(slashed, uint64(i))
		}
	}
	return slashed
}

// ExitedValidatorIndices determines the indices exited during the current epoch.
func ExitedValidatorIndices(epoch uint64, validators []*ethpb.Validator, activeValidatorCount uint64) ([]uint64, error) {
	exited := make([]uint64, 0)
	exitEpochs := make([]uint64, 0)
	for i := 0; i < len(validators); i++ {
		val := validators[i]
		if val.ExitEpoch != params.BeaconConfig().FarFutureEpoch {
			exitEpochs = append(exitEpochs, val.ExitEpoch)
		}
	}
	exitQueueEpoch := uint64(0)
	for _, i := range exitEpochs {
		if exitQueueEpoch < i {
			exitQueueEpoch = i
		}
	}

	// We use the exit queue churn to determine if we have passed a churn limit.
	exitQueueChurn := 0
	for _, val := range validators {
		if val.ExitEpoch == exitQueueEpoch {
			exitQueueChurn++
		}
	}
	churn, err := helpers.ValidatorChurnLimit(activeValidatorCount)
	if err != nil {
		return nil, errors.Wrap(err, "could not get churn limit")
	}
	if churn < uint64(exitQueueChurn) {
		exitQueueEpoch++
	}
	withdrawableEpoch := exitQueueEpoch + params.BeaconConfig().MinValidatorWithdrawabilityDelay
	for i, val := range validators {
		if val.ExitEpoch == epoch && val.WithdrawableEpoch == withdrawableEpoch &&
			val.EffectiveBalance > params.BeaconConfig().EjectionBalance {
			exited = append(exited, uint64(i))
		}
	}
	return exited, nil
}

// EjectedValidatorIndices determines the indices ejected during the given epoch.
func EjectedValidatorIndices(epoch uint64, validators []*ethpb.Validator, activeValidatorCount uint64) ([]uint64, error) {
	ejected := make([]uint64, 0)
	exitEpochs := make([]uint64, 0)
	for i := 0; i < len(validators); i++ {
		val := validators[i]
		if val.ExitEpoch != params.BeaconConfig().FarFutureEpoch {
			exitEpochs = append(exitEpochs, val.ExitEpoch)
		}
	}
	exitQueueEpoch := uint64(0)
	for _, i := range exitEpochs {
		if exitQueueEpoch < i {
			exitQueueEpoch = i
		}
	}

	// We use the exit queue churn to determine if we have passed a churn limit.
	exitQueueChurn := 0
	for _, val := range validators {
		if val.ExitEpoch == exitQueueEpoch {
			exitQueueChurn++
		}
	}
	churn, err := helpers.ValidatorChurnLimit(activeValidatorCount)
	if err != nil {
		return nil, errors.Wrap(err, "could not get churn limit")
	}
	if churn < uint64(exitQueueChurn) {
		exitQueueEpoch++
	}
	withdrawableEpoch := exitQueueEpoch + params.BeaconConfig().MinValidatorWithdrawabilityDelay
	for i, val := range validators {
		if val.ExitEpoch == epoch && val.WithdrawableEpoch == withdrawableEpoch &&
			val.EffectiveBalance <= params.BeaconConfig().EjectionBalance {
			ejected = append(ejected, uint64(i))
		}
	}
	return ejected, nil
}
