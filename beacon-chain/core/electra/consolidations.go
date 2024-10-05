package electra

import (
	"bytes"
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v5/monitoring/tracing/trace"
	enginev1 "github.com/prysmaticlabs/prysm/v5/proto/engine/v1"
	eth "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/time/slots"
)

// ProcessPendingConsolidations implements the spec definition below. This method makes mutating
// calls to the beacon state.
//
// Spec definition:
//
//	def process_pending_consolidations(state: BeaconState) -> None:
//	    next_pending_consolidation = 0
//	    for pending_consolidation in state.pending_consolidations:
//	        source_validator = state.validators[pending_consolidation.source_index]
//	        if source_validator.slashed:
//	            next_pending_consolidation += 1
//	            continue
//	        if source_validator.withdrawable_epoch > get_current_epoch(state):
//	            break
//
//	        # Move active balance to target. Excess balance is withdrawable.
//	        active_balance = get_active_balance(state, pending_consolidation.source_index)
//	        decrease_balance(state, pending_consolidation.source_index, active_balance)
//	        increase_balance(state, pending_consolidation.target_index, active_balance)
//	        next_pending_consolidation += 1
//
//	    state.pending_consolidations = state.pending_consolidations[next_pending_consolidation:]
func ProcessPendingConsolidations(ctx context.Context, st state.BeaconState) error {
	_, span := trace.StartSpan(ctx, "electra.ProcessPendingConsolidations")
	defer span.End()

	if st == nil || st.IsNil() {
		return errors.New("nil state")
	}

	nextEpoch := slots.ToEpoch(st.Slot()) + 1

	var nextPendingConsolidation uint64
	pendingConsolidations, err := st.PendingConsolidations()
	if err != nil {
		return err
	}

	for _, pc := range pendingConsolidations {
		sourceValidator, err := st.ValidatorAtIndex(pc.SourceIndex)
		if err != nil {
			return err
		}
		if sourceValidator.Slashed {
			nextPendingConsolidation++
			continue
		}
		if sourceValidator.WithdrawableEpoch > nextEpoch {
			break
		}

		activeBalance, err := st.ActiveBalanceAtIndex(pc.SourceIndex)
		if err != nil {
			return err
		}
		if err := helpers.DecreaseBalance(st, pc.SourceIndex, activeBalance); err != nil {
			return err
		}
		if err := helpers.IncreaseBalance(st, pc.TargetIndex, activeBalance); err != nil {
			return err
		}
		nextPendingConsolidation++
	}

	if nextPendingConsolidation > 0 {
		return st.SetPendingConsolidations(pendingConsolidations[nextPendingConsolidation:])
	}

	return nil
}

