package electra

import (
	"context"
	"errors"

	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	enginev1 "github.com/prysmaticlabs/prysm/v5/proto/engine/v1"
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
	// TODO: replace with 6110 logic
	// return b.ProcessDepositRequests(beaconState, requests)
	return beaconState, nil
}
