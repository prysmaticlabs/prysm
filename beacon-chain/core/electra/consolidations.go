package electra

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/signing"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/crypto/bls"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/time/slots"
	"go.opencensus.io/trace"
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
//	        # Churn any target excess active balance of target and raise its max
//	        switch_to_compounding_validator(state, pending_consolidation.target_index)
//	        # Move active balance to target. Excess balance is withdrawable.
//	        active_balance = get_active_balance(state, pending_consolidation.source_index)
//	        decrease_balance(state, pending_consolidation.source_index, active_balance)
//	        increase_balance(state, pending_consolidation.target_index, active_balance)
//	        next_pending_consolidation += 1
//
//	    state.pending_consolidations = state.pending_consolidations[next_pending_consolidation:]
func ProcessPendingConsolidations(ctx context.Context, st state.BeaconState) error {
	ctx, span := trace.StartSpan(ctx, "electra.ProcessPendingConsolidations")
	defer span.End()

	if st == nil || st.IsNil() {
		return errors.New("nil state")
	}

	currentEpoch := slots.ToEpoch(st.Slot())

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
		if sourceValidator.WithdrawableEpoch > currentEpoch {
			break
		}

		if err := SwitchToCompoundingValidator(ctx, st, pc.TargetIndex); err != nil {
			return err
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

// ProcessConsolidations implements the spec definition below. This method makes mutating calls to
// the beacon state.
//
// Spec definition:
//
//	def process_consolidation(state: BeaconState, signed_consolidation: SignedConsolidation) -> None:
//	    # If the pending consolidations queue is full, no consolidations are allowed in the block
//	    assert len(state.pending_consolidations) < PENDING_CONSOLIDATIONS_LIMIT
//	    # If there is too little available consolidation churn limit, no consolidations are allowed in the block
//	    assert get_consolidation_churn_limit(state) > MIN_ACTIVATION_BALANCE
//	    consolidation = signed_consolidation.message
//	    # Verify that source != target, so a consolidation cannot be used as an exit.
//	    assert consolidation.source_index != consolidation.target_index
//
//	    source_validator = state.validators[consolidation.source_index]
//	    target_validator = state.validators[consolidation.target_index]
//	    # Verify the source and the target are active
//	    current_epoch = get_current_epoch(state)
//	    assert is_active_validator(source_validator, current_epoch)
//	    assert is_active_validator(target_validator, current_epoch)
//	    # Verify exits for source and target have not been initiated
//	    assert source_validator.exit_epoch == FAR_FUTURE_EPOCH
//	    assert target_validator.exit_epoch == FAR_FUTURE_EPOCH
//	    # Consolidations must specify an epoch when they become valid; they are not valid before then
//	    assert current_epoch >= consolidation.epoch
//
//	    # Verify the source and the target have Execution layer withdrawal credentials
//	    assert has_execution_withdrawal_credential(source_validator)
//	    assert has_execution_withdrawal_credential(target_validator)
//	    # Verify the same withdrawal address
//	    assert source_validator.withdrawal_credentials[12:] == target_validator.withdrawal_credentials[12:]
//
//	    # Verify consolidation is signed by the source and the target
//	    domain = compute_domain(DOMAIN_CONSOLIDATION, genesis_validators_root=state.genesis_validators_root)
//	    signing_root = compute_signing_root(consolidation, domain)
//	    pubkeys = [source_validator.pubkey, target_validator.pubkey]
//	    assert bls.FastAggregateVerify(pubkeys, signing_root, signed_consolidation.signature)
//
//	    # Initiate source validator exit and append pending consolidation
//	    source_validator.exit_epoch = compute_consolidation_epoch_and_update_churn(
//	        state, source_validator.effective_balance)
//	    source_validator.withdrawable_epoch = Epoch(
//	        source_validator.exit_epoch + MIN_VALIDATOR_WITHDRAWABILITY_DELAY
//	    )
//	    state.pending_consolidations.append(PendingConsolidation(
//	        source_index=consolidation.source_index,
//	        target_index=consolidation.target_index
//	    ))
func ProcessConsolidations(ctx context.Context, st state.BeaconState, cs []*ethpb.SignedConsolidation) error {
	_, span := trace.StartSpan(ctx, "electra.ProcessConsolidations")
	defer span.End()

	if st == nil || st.IsNil() {
		return errors.New("nil state")
	}

	if len(cs) == 0 {
		return nil // Nothing to process.
	}

	domain, err := signing.ComputeDomain(
		params.BeaconConfig().DomainConsolidation,
		nil, // Use genesis fork version
		st.GenesisValidatorsRoot(),
	)
	if err != nil {
		return err
	}

	totalBalance, err := helpers.TotalActiveBalance(st)
	if err != nil {
		return err
	}

	if helpers.ConsolidationChurnLimit(primitives.Gwei(totalBalance)) <= primitives.Gwei(params.BeaconConfig().MinActivationBalance) {
		return errors.New("too little available consolidation churn limit")
	}

	currentEpoch := slots.ToEpoch(st.Slot())

	for _, c := range cs {
		if c == nil || c.Message == nil {
			return errors.New("nil consolidation")
		}

		if n, err := st.NumPendingConsolidations(); err != nil {
			return err
		} else if n >= params.BeaconConfig().PendingConsolidationsLimit {
			return errors.New("pending consolidations queue is full")
		}

		if c.Message.SourceIndex == c.Message.TargetIndex {
			return errors.New("source and target index are the same")
		}
		source, err := st.ValidatorAtIndex(c.Message.SourceIndex)
		if err != nil {
			return err
		}
		target, err := st.ValidatorAtIndex(c.Message.TargetIndex)
		if err != nil {
			return err
		}
		if !helpers.IsActiveValidator(source, currentEpoch) {
			return errors.New("source is not active")
		}
		if !helpers.IsActiveValidator(target, currentEpoch) {
			return errors.New("target is not active")
		}
		if source.ExitEpoch != params.BeaconConfig().FarFutureEpoch {
			return errors.New("source exit epoch has been initiated")
		}
		if target.ExitEpoch != params.BeaconConfig().FarFutureEpoch {
			return errors.New("target exit epoch has been initiated")
		}
		if currentEpoch < c.Message.Epoch {
			return errors.New("consolidation is not valid yet")
		}

		if !helpers.HasExecutionWithdrawalCredentials(source) {
			return errors.New("source does not have execution withdrawal credentials")
		}
		if !helpers.HasExecutionWithdrawalCredentials(target) {
			return errors.New("target does not have execution withdrawal credentials")
		}
		if !helpers.IsSameWithdrawalCredentials(source, target) {
			return errors.New("source and target have different withdrawal credentials")
		}

		sr, err := signing.ComputeSigningRoot(c.Message, domain)
		if err != nil {
			return err
		}
		sourcePk, err := bls.PublicKeyFromBytes(source.PublicKey)
		if err != nil {
			return errors.Wrap(err, "could not convert source public key bytes to bls public key")
		}
		targetPk, err := bls.PublicKeyFromBytes(target.PublicKey)
		if err != nil {
			return errors.Wrap(err, "could not convert target public key bytes to bls public key")
		}
		sig, err := bls.SignatureFromBytes(c.Signature)
		if err != nil {
			return errors.Wrap(err, "could not convert bytes to signature")
		}
		if !sig.FastAggregateVerify([]bls.PublicKey{sourcePk, targetPk}, sr) {
			return errors.New("consolidation signature verification failed")
		}

		sEE, err := ComputeConsolidationEpochAndUpdateChurn(ctx, st, primitives.Gwei(source.EffectiveBalance))
		if err != nil {
			return err
		}
		source.ExitEpoch = sEE
		source.WithdrawableEpoch = sEE + params.BeaconConfig().MinValidatorWithdrawabilityDelay
		if err := st.UpdateValidatorAtIndex(c.Message.SourceIndex, source); err != nil {
			return err
		}
		if err := st.AppendPendingConsolidation(c.Message.ToPendingConsolidation()); err != nil {
			return err
		}
	}

	return nil
}
