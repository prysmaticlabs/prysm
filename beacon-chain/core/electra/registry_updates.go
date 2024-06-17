package electra

import (
	"context"

	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/time"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/validators"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
)

// ProcessRegistryUpdates rotates validators in and out of active pool.
//
// Spec pseudocode definition:
//
//		def process_registry_updates(state: BeaconState) -> None:
//		    # Process activation eligibility and ejections
//		    for index, validator in enumerate(state.validators):
//		        if is_eligible_for_activation_queue(validator):
//		            validator.activation_eligibility_epoch = get_current_epoch(state) + 1
//
//		        if (
//		            is_active_validator(validator, get_current_epoch(state))
//		            and validator.effective_balance <= EJECTION_BALANCE
//		        ):
//		            initiate_validator_exit(state, ValidatorIndex(index))
//
//	         # Activate all eligible validators
//	         activation_epoch = compute_activation_exit_epoch(get_current_epoch(state))
//	         for validator in state.validators:
//	             if is_eligible_for_activation(state, validator):
//	                 validator.activation_epoch = activation_epoch
func ProcessRegistryUpdates(ctx context.Context, state state.BeaconState) (state.BeaconState, error) {
	currentEpoch := time.CurrentEpoch(state)
	ejectionBal := params.BeaconConfig().EjectionBalance
	activationEpoch := helpers.ActivationExitEpoch(currentEpoch)
	vals := state.Validators()
	for idx, val := range vals {
		if helpers.IsEligibleForActivationQueue(val, currentEpoch) {
			val.ActivationEligibilityEpoch = currentEpoch + 1
			if err := state.UpdateValidatorAtIndex(primitives.ValidatorIndex(idx), val); err != nil {
				return nil, err
			}
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
