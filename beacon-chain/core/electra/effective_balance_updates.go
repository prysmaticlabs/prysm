package electra

import (
	"fmt"

	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
)

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
func ProcessEffectiveBalanceUpdates(state state.BeaconState) error {
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
			newVal := ethpb.CopyValidator(val)
			newVal.EffectiveBalance = effectiveBal
			return true, newVal, nil
		}
		return false, val, nil
	}

	return state.ApplyToEveryValidator(validatorFunc)
}
