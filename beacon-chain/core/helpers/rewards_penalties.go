package helpers

import (
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// EffectiveBalance returns the balance at stake for the validator.
// Beacon chain allows validators to top off their balance above MAX_DEPOSIT,
// but they can be slashed at most MAX_DEPOSIT at any time.
//
// Spec pseudocode definition:
//   def get_effective_balance(state: State, index: int) -> int:
//     """
//     Returns the effective balance (also known as "balance at stake") for a ``validator`` with the given ``index``.
//     """
//     return min(state.validator_balances[idx], MAX_DEPOSIT)
func EffectiveBalance(state *pb.BeaconState, idx uint64) uint64 {
	if state.ValidatorBalances[idx] > params.BeaconConfig().MaxDepositAmount {
		return params.BeaconConfig().MaxDepositAmount
	}
	return state.ValidatorBalances[idx]
}

// TotalBalance returns the total deposited amount at stake in Gwei
// of all active validators.
//
// Spec pseudocode definition:
//   def get_total_balance(state: BeaconState, validators: List[ValidatorIndex]) -> Gwei:
//    """
//    Return the combined effective balance of an array of validators.
//    """
//    return sum([get_effective_balance(state, i) for i in validators])
func TotalBalance(state *pb.BeaconState, validators []uint64) uint64 {
	var totalBalance uint64

	for _, idx := range validators {
		totalBalance += EffectiveBalance(state, idx)
	}
	return totalBalance
}

// BaseRewardQuotient takes the previous total balance and calculates for
// the quotient of the base reward.
//
// Spec pseudocode definition:
//    base_reward_quotient =
//      integer_squareroot(previous_total_balance) // BASE_REWARD_QUOTIENT
func BaseRewardQuotient(prevTotalBalance uint64) uint64 {
	return mathutil.IntegerSquareRoot(prevTotalBalance) / params.BeaconConfig().BaseRewardQuotient
}

// BaseReward takes state and validator index to calculate for
// individual validator's base reward.
//
// Spec pseudocode definition:
//    base_reward(state, index) =
//    	get_effective_balance(state, index) // base_reward_quotient // 5
func BaseReward(
	state *pb.BeaconState,
	validatorIndex uint64,
	baseRewardQuotient uint64) uint64 {

	validatorBalance := EffectiveBalance(state, validatorIndex)
	return validatorBalance / baseRewardQuotient / 5
}

// InactivityPenalty takes state and validator index to calculate for
// individual validator's penalty for being offline.
//
// Spec pseudocode definition:
//    inactivity_penalty(state, index, epochs_since_finality) =
//    	base_reward(state, index) + get_effective_balance(state, index)
//    	* epochs_since_finality // INACTIVITY_PENALTY_QUOTIENT // 2
func InactivityPenalty(
	state *pb.BeaconState,
	validatorIndex uint64,
	baseRewardQuotient uint64,
	epochsSinceFinality uint64) uint64 {

	baseReward := BaseReward(state, validatorIndex, baseRewardQuotient)
	validatorBalance := EffectiveBalance(state, validatorIndex)
	return baseReward + validatorBalance*epochsSinceFinality/params.BeaconConfig().InactivityPenaltyQuotient/2
}
