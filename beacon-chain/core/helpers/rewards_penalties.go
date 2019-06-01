package helpers

import (
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

var totalBalanceCache = make(map[uint64]uint64)
var totalActiveBalanceCache = make(map[uint64]uint64)

// TotalBalance returns the total amount at stake in Gwei
// of input validators.
//
// Spec pseudocode definition:
//   def get_total_balance(state: BeaconState, indices: List[ValidatorIndex]) -> Gwei:
//    """
//    Return the combined effective balance of an array of ``validators``.
//    """
//    return sum([state.validator_registry[index].effective_balance for index in indices])
func TotalBalance(state *pb.BeaconState, indices []uint64) uint64 {
	epoch := CurrentEpoch(state)
	if _, ok := totalBalanceCache[epoch]; ok {
		return totalBalanceCache[epoch]
	}

	var total uint64
	for _, idx := range indices {
		total += state.ValidatorRegistry[idx].EffectiveBalance
	}

	totalBalanceCache[epoch] = total

	return total
}

// TotalActiveBalance returns the total amount at stake in Gwei
// of active validators.
func TotalActiveBalance(state *pb.BeaconState) uint64 {
	epoch := CurrentEpoch(state)
	if _, ok := totalActiveBalanceCache[epoch]; ok {
		return totalActiveBalanceCache[epoch]
	}

	var total uint64
	for i, v := range state.ValidatorRegistry {
		if IsActiveValidator(v, epoch) {
			total += state.ValidatorRegistry[i].EffectiveBalance
		}
	}

	totalActiveBalanceCache[epoch] = total

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

// RestartTotalBalanceCache restarts the total validator balance cache from scratch.
func RestartTotalBalanceCache() {
	totalBalanceCache = make(map[uint64]uint64)
}

// RestartTotalActiveBalanceCache restarts the total active validator balance cache from scratch.
func RestartTotalActiveBalanceCache() {
	totalActiveBalanceCache = make(map[uint64]uint64)
}
