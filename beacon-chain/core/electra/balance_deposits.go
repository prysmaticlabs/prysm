package electra

import (
	"context"
	"errors"
	"fmt"

	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v5/math"
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
func ProcessPendingBalanceDeposits(ctx context.Context, st state.BeaconState, activeBalance math.Gwei) error {
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
	processedAmount := math.Gwei(0)
	nextDepositIndex := 0

	deposits, err := st.PendingBalanceDeposits()
	if err != nil {
		return err
	}

	for _, deposit := range deposits {
		if processedAmount+math.Gwei(deposit.Amount) > availableForProcessing {
			break
		}
		if err := helpers.IncreaseBalance(st, deposit.Index, deposit.Amount); err != nil {
			return err
		}
		processedAmount += math.Gwei(deposit.Amount)
		nextDepositIndex++
	}

	deposits = deposits[nextDepositIndex:]
	if err := st.SetPendingBalanceDeposits(deposits); err != nil {
		return err
	}

	if len(deposits) == 0 {
		return st.SetDepositBalanceToConsume(0)
	} else {
		dbtc, err := math.Sub64(uint64(availableForProcessing), uint64(processedAmount))
		if err != nil {
			return fmt.Errorf("failed to compute new deposit balance to consume: %w", err)
		}
		return st.SetDepositBalanceToConsume(math.Gwei(dbtc))
	}
}
