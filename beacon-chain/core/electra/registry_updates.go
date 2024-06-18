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
)

// ProcessRegistryUpdates processes all validators eligible for the activation queue, all validators
// which should be ejected, and all validators which are eligible for activation from the queue.
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
func ProcessRegistryUpdates(ctx context.Context, state state.BeaconState) error {
	currentEpoch := time.CurrentEpoch(state)
	ejectionBal := params.BeaconConfig().EjectionBalance
	activationEpoch := helpers.ActivationExitEpoch(currentEpoch)
	vals := state.Validators()
	for idx, val := range vals {
		// Handle validators eligible to join the activation queue.
		if helpers.IsEligibleForActivationQueue(val, currentEpoch) {
			val.ActivationEligibilityEpoch = currentEpoch + 1
			if err := state.UpdateValidatorAtIndex(primitives.ValidatorIndex(idx), val); err != nil {
				return fmt.Errorf("failed to update eligible validator at index %d: %w", idx, err)
			}
		}
		// Handle validator ejections.
		if val.EffectiveBalance <= ejectionBal && helpers.IsActiveValidator(val, currentEpoch) {
			var err error
			// exitQueueEpoch and churn arguments are not used in electra.
			state, _, err = validators.InitiateValidatorExit(ctx, state, primitives.ValidatorIndex(idx), 0 /*exitQueueEpoch*/, 0 /*churn*/)
			if err != nil {
				return fmt.Errorf("failed to initiate validator exit at index %d: %w", idx, err)
			}
		}

		// Activate all eligible validators.
		if helpers.IsEligibleForActivation(state, val) {
			val.ActivationEpoch = activationEpoch
			if err := state.UpdateValidatorAtIndex(primitives.ValidatorIndex(idx), val); err != nil {
				return fmt.Errorf("failed to activate validator at index %d: %w", idx, err)
			}
		}
	}

	return nil
}
