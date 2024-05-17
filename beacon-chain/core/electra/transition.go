package electra

import (
	"bytes"
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/altair"
	b "github.com/prysmaticlabs/prysm/v5/beacon-chain/core/blocks"
	e "github.com/prysmaticlabs/prysm/v5/beacon-chain/core/epoch"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/epoch/precompute"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/validators"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v5/math"
	enginev1 "github.com/prysmaticlabs/prysm/v5/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/time/slots"
	"go.opencensus.io/trace"
)

// Re-exports for methods that haven't changed in Electra.
var (
	InitializePrecomputeValidators       = altair.InitializePrecomputeValidators
	ProcessEpochParticipation            = altair.ProcessEpochParticipation
	ProcessInactivityScores              = altair.ProcessInactivityScores
	ProcessRewardsAndPenaltiesPrecompute = altair.ProcessRewardsAndPenaltiesPrecompute
	ProcessSlashings                     = e.ProcessSlashings
	ProcessEth1DataReset                 = e.ProcessEth1DataReset
	ProcessSlashingsReset                = e.ProcessSlashingsReset
	ProcessRandaoMixesReset              = e.ProcessRandaoMixesReset
	ProcessHistoricalDataUpdate          = e.ProcessHistoricalDataUpdate
	ProcessParticipationFlagUpdates      = altair.ProcessParticipationFlagUpdates
	ProcessSyncCommitteeUpdates          = altair.ProcessSyncCommitteeUpdates
	AttestationsDelta                    = altair.AttestationsDelta
	ProcessSyncAggregate                 = altair.ProcessSyncAggregate

	// These need to be rewritten for electra.
	ProcessDeposits                      = altair.ProcessDeposits
	ProcessAttestationsNoVerifySignature = altair.ProcessAttestationsNoVerifySignature
)

// ProcessEpoch describes the per epoch operations that are performed on the beacon state.
// It's optimized by pre computing validator attested info and epoch total/attested balances upfront.
//
// Spec definition:
//
//	def process_epoch(state: BeaconState) -> None:
//	    process_justification_and_finalization(state)
//	    process_inactivity_updates(state)
//	    process_rewards_and_penalties(state)
//	    process_registry_updates(state)
//	    process_slashings(state)
//	    process_eth1_data_reset(state)
//	    process_pending_balance_deposits(state)  # New in EIP7251
//	    process_pending_consolidations(state)  # New in EIP7251
//	    process_effective_balance_updates(state)
//	    process_slashings_reset(state)
//	    process_randao_mixes_reset(state)
func ProcessEpoch(ctx context.Context, state state.BeaconState) (state.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "electra.ProcessEpoch")
	defer span.End()

	if state == nil || state.IsNil() {
		return nil, errors.New("nil state")
	}
	vp, bp, err := InitializePrecomputeValidators(ctx, state)
	if err != nil {
		return nil, err
	}
	vp, bp, err = ProcessEpochParticipation(ctx, state, bp, vp)
	if err != nil {
		return nil, err
	}
	state, err = precompute.ProcessJustificationAndFinalizationPreCompute(state, bp)
	if err != nil {
		return nil, errors.Wrap(err, "could not process justification")
	}
	state, vp, err = ProcessInactivityScores(ctx, state, vp)
	if err != nil {
		return nil, errors.Wrap(err, "could not process inactivity updates")
	}
	state, err = ProcessRewardsAndPenaltiesPrecompute(state, bp, vp)
	if err != nil {
		return nil, errors.Wrap(err, "could not process rewards and penalties")
	}
	state, err = ProcessRegistryUpdates(ctx, state)
	if err != nil {
		return nil, errors.Wrap(err, "could not process registry updates")
	}
	proportionalSlashingMultiplier, err := state.ProportionalSlashingMultiplier()
	if err != nil {
		return nil, err
	}
	state, err = ProcessSlashings(state, proportionalSlashingMultiplier)
	if err != nil {
		return nil, err
	}
	state, err = ProcessEth1DataReset(state)
	if err != nil {
		return nil, err
	}
	if err = ProcessPendingBalanceDeposits(ctx, state, math.Gwei(bp.ActiveCurrentEpoch)); err != nil {
		return nil, err
	}
	if err := ProcessPendingConsolidations(ctx, state); err != nil {
		return nil, err
	}
	if err := ProcessEffectiveBalanceUpdates(state); err != nil {
		return nil, err
	}
	state, err = ProcessSlashingsReset(state)
	if err != nil {
		return nil, err
	}
	state, err = ProcessRandaoMixesReset(state)
	if err != nil {
		return nil, err
	}
	state, err = ProcessHistoricalDataUpdate(state)
	if err != nil {
		return nil, err
	}

	state, err = ProcessParticipationFlagUpdates(state)
	if err != nil {
		return nil, err
	}

	state, err = ProcessSyncCommitteeUpdates(ctx, state)
	if err != nil {
		return nil, err
	}

	return state, nil
}

