package electra

import (
	"context"
	"fmt"

	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/time"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/validators"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
)

// sortableIndices implements the Sort interface to sort newly activated validator indices
// by activation epoch and by index number.
type sortableIndices struct {
	indices    []primitives.ValidatorIndex
	validators []*ethpb.Validator
}

// Len is the number of elements in the collection.
func (s sortableIndices) Len() int { return len(s.indices) }

// Swap swaps the elements with indexes i and j.
func (s sortableIndices) Swap(i, j int) { s.indices[i], s.indices[j] = s.indices[j], s.indices[i] }

// Less reports whether the element with index i must sort before the element with index j.
func (s sortableIndices) Less(i, j int) bool {
	if s.validators[s.indices[i]].ActivationEligibilityEpoch == s.validators[s.indices[j]].ActivationEligibilityEpoch {
		return s.indices[i] < s.indices[j]
	}
	return s.validators[s.indices[i]].ActivationEligibilityEpoch < s.validators[s.indices[j]].ActivationEligibilityEpoch
}

// ProcessRegistryUpdates rotates validators in and out of active pool.
// the amount to rotate is determined churn limit.
//
// Spec pseudocode definition:
//
//	def process_registry_updates(state: BeaconState) -> None:
//	    # Process activation eligibility and ejections
//	    for index, validator in enumerate(state.validators):
//	        if is_eligible_for_activation_queue(validator):
//	            validator.activation_eligibility_epoch = get_current_epoch(state) + 1
//
//	        if (
//	            is_active_validator(validator, get_current_epoch(state))
//	            and validator.effective_balance <= EJECTION_BALANCE
//	        ):
//	            initiate_validator_exit(state, ValidatorIndex(index))
//
//	    # Activate all eligible validators
//	    activation_epoch = compute_activation_exit_epoch(get_current_epoch(state))
//	    for validator in state.validators:
//	        if is_eligible_for_activation(state, validator):
//	            validator.activation_epoch = activation_epoch
func ProcessRegistryUpdates(ctx context.Context, state state.BeaconState) (state.BeaconState, error) {
	currentEpoch := time.CurrentEpoch(state)
	ejectionBal := params.BeaconConfig().EjectionBalance
	activationEpoch := helpers.ActivationExitEpoch(currentEpoch)
	vals := state.Validators()
	for idx, val := range vals {
		if helpers.IsEligibleForActivationQueue(val, currentEpoch) {
			val.ActivationEligibilityEpoch = currentEpoch + 1
		}
		if helpers.IsActiveValidator(val, currentEpoch) && val.EffectiveBalance <= ejectionBal {
			var err error
			maxExitEpoch, churn := validators.MaxExitEpochAndChurn(state)
			state, _, err = validators.InitiateValidatorExit(ctx, state, primitives.ValidatorIndex(idx), maxExitEpoch, churn)
			if err != nil {
				return nil, err
			}
		}

		if helpers.IsEligibleForActivation(state, val) {
			val.ActivationEpoch = activationEpoch
			if err := state.UpdateValidatorAtIndex(primitives.ValidatorIndex(idx), val); err != nil {
				return nil, err
			}
		}
	}

	return state, nil
}

// ProcessEffectiveBalanceUpdates processes effective balance updates during epoch processing.
//
// Spec pseudocode definition:
//
//	def process_effective_balance_updates(state: BeaconState) -> None:
//	    # Update effective balances with hysteresis
//	    for index, validator in enumerate(state.validators):
//	        balance = state.balances[index]
//	        HYSTERESIS_INCREMENT = uint64(EFFECTIVE_BALANCE_INCREMENT // HYSTERESIS_QUOTIENT)
//	        DOWNWARD_THRESHOLD = HYSTERESIS_INCREMENT * HYSTERESIS_DOWNWARD_MULTIPLIER
//	        UPWARD_THRESHOLD = HYSTERESIS_INCREMENT * HYSTERESIS_UPWARD_MULTIPLIER
//	        EFFECTIVE_BALANCE_LIMIT = (
//	            MAX_EFFECTIVE_BALANCE_EIP7251 if has_compounding_withdrawal_credential(validator)
//	            else MIN_ACTIVATION_BALANCE
//	        )
//
//	        if (
//	            balance + DOWNWARD_THRESHOLD < validator.effective_balance
//	            or validator.effective_balance + UPWARD_THRESHOLD < balance
//	        ):
//	            validator.effective_balance = min(balance - balance % EFFECTIVE_BALANCE_INCREMENT, EFFECTIVE_BALANCE_LIMIT)
func ProcessEffectiveBalanceUpdates(state state.BeaconState) (state.BeaconState, error) {
	effBalanceInc := params.BeaconConfig().EffectiveBalanceIncrement
	hysteresisInc := effBalanceInc / params.BeaconConfig().HysteresisQuotient
	downwardThreshold := hysteresisInc * params.BeaconConfig().HysteresisDownwardMultiplier
	upwardThreshold := hysteresisInc * params.BeaconConfig().HysteresisUpwardMultiplier

	bals := state.Balances()

	// Update effective balances with hysteresis.
	validatorFunc := func(idx int, val *ethpb.Validator) (bool, *ethpb.Validator, error) {
		if val == nil {
			return false, nil, fmt.Errorf("validator %d is nil in state", idx)
		}
		if idx >= len(bals) {
			return false, nil, fmt.Errorf("validator index exceeds validator length in state %d >= %d", idx, len(state.Balances()))
		}
		balance := bals[idx]

		effectiveBalanceLimit := params.BeaconConfig().MinActivationBalance
		if helpers.HasCompoundingWithdrawalCredential(val) {
			effectiveBalanceLimit = params.BeaconConfig().MaxEffectiveBalanceElectra
		}

		if balance+downwardThreshold < val.EffectiveBalance || val.EffectiveBalance+upwardThreshold < balance {
			effectiveBal := min(balance-balance%effBalanceInc, effectiveBalanceLimit)
			if effectiveBal != val.EffectiveBalance {
				newVal := ethpb.CopyValidator(val)
				newVal.EffectiveBalance = effectiveBal
				return true, newVal, nil
			}
			return false, val, nil
		}
		return false, val, nil
	}

	if err := state.ApplyToEveryValidator(validatorFunc); err != nil {
		return nil, err
	}

	return state, nil
}
