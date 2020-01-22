package helpers

import (
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
)

// TotalBalance returns the total amount at stake in Gwei
// of input validators.
//
// Spec pseudocode definition:
//   def get_total_balance(state: BeaconState, indices: Set[ValidatorIndex]) -> Gwei:
//    """
//    Return the combined effective balance of the ``indices``. (1 Gwei minimum to avoid divisions by zero.)
//    """
//    return Gwei(max(1, sum([state.validators[index].effective_balance for index in indices])))
func TotalBalance(state *stateTrie.BeaconState, indices []uint64) uint64 {
	vals := state.Validators()
	total := uint64(0)
	for _, idx := range indices {
		total += vals[idx].EffectiveBalance
	}

	// Return 1 Gwei minimum to avoid divisions by zero
	if total == 0 {
		return 1
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
	vals := state.Validators()
	total := uint64(0)
	for i, v := range vals {
		if IsActiveValidator(v, SlotToEpoch(state.Slot())) {
			total += vals[i].EffectiveBalance
		}
	}
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
	return state.UpdateBalancesAtIndex(balAtIdx+delta, idx)
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
		return state.UpdateBalancesAtIndex(0, idx)
	}
	return state.UpdateBalancesAtIndex(balAtIdx-delta, idx)
}
