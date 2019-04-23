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
//   def get_effective_balance(state: BeaconState, index: ValidatorIndex) -> Gwei:
//    """
//    Return the effective balance (also known as "balance at stake") for a validator with the given ``index``.
//    """
//    return min(get_balance(state, index), MAX_DEPOSIT_AMOUNT)
func EffectiveBalance(state *pb.BeaconState, idx uint64) uint64 {
	if state.Balances[idx] > params.BeaconConfig().MaxDepositAmount {
		return params.BeaconConfig().MaxDepositAmount
	}
	return state.Balances[idx]
}

// TotalBalance returns the total amount at stake in Gwei
// of all active validators.
//
// Spec pseudocode definition:
//   def get_total_balance(state: BeaconState, validators: List[ValidatorIndex]) -> Gwei:
//    """
//    Return the combined effective balance of an array of ``validators``.
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

// GetBalance returns the Gwei balance of a validator by its index
//
// Spec pseudocode definition:
//   def get_balance(state: BeaconState, index: ValidatorIndex) -> Gwei:
//     """
//     Return the balance for a validator with the given ``index``.
//     """
//     return state.balances[index]
func GetBalance(state *pb.BeaconState, validatorIndex uint64) uint64 {
	return state.ValidatorBalances[validatorIndex]
}

// SetBalance sets the Gwei balance of a validator by its index
//
// Spec pseudocode definition:
// def set_balance(state: BeaconState, index: ValidatorIndex, balance: Gwei) -> None:
//     """
//     Set the balance for a validator with the given ``index`` in both ``BeaconState``
//     and validator's rounded balance ``high_balance``.
//     """
//     validator = state.validator_registry[index]
//     HALF_INCREMENT = HIGH_BALANCE_INCREMENT // 2
//     if validator.high_balance > balance or validator.high_balance + 3 * HALF_INCREMENT < balance:
//         validator.high_balance = balance - balance % HIGH_BALANCE_INCREMENT
//     state.balances[index] = balance
func SetBalance(state *pb.BeaconState, validatorIndex uint64, balance uint64) {
	validator := state.ValidatorRegistry[validatorIndex]
	halfIncrement := params.BeaconConfig().HighBalanceIncrement / 2
	if validator.HighBalance > balance || validator.HighBalance+3*halfIncrement < balance {
		validator.HighBalance = balance - balance%params.BeaconConfig().HighBalanceIncrement
	}
	state.ValidatorBalances[validatorIndex] = balance
}

// IncreaseBalance increases validator with the given 'index' balance by 'delta' in Gwei
//
// Spec pseudocode definition:
// def increase_balance(state: BeaconState, index: ValidatorIndex, delta: Gwei) -> None:
//     """
//     Increase the balance for a validator with the given ``index`` by ``delta``.
//     """
//     set_balance(state, index, get_balance(state, index) + delta)
func IncreaseBalance(state *pb.BeaconState, validatorIndex uint64, delta uint64) {
	SetBalance(state, validatorIndex, GetBalance(state, validatorIndex)+delta)
}

// DecreaseBalance decreases validator with the given 'index' balance by 'delta' in Gwei
//
// def decrease_balance(state: BeaconState, index: ValidatorIndex, delta: Gwei) -> None:
//     """
//     Decrease the balance for a validator with the given ``index`` by ``delta``.
//     Set to ``0`` when underflow.
//     """
//     current_balance = get_balance(state, index)
//     set_balance(state, index, current_balance - delta if current_balance >= delta else 0)
func DecreaseBalance(state *pb.BeaconState, validatorIndex uint64, delta uint64) {
	currentBalance := GetBalance(state, validatorIndex)
	if currentBalance >= delta {
		SetBalance(state, validatorIndex, currentBalance-delta)
	} else {
		SetBalance(state, validatorIndex, 0)
	}
}