// ProcessExecutionLayerWithdrawRequests
//
// Spec definition:
//
//	def process_execution_layer_withdrawal_request(
//	    state: BeaconState,
//	    execution_layer_withdrawal_request: ExecutionLayerWithdrawalRequest
//	) -> None:
//	    amount = execution_layer_withdrawal_request.amount
//	    is_full_exit_request = amount == FULL_EXIT_REQUEST_AMOUNT
//
//	    # If partial withdrawal queue is full, only full exits are processed
//	    if len(state.pending_partial_withdrawals) == PENDING_PARTIAL_WITHDRAWALS_LIMIT and not is_full_exit_request:
//	        return
//
//	    validator_pubkeys = [v.pubkey for v in state.validators]
//	    # Verify pubkey exists
//	    request_pubkey = execution_layer_withdrawal_request.validator_pubkey
//	    if request_pubkey not in validator_pubkeys:
//	        return
//	    index = ValidatorIndex(validator_pubkeys.index(request_pubkey))
//	    validator = state.validators[index]
//
//	    # Verify withdrawal credentials
//	    has_correct_credential = has_execution_withdrawal_credential(validator)
//	    is_correct_source_address = (
//	        validator.withdrawal_credentials[12:] == execution_layer_withdrawal_request.source_address
//	    )
//	    if not (has_correct_credential and is_correct_source_address):
//	        return
//	    # Verify the validator is active
//	    if not is_active_validator(validator, get_current_epoch(state)):
//	        return
//	    # Verify exit has not been initiated
//	    if validator.exit_epoch != FAR_FUTURE_EPOCH:
//	        return
//	    # Verify the validator has been active long enough
//	    if get_current_epoch(state) < validator.activation_epoch + SHARD_COMMITTEE_PERIOD:
//	        return
//
//	    pending_balance_to_withdraw = get_pending_balance_to_withdraw(state, index)
//
//	    if is_full_exit_request:
//	        # Only exit validator if it has no pending withdrawals in the queue
//	        if pending_balance_to_withdraw == 0:
//	            initiate_validator_exit(state, index)
//	        return
//
//	    has_sufficient_effective_balance = validator.effective_balance >= MIN_ACTIVATION_BALANCE
//	    has_excess_balance = state.balances[index] > MIN_ACTIVATION_BALANCE + pending_balance_to_withdraw
//
//	    # Only allow partial withdrawals with compounding withdrawal credentials
//	    if has_compounding_withdrawal_credential(validator) and has_sufficient_effective_balance and has_excess_balance:
//	        to_withdraw = min(
//	            state.balances[index] - MIN_ACTIVATION_BALANCE - pending_balance_to_withdraw,
//	            amount
//	        )
//	        exit_queue_epoch = compute_exit_epoch_and_update_churn(state, to_withdraw)
//	        withdrawable_epoch = Epoch(exit_queue_epoch + MIN_VALIDATOR_WITHDRAWABILITY_DELAY)
//	        state.pending_partial_withdrawals.append(PendingPartialWithdrawal(
//	            index=index,
//	            amount=to_withdraw,
//	            withdrawable_epoch=withdrawable_epoch,
//	        ))
func ProcessExecutionLayerWithdrawRequests(ctx context.Context, st state.BeaconState, wrs []*enginev1.ExecutionLayerWithdrawalRequest) (state.BeaconState, error) {
	// TODO: Use PR 13888
	for _, wr := range wrs {
		if wr == nil {
			return nil, errors.New("nil execution layer withdrawal request")
		}
		amount := wr.Amount
		isFullExitRequest := amount == params.BeaconConfig().FullExitRequestAmount
		// If partial withdrawal queue is full, only full exits are processed
		if n, err := st.NumPendingPartialWithdrawals(); err != nil {
			return nil, err
		} else if n == params.BeaconConfig().PendingPartialWithdrawalsLimit && !isFullExitRequest {
			continue
		}

		vIdx, exists := st.ValidatorIndexByPubkey(bytesutil.ToBytes48(wr.ValidatorPubkey))
		if !exists {
			continue
		}
		validator, err := st.ValidatorAtIndex(vIdx)
		if err != nil {
			return nil, err
		}
		hasCorrectCredential := helpers.HasExecutionWithdrawalCredentials(validator)
		isCorrectSourceAddress := bytes.Equal(validator.WithdrawalCredentials[12:], wr.SourceAddress) // TODO: Is the correct way to perform this check safely?
		if !hasCorrectCredential || !isCorrectSourceAddress {
			continue
		}
		if !helpers.IsActiveValidator(validator, slots.ToEpoch(st.Slot())) {
			continue
		}
		if validator.ExitEpoch != params.BeaconConfig().FarFutureEpoch {
			continue
		}
		if slots.ToEpoch(st.Slot()) < validator.ActivationEpoch+params.BeaconConfig().ShardCommitteePeriod {
			continue
		}

		pendingBalanceToWithdraw, err := st.PendingBalanceToWithdraw(vIdx)
		if err != nil {
			return nil, err
		}
		if isFullExitRequest {
			if pendingBalanceToWithdraw == 0 {
				maxExitEpoch, churn := validators.MaxExitEpochAndChurn(st)
				var err error
				st, _, err = validators.InitiateValidatorExit(ctx, st, vIdx, maxExitEpoch, churn)
				if err != nil {
					return nil, err
				}
			}
			continue
		}

		hasSufficientEffectiveBalance := validator.EffectiveBalance >= params.BeaconConfig().MinActivationBalance
		vBal, err := st.BalanceAtIndex(vIdx)
		if err != nil {
			return nil, err
		}
		hasExcessBalance := vBal > params.BeaconConfig().MinActivationBalance+pendingBalanceToWithdraw

		// Only allow partial withdrawals with compounding withdrawal credentials
		if helpers.HasCompoundingWithdrawalCredential(validator) && hasSufficientEffectiveBalance && hasExcessBalance {
			toWithdraw := min(vBal-params.BeaconConfig().MinActivationBalance-pendingBalanceToWithdraw, amount)
			exitQueueEpoch, err := st.ExitEpochAndUpdateChurn(math.Gwei(toWithdraw))
			if err != nil {
				return nil, err
			}
			withdrawableEpoch := exitQueueEpoch + params.BeaconConfig().MinValidatorWithdrawabilityDelay
			if err := st.AppendPendingPartialWithdrawal(&ethpb.PendingPartialWithdrawal{
				Index:             vIdx,
				Amount:            toWithdraw,
				WithdrawableEpoch: withdrawableEpoch,
			}); err != nil {
				return nil, err
			}
		}
	}

	return st, nil
}

// ProcessDepositReceipts is a function as part of electra to process execution layer deposits
func ProcessDepositReceipts(beaconState state.BeaconState, receipts []*enginev1.DepositReceipt) (state.BeaconState, error) {
	return b.ProcessDepositReceipts(beaconState, receipts)
}
