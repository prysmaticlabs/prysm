// Package validators contains libraries to shuffle validators
// and retrieve active validator indices from a given slot
// or an attestation. It also provides helper functions to locate
// validator based on pubic key.
package validators

import (
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
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
		return nil, errors.Wrap(err, "could not get churn limit")
	}

	if uint64(exitQueueChurn) >= churn {
		exitQueueEpoch++
	}
	state.Validators[idx].ExitEpoch = exitQueueEpoch
	state.Validators[idx].WithdrawableEpoch = exitQueueEpoch + params.BeaconConfig().MinValidatorWithdrawabilityDelay
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
//    whistleblower_reward = Gwei(validator.effective_balance // WHISTLEBLOWER_REWARD_QUOTIENT)
//    proposer_reward = Gwei(whistleblower_reward // PROPOSER_REWARD_QUOTIENT)
//    increase_balance(state, proposer_index, proposer_reward)
//    increase_balance(state, whistleblower_index, whistleblower_reward - proposer_reward)
func SlashValidator(state *pb.BeaconState, slashedIdx uint64, whistleBlowerIdx uint64) (*pb.BeaconState, error) {
	state, err := InitiateValidatorExit(state, slashedIdx)
	if err != nil {
		return nil, errors.Wrapf(err, "could not initiate validator %d exit", slashedIdx)
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
		return nil, errors.Wrap(err, "could not get proposer idx")
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
