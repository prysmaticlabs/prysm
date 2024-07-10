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

	// To avoid copying the state validator set via st.Validators(), we will perform a read only pass
	// over the validator set while collecting validator indices where the validator copy is actually
	// necessary, then we will process these operations.
	eligibleForActivationQ := make([]primitives.ValidatorIndex, 0)
	eligibleForEjection := make([]primitives.ValidatorIndex, 0)
	eligibleForActivation := make([]primitives.ValidatorIndex, 0)

	if err := st.ReadFromEveryValidator(func(idx int, val state.ReadOnlyValidator) error {
		// Collect validators eligible to enter the activation queue.
		if helpers.IsEligibleForActivationQueue(val, currentEpoch) {
			eligibleForActivationQ = append(eligibleForActivationQ, primitives.ValidatorIndex(idx))
		}

		// Collect validators to eject.
		if val.EffectiveBalance() <= ejectionBal && helpers.IsActiveValidatorUsingTrie(val, currentEpoch) {
			eligibleForEjection = append(eligibleForEjection, primitives.ValidatorIndex(idx))
		}

		// Collect validators eligible for activation and not yet dequeued for activation.
		if helpers.IsEligibleForActivationUsingROVal(st, val) {
			eligibleForActivation = append(eligibleForActivation, primitives.ValidatorIndex(idx))
		}

		return nil
	}); err != nil {
		return fmt.Errorf("failed to read validators: %w", err)
	}

	// Handle validators eligible to join the activation queue.
	for _, idx := range eligibleForActivationQ {
		v, err := st.ValidatorAtIndex(idx)
		if err != nil {
			return err
		}
		v.ActivationEligibilityEpoch = currentEpoch + 1
		if err := st.UpdateValidatorAtIndex(idx, v); err != nil {
			return fmt.Errorf("failed to updated eligible validator at index %d: %w", idx, err)
		}
	}

	// Handle validator ejections.
	for _, idx := range eligibleForEjection {
		var err error
		// exitQueueEpoch and churn arguments are not used in electra.
		st, _, err = validators.InitiateValidatorExit(ctx, st, idx, 0 /*exitQueueEpoch*/, 0 /*churn*/)
		if err != nil {
			return fmt.Errorf("failed to initiate validator exit at index %d: %w", idx, err)
		}
	}

	for _, idx := range eligibleForActivation {
		// Activate all eligible validators.
		v, err := st.ValidatorAtIndex(idx)
		if err != nil {
			return err
		}
		v.ActivationEpoch = activationEpoch
		if err := st.UpdateValidatorAtIndex(idx, v); err != nil {
			return fmt.Errorf("failed to activate validator at index %d: %w", idx, err)
		}
	}

	return nil
}
