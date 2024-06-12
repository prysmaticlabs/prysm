package electra

import (
	"context"

	"github.com/pkg/errors"
	b "github.com/prysmaticlabs/prysm/v5/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
	enginev1 "github.com/prysmaticlabs/prysm/v5/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"go.opencensus.io/trace"
)

// ProcessPendingBalanceDeposits implements the spec definition below. This method mutates the state.
//
// Spec definition:
//
//	def process_pending_balance_deposits(state: BeaconState) -> None:
//	    available_for_processing = state.deposit_balance_to_consume + get_activation_exit_churn_limit(state)
//	    processed_amount = 0
//	    next_deposit_index = 0
//
//	    for deposit in state.pending_balance_deposits:
//	        if processed_amount + deposit.amount > available_for_processing:
//	            break
//	        increase_balance(state, deposit.index, deposit.amount)
//	        processed_amount += deposit.amount
//	        next_deposit_index += 1
//
//	    state.pending_balance_deposits = state.pending_balance_deposits[next_deposit_index:]
//
//	    if len(state.pending_balance_deposits) == 0:
//	        state.deposit_balance_to_consume = Gwei(0)
//	    else:
//	        state.deposit_balance_to_consume = available_for_processing - processed_amount
func ProcessPendingBalanceDeposits(ctx context.Context, st state.BeaconState, activeBalance primitives.Gwei) error {
	_, span := trace.StartSpan(ctx, "electra.ProcessPendingBalanceDeposits")
	defer span.End()

	if st == nil || st.IsNil() {
		return errors.New("nil state")
	}

	depBalToConsume, err := st.DepositBalanceToConsume()
	if err != nil {
		return err
	}

	availableForProcessing := depBalToConsume + helpers.ActivationExitChurnLimit(activeBalance)
	nextDepositIndex := 0

	deposits, err := st.PendingBalanceDeposits()
	if err != nil {
		return err
	}

	for _, deposit := range deposits {
		if primitives.Gwei(deposit.Amount) > availableForProcessing {
			break
		}
		if err := helpers.IncreaseBalance(st, deposit.Index, deposit.Amount); err != nil {
			return err
		}
		availableForProcessing -= primitives.Gwei(deposit.Amount)
		nextDepositIndex++
	}

	deposits = deposits[nextDepositIndex:]
	if err := st.SetPendingBalanceDeposits(deposits); err != nil {
		return err
	}

	if len(deposits) == 0 {
		return st.SetDepositBalanceToConsume(0)
	} else {
		return st.SetDepositBalanceToConsume(availableForProcessing)
	}
}

// ProcessDepositRequests is a function as part of electra to process execution layer deposits
func ProcessDepositRequests(ctx context.Context, beaconState state.BeaconState, requests []*enginev1.DepositRequest) (state.BeaconState, error) {
	_, span := trace.StartSpan(ctx, "electra.ProcessDepositRequests")
	defer span.End()
	var err error
	for _, receipt := range requests {
		beaconState, err = processDepositRequest(beaconState, receipt)
		if err != nil {
			return nil, errors.Wrap(err, "could not apply deposit receipt")
		}
	}
	return beaconState, nil
}

// processDepositRequest processes the specific deposit receipt
// def process_deposit_request(state: BeaconState, deposit_request: DepositRequest) -> None:
//
//	# Set deposit request start index
//	if state.deposit_requests_start_index == UNSET_DEPOSIT_REQUEST_START_INDEX:
//	    state.deposit_requests_start_index = deposit_request.index
//
//	apply_deposit(
//	    state=state,
//	    pubkey=deposit_request.pubkey,
//	    withdrawal_credentials=deposit_request.withdrawal_credentials,
//	    amount=deposit_request.amount,
//	    signature=deposit_request.signature,
//	)
func processDepositRequest(beaconState state.BeaconState, request *enginev1.DepositRequest) (state.BeaconState, error) {
	receiptsStartIndex, err := beaconState.DepositRequestsStartIndex()
	if err != nil {
		return nil, errors.Wrap(err, "could not get deposit requests start index")
	}
	if receiptsStartIndex == params.BeaconConfig().UnsetDepositRequestsStartIndex {
		if err := beaconState.SetDepositRequestsStartIndex(request.Index); err != nil {
			return nil, errors.Wrap(err, "could not set deposit requests start index")
		}
	}
	return b.ApplyDeposit(beaconState, &ethpb.Deposit_Data{
		PublicKey:             bytesutil.SafeCopyBytes(request.Pubkey),
		Amount:                request.Amount,
		WithdrawalCredentials: bytesutil.SafeCopyBytes(request.WithdrawalCredentials),
		Signature:             bytesutil.SafeCopyBytes(request.Signature),
	}, true) // individually verify signatures instead of batch verify
}
