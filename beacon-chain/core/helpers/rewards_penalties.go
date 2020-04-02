package helpers

import (
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// TotalBalance returns the total amount at stake in Gwei
// of input validators.
//
// Spec pseudocode definition:
//   def get_total_balance(state: BeaconState, indices: Set[ValidatorIndex]) -> Gwei:
//    """
//    Return the combined effective balance of the ``indices``.
//    ``EFFECTIVE_BALANCE_INCREMENT`` Gwei minimum to avoid divisions by zero.
//    """
//    return Gwei(max(EFFECTIVE_BALANCE_INCREMENT, sum([state.validators[index].effective_balance for index in indices])))
func TotalBalance(state *stateTrie.BeaconState, indices []uint64) uint64 {
	total := uint64(0)

	for _, idx := range indices {
		val, err := state.ValidatorAtIndexReadOnly(idx)
		if err != nil {
			continue
		}
		total += val.EffectiveBalance()
	}

	// Return EFFECTIVE_BALANCE_INCREMENT to avoid divisions by zero.
	if total == 0 {
		return params.BeaconConfig().EffectiveBalanceIncrement
	}

	return total
}

// TotalActiveBalance returns the total amount at stake in Gwei
// of active validators.
//
// Spec pseudocode definition:
//   def get_total_active_balance(state: BeaconState) -> Gwei:
//    """
//    Return the combined effective balance of the active validators.
//    """
//    return get_total_balance(state, set(get_active_validator_indices(state, get_current_epoch(state))))
func TotalActiveBalance(state *stateTrie.BeaconState) (uint64, error) {
	total := uint64(0)
	state.ReadFromEveryValidator(func(idx int, val *stateTrie.ReadOnlyValidator) error {
		if IsActiveValidatorUsingTrie(val, SlotToEpoch(state.Slot())) {
			total += val.EffectiveBalance()
		}
		return nil
	})
	return total, nil
}

// IncreaseBalance increases validator with the given 'index' balance by 'delta' in Gwei.
//
// Spec pseudocode definition:
//  def increase_balance(state: BeaconState, index: ValidatorIndex, delta: Gwei) -> None:
//    """
//    Increase the validator balance at index ``index`` by ``delta``.
//    """
//    state.balances[index] += delta
func IncreaseBalance(state *stateTrie.BeaconState, idx uint64, delta uint64) error {
	balAtIdx, err := state.BalanceAtIndex(idx)
	if err != nil {
		return err
	}
	return state.UpdateBalancesAtIndex(idx, balAtIdx+delta)
}

// DecreaseBalance decreases validator with the given 'index' balance by 'delta' in Gwei.
//
// Spec pseudocode definition:
//  def decrease_balance(state: BeaconState, index: ValidatorIndex, delta: Gwei) -> None:
//    """
//    Decrease the validator balance at index ``index`` by ``delta``, with underflow protection.
//    """
//    state.balances[index] = 0 if delta > state.balances[index] else state.balances[index] - delta
func DecreaseBalance(state *stateTrie.BeaconState, idx uint64, delta uint64) error {
	balAtIdx, err := state.BalanceAtIndex(idx)
	if err != nil {
		return err
	}
	if delta > balAtIdx {
		return state.UpdateBalancesAtIndex(idx, 0)
	}
	return state.UpdateBalancesAtIndex(idx, balAtIdx-delta)
}