// ProcessConsolidationRequests implements the spec definition below. This method makes mutating
// calls to the beacon state.
//
//	def process_consolidation_request(
//	    state: BeaconState,
//	    consolidation_request: ConsolidationRequest
//	) -> None:
//	    if is_valid_switch_to_compounding_request(state, consolidation_request):
//	        validator_pubkeys = [v.pubkey for v in state.validators]
//	        request_source_pubkey = consolidation_request.source_pubkey
//	        source_index = ValidatorIndex(validator_pubkeys.index(request_source_pubkey))
//	        switch_to_compounding_validator(state, source_index)
//	        return
//
//	    # Verify that source != target, so a consolidation cannot be used as an exit.
//	    if consolidation_request.source_pubkey == consolidation_request.target_pubkey:
//	        return
//	    # If the pending consolidations queue is full, consolidation requests are ignored
//	    if len(state.pending_consolidations) == PENDING_CONSOLIDATIONS_LIMIT:
//	        return
//	    # If there is too little available consolidation churn limit, consolidation requests are ignored
//	    if get_consolidation_churn_limit(state) <= MIN_ACTIVATION_BALANCE:
//	        return
//
//	    validator_pubkeys = [v.pubkey for v in state.validators]
//	    # Verify pubkeys exists
//	    request_source_pubkey = consolidation_request.source_pubkey
//	    request_target_pubkey = consolidation_request.target_pubkey
//	    if request_source_pubkey not in validator_pubkeys:
//	        return
//	    if request_target_pubkey not in validator_pubkeys:
//	        return
//	    source_index = ValidatorIndex(validator_pubkeys.index(request_source_pubkey))
//	    target_index = ValidatorIndex(validator_pubkeys.index(request_target_pubkey))
//	    source_validator = state.validators[source_index]
//	    target_validator = state.validators[target_index]
//
//	    # Verify source withdrawal credentials
//	    has_correct_credential = has_execution_withdrawal_credential(source_validator)
//	    is_correct_source_address = (
//	        source_validator.withdrawal_credentials[12:] == consolidation_request.source_address
//	    )
//	    if not (has_correct_credential and is_correct_source_address):
//	        return
//
//	    # Verify that target has execution withdrawal credentials
//	    if not has_execution_withdrawal_credential(target_validator):
//	        return
//
//	    # Verify the source and the target are active
//	    current_epoch = get_current_epoch(state)
//	    if not is_active_validator(source_validator, current_epoch):
//	        return
//	    if not is_active_validator(target_validator, current_epoch):
//	        return
//	    # Verify exits for source and target have not been initiated
//	    if source_validator.exit_epoch != FAR_FUTURE_EPOCH:
//	        return
//	    if target_validator.exit_epoch != FAR_FUTURE_EPOCH:
//	        return
//
//	    # Initiate source validator exit and append pending consolidation
//	    source_validator.exit_epoch = compute_consolidation_epoch_and_update_churn(
//	        state, source_validator.effective_balance
//	    )
//	    source_validator.withdrawable_epoch = Epoch(
//	        source_validator.exit_epoch + MIN_VALIDATOR_WITHDRAWABILITY_DELAY
//	    )
//	    state.pending_consolidations.append(PendingConsolidation(
//	        source_index=source_index,
//	        target_index=target_index
//	    ))
//
//	    # Churn any target excess active balance of target and raise its max
//	    if has_eth1_withdrawal_credential(target_validator):
//	        switch_to_compounding_validator(state, target_index)
func ProcessConsolidationRequests(ctx context.Context, st state.BeaconState, reqs []*enginev1.ConsolidationRequest) error {
	if len(reqs) == 0 || st == nil {
		return nil
	}

	activeBal, err := helpers.TotalActiveBalance(st)
	if err != nil {
		return err
	}
	churnLimit := helpers.ConsolidationChurnLimit(primitives.Gwei(activeBal))
	if churnLimit <= primitives.Gwei(params.BeaconConfig().MinActivationBalance) {
		return nil
	}
	curEpoch := slots.ToEpoch(st.Slot())
	ffe := params.BeaconConfig().FarFutureEpoch
	minValWithdrawDelay := params.BeaconConfig().MinValidatorWithdrawabilityDelay
	pcLimit := params.BeaconConfig().PendingConsolidationsLimit

	for _, cr := range reqs {
		if ctx.Err() != nil {
			return fmt.Errorf("cannot process consolidation requests: %w", ctx.Err())
		}
		canSwitch, err := IsValidSwitchToCompoundingRequest(ctx, st, cr)
		if err != nil {
			return fmt.Errorf("failed to validate consolidation request: %w", err)
		}
		if canSwitch {
			srcIdx, ok := st.ValidatorIndexByPubkey(bytesutil.ToBytes48(cr.SourcePubkey))
			if !ok {
				return errors.New("could not find validator in registry")
			}
			if err := SwitchToCompoundingValidator(st, srcIdx); err != nil {
				return fmt.Errorf("failed to switch to compounding validator: %w", err)
			}
			return nil
		}
		sourcePubkey := bytesutil.ToBytes48(cr.SourcePubkey)
		targetPubkey := bytesutil.ToBytes48(cr.TargetPubkey)
		if sourcePubkey == targetPubkey {
			continue
		}

		if npc, err := st.NumPendingConsolidations(); err != nil {
			return fmt.Errorf("failed to fetch number of pending consolidations: %w", err) // This should never happen.
		} else if npc >= pcLimit {
			return nil
		}

		srcIdx, ok := st.ValidatorIndexByPubkey(sourcePubkey)
		if !ok {
			continue
		}
		tgtIdx, ok := st.ValidatorIndexByPubkey(targetPubkey)
		if !ok {
			continue
		}

		srcV, err := st.ValidatorAtIndex(srcIdx)
		if err != nil {
			return fmt.Errorf("failed to fetch source validator: %w", err) // This should never happen.
		}

		tgtV, err := st.ValidatorAtIndexReadOnly(tgtIdx)
		if err != nil {
			return fmt.Errorf("failed to fetch target validator: %w", err) // This should never happen.
		}

		// Verify source withdrawal credentials
		if !helpers.HasExecutionWithdrawalCredentials(srcV) {
			continue
		}
		// Confirm source_validator.withdrawal_credentials[12:] == consolidation_request.source_address
		if len(srcV.WithdrawalCredentials) != 32 || len(cr.SourceAddress) != 20 || !bytes.HasSuffix(srcV.WithdrawalCredentials, cr.SourceAddress) {
			continue
		}

		// Target validator must have their withdrawal credentials set appropriately.
		if !helpers.HasExecutionWithdrawalCredentials(tgtV) {
			continue
		}

		// Both validators must be active.
		if !helpers.IsActiveValidator(srcV, curEpoch) || !helpers.IsActiveValidatorUsingTrie(tgtV, curEpoch) {
			continue
		}
		// Neither validator are exiting.
		if srcV.ExitEpoch != ffe || tgtV.ExitEpoch() != ffe {
			continue
		}

		// Initiate the exit of the source validator.
		exitEpoch, err := ComputeConsolidationEpochAndUpdateChurn(ctx, st, primitives.Gwei(srcV.EffectiveBalance))
		if err != nil {
			return fmt.Errorf("failed to compute consolidaiton epoch: %w", err)
		}
		srcV.ExitEpoch = exitEpoch
		srcV.WithdrawableEpoch = exitEpoch + minValWithdrawDelay
		if err := st.UpdateValidatorAtIndex(srcIdx, srcV); err != nil {
			return fmt.Errorf("failed to update validator: %w", err) // This should never happen.
		}

		if err := st.AppendPendingConsolidation(&eth.PendingConsolidation{SourceIndex: srcIdx, TargetIndex: tgtIdx}); err != nil {
			return fmt.Errorf("failed to append pending consolidation: %w", err) // This should never happen.
		}

		if helpers.HasETH1WithdrawalCredential(tgtV) {
			if err := SwitchToCompoundingValidator(st, tgtIdx); err != nil {
				return fmt.Errorf("failed to switch to compounding validator: %w", err)
			}
		}
	}

	return nil
}

