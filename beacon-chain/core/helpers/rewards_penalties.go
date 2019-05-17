package helpers

import (
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// TotalBalance returns the total amount at stake in Gwei
// of all active validators.
//
// Spec pseudocode definition:
//   def get_total_balance(state: BeaconState, indices: List[ValidatorIndex]) -> Gwei:
//    """
//    Return the combined effective balance of an array of ``validators``.
//    """
//    return sum([state.validator_registry[index].effective_balance for index in indices])
func TotalBalance(state *pb.BeaconState, indices []uint64) uint64 {
	var total uint64

	for _, idx := range indices {
		total += state.ValidatorRegistry[idx].EffectiveBalance
	}
	return total
}

// IncreaseBalance increases validator with the given 'index' balance by 'delta' in Gwei.
//
// Spec pseudocode definition:
// def increase_balance(state: BeaconState, index: ValidatorIndex, delta: Gwei) -> None:
//    """
//    Increase validator balance by ``delta``.
//    """
//    state.balances[index] += delta
func IncreaseBalance(state *pb.BeaconState, idx uint64, delta uint64) *pb.BeaconState {
	state.Balances[idx] += delta
	return state
}

// DecreaseBalance decreases validator with the given 'index' balance by 'delta' in Gwei.
//
// def decrease_balance(state: BeaconState, index: ValidatorIndex, delta: Gwei) -> None:
//    """
//    Decrease validator balance by ``delta`` with underflow protection.
//    """
//    state.balances[index] = 0 if delta > state.balances[index] else state.balances[index] - delta
func DecreaseBalance(state *pb.BeaconState, idx uint64, delta uint64) *pb.BeaconState {
	if delta > state.Balances[idx] {
		state.Balances[idx] = 0
		return state
	}
	state.Balances[idx] -= delta
	return state
}
