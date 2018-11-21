package incentives

import (
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// CalculateRewards adjusts validators balances by applying rewards or penalties
// based on FFG incentive structure.
// FFG Rewards scheme rewards validator who have voted on blocks, and penalises those validators
// who are offline. The penalties are more severe the longer they are offline.
func CalculateRewards(
	voterIndices []uint32,
	activeValidatorIndices []uint32,
	validators []*pb.ValidatorRecord,
	totalActiveValidatorDeposit uint64,
	totalParticipatedDeposit uint64,
	timeSinceFinality uint64,
) []*pb.ValidatorRecord {

	// Calculate the reward and penalty quotients for the validator set.
	rewardQuotient := RewardQuotient(totalActiveValidatorDeposit)
	penaltyQuotient := QuadraticPenaltyQuotient()

	if timeSinceFinality <= 3*params.BeaconConfig().CycleLength {
		for _, validatorIndex := range activeValidatorIndices {
			var voted bool

			for _, voterIndex := range voterIndices {
				if voterIndex == validatorIndex {
					voted = true
					balance := validators[validatorIndex].GetBalance()
					newBalance := calculateBalance(balance, rewardQuotient, totalParticipatedDeposit, totalActiveValidatorDeposit)
					validators[validatorIndex].Balance = uint64(newBalance)
					break
				}
			}

			if !voted {
				newBalance := validators[validatorIndex].GetBalance()
				newBalance -= newBalance / rewardQuotient
				validators[validatorIndex].Balance = newBalance
			}
		}

	} else {
		for _, validatorIndex := range activeValidatorIndices {
			var voted bool

			for _, voterIndex := range voterIndices {
				if voterIndex == validatorIndex {
					voted = true
					break
				}
			}

			if !voted {
				newBalance := validators[validatorIndex].GetBalance()
				newBalance -= newBalance/rewardQuotient + newBalance*timeSinceFinality/penaltyQuotient
				validators[validatorIndex].Balance = newBalance
			}
		}

	}

	return validators
}

// RewardQuotient returns the reward quotient for validators which will be used to
// reward validators for voting on blocks, or penalise them for being offline.
func RewardQuotient(totalActiveValidatorDeposit uint64) uint64 {
	totalDepositETH := totalActiveValidatorDeposit / params.BeaconConfig().Gwei
	return params.BeaconConfig().BaseRewardQuotient * mathutil.IntegerSquareRoot(totalDepositETH)
}

// QuadraticPenaltyQuotient is the quotient that will be used to apply penalties to offline
// validators.
func QuadraticPenaltyQuotient() uint64 {
	dropTimeFactor := params.BeaconConfig().SqrtExpDropTime
	return dropTimeFactor * dropTimeFactor
}

// QuadraticPenalty returns the penalty that will be applied to an offline validator
// based on the number of slots that they are offline.
func QuadraticPenalty(numberOfSlots uint64) uint64 {
	slotFactor := (numberOfSlots * numberOfSlots) / 2
	penaltyQuotient := QuadraticPenaltyQuotient()
	return slotFactor / penaltyQuotient
}

// RewardValidatorCrosslink applies rewards to validators part of a shard committee for voting on a shard.
// TODO(#538): Change this to big.Int as tests using 64 bit integers fail due to integer overflow.
func RewardValidatorCrosslink(totalDeposit uint64, participatedDeposits uint64, rewardQuotient uint64, validator *pb.ValidatorRecord) {
	currentBalance := int64(validator.Balance)
	currentBalance += int64(currentBalance) / int64(rewardQuotient) * (2*int64(participatedDeposits) - int64(totalDeposit)) / int64(totalDeposit)
	validator.Balance = uint64(currentBalance)
}

// PenaliseValidatorCrosslink applies penalties to validators part of a shard committee for not voting on a shard.
func PenaliseValidatorCrosslink(timeSinceLastConfirmation uint64, rewardQuotient uint64, validator *pb.ValidatorRecord) {
	newBalance := validator.Balance
	quadraticQuotient := QuadraticPenaltyQuotient()
	newBalance -= newBalance/rewardQuotient + newBalance*timeSinceLastConfirmation/quadraticQuotient
	validator.Balance = newBalance
}

// calculateBalance applies the Casper FFG reward calculation based on reward quotients
// and total deposits from validators.
func calculateBalance(
	balance uint64,
	rewardQuotient uint64,
	totalParticipatedDeposit uint64,
	totalActiveValidatorDeposit uint64,
) uint64 {
	participationNumerator := 2*int64(totalParticipatedDeposit) - int64(totalActiveValidatorDeposit)
	return uint64(int64(balance) + int64(balance/rewardQuotient)*participationNumerator/int64(totalActiveValidatorDeposit))
}