// IsValidSwitchToCompoundingRequest returns true if the given consolidation request is valid for switching to compounding.
//
// Spec code:
//
// def is_valid_switch_to_compounding_request(
//
//	state: BeaconState,
//	consolidation_request: ConsolidationRequest
//
// ) -> bool:
//
//	# Switch to compounding requires source and target be equal
//	if consolidation_request.source_pubkey != consolidation_request.target_pubkey:
//	    return False
//
//	# Verify pubkey exists
//	source_pubkey = consolidation_request.source_pubkey
//	validator_pubkeys = [v.pubkey for v in state.validators]
//	if source_pubkey not in validator_pubkeys:
//	    return False
//
//	source_validator = state.validators[ValidatorIndex(validator_pubkeys.index(source_pubkey))]
//
//	# Verify request has been authorized
//	if source_validator.withdrawal_credentials[12:] != consolidation_request.source_address:
//	    return False
//
//	# Verify source withdrawal credentials
//	if not has_eth1_withdrawal_credential(source_validator):
//	    return False
//
//	# Verify the source is active
//	current_epoch = get_current_epoch(state)
//	if not is_active_validator(source_validator, current_epoch):
//	    return False
//
//	# Verify exit for source have not been initiated
//	if source_validator.exit_epoch != FAR_FUTURE_EPOCH:
//	    return False
//
//	return True
func IsValidSwitchToCompoundingRequest(ctx context.Context, st state.BeaconState, req *enginev1.ConsolidationRequest) (bool, error) {
	if req.SourcePubkey == nil || req.TargetPubkey == nil {
		return false, errors.New("nil source or target pubkey")
	}

	sourcePubKey := bytesutil.ToBytes48(req.SourcePubkey)
	targetPubKey := bytesutil.ToBytes48(req.TargetPubkey)
	if sourcePubKey != targetPubKey {
		return false, nil
	}

	srcIdx, ok := st.ValidatorIndexByPubkey(sourcePubKey)
	if !ok {
		return false, nil
	}
	srcV, err := st.ValidatorAtIndex(srcIdx)
	if err != nil {
		return false, err
	}
	sourceAddress := req.SourceAddress
	withdrawalCreds := srcV.WithdrawalCredentials
	if len(withdrawalCreds) != 32 || len(sourceAddress) != 20 || !bytes.HasSuffix(withdrawalCreds, sourceAddress) {
		return false, nil
	}

	if !helpers.HasETH1WithdrawalCredential(srcV) {
		return false, nil
	}

	curEpoch := slots.ToEpoch(st.Slot())
	if !helpers.IsActiveValidator(srcV, curEpoch) {
		return false, nil
	}

	if srcV.ExitEpoch != params.BeaconConfig().FarFutureEpoch {
		return false, nil
	}
	return true, nil
}
