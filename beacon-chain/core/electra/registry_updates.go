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
func ProcessRegistryUpdates(ctx context.Context, st state.BeaconState) error {
	currentEpoch := time.CurrentEpoch(st)
	ejectionBal := params.BeaconConfig().EjectionBalance
	activationEpoch := helpers.ActivationExitEpoch(currentEpoch)

	finalizedEpoch := st.FinalizedCheckpointEpoch()
	eligibleForActivationQueueValidators := make([]primitives.ValidatorIndex, 0)
	eligibleForActivationValidators := make([]primitives.ValidatorIndex, 0)
	if err := st.ReadFromEveryValidator(func(idx int, val state.ReadOnlyValidator) error {
		alreadyActivated := false
		if helpers.IsEligibleForActivationQueue(val, currentEpoch) {
			eligibleForActivationQueueValidators = append(eligibleForActivationQueueValidators, primitives.ValidatorIndex(idx))

			if currentEpoch+1 <= finalizedEpoch && val.ActivationEpoch() == params.BeaconConfig().FarFutureEpoch {
				eligibleForActivationValidators = append(eligibleForActivationValidators, primitives.ValidatorIndex(idx))
				alreadyActivated = true
			}
		}

		if val.EffectiveBalance() <= ejectionBal && helpers.IsActiveValidator(val, currentEpoch) {
			var err error
			// exitQueueEpoch and churn arguments are not used in electra.
			st, _, err = validators.InitiateValidatorExit(ctx, st, primitives.ValidatorIndex(idx), 0 /*exitQueueEpoch*/, 0 /*churn*/)
			if err != nil {
				return fmt.Errorf("failed to initiate validator exit at index %d: %w", idx, err)
			}
		}

		if !alreadyActivated && helpers.IsEligibleForActivation(st, val) {
			eligibleForActivationValidators = append(eligibleForActivationValidators, primitives.ValidatorIndex(idx))
		}
		return nil
	}); err != nil {
		return err
	}

	for _, idx := range eligibleForActivationQueueValidators {
		val, err := st.ValidatorAtIndex(idx)
		if err != nil {
			return fmt.Errorf("failed to get validator at index %d: %w", idx, err)
		}
		val.ActivationEligibilityEpoch = currentEpoch + 1
		if err := st.UpdateValidatorAtIndex(idx, val); err != nil {
			return fmt.Errorf("failed to update eligible validator at index %d: %w", idx, err)
		}
	}

	for _, idx := range eligibleForActivationValidators {
		val, err := st.ValidatorAtIndex(idx)
		if err != nil {
			return fmt.Errorf("failed to get validator at index %d: %w", idx, err)
		}
		val.ActivationEpoch = activationEpoch
		if err := st.UpdateValidatorAtIndex(idx, val); err != nil {
			return fmt.Errorf("failed to activate validator at index %d: %w", idx, err)
		}
	}

	return nil
}
