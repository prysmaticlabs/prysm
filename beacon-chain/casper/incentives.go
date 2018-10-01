package casper

import (
	"github.com/prysmaticlabs/prysm/beacon-chain/params"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "casper")

// CalculateRewards adjusts validators balances by applying rewards or penalties
// based on FFG incentive structure.
// FFG Rewards scheme rewards validator who have voted on blocks, and penalises those validators
// who are offline. The penalties are more severe the longer they are offline.
func CalculateRewards(
	slot uint64,
	voterIndices []uint32,
	validators []*pb.ValidatorRecord,
	dynasty uint64,
	totalParticipatedDeposit uint64,
	timeSinceFinality uint64) []*pb.ValidatorRecord {
	totalDeposit := TotalActiveValidatorDeposit(dynasty, validators)
	activeValidators := ActiveValidatorIndices(validators, dynasty)
	rewardQuotient := uint64(RewardQuotient(dynasty, validators))
	penaltyQuotient := uint64(quadraticPenaltyQuotient())

	log.Debugf("Applying rewards and penalties for the validators for slot %d", slot)
	if timeSinceFinality <= 3*params.GetConfig().CycleLength {
		for _, validatorIndex := range activeValidators {
			var voted bool

			for _, voterIndex := range voterIndices {
				if voterIndex == validatorIndex {
					voted = true
					balance := validators[validatorIndex].GetBalance()
					newbalance := int64(balance) + int64(balance/rewardQuotient)*(2*int64(totalParticipatedDeposit)-int64(totalDeposit))/int64(totalDeposit)
					validators[validatorIndex].Balance = uint64(newbalance)
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
		for _, validatorIndex := range activeValidators {
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
func RewardQuotient(dynasty uint64, validators []*pb.ValidatorRecord) uint64 {
	totalDepositETH := TotalActiveValidatorDepositInEth(dynasty, validators)
	return params.GetConfig().BaseRewardQuotient * shared.IntegerSquareRoot(totalDepositETH)
}

// SlotMaxInterestRate returns the interest rate for a validator in a slot, the interest
// rate is targeted for a compunded annual rate of 3.88%.
func SlotMaxInterestRate(dynasty uint64, validators []*pb.ValidatorRecord) float64 {
	rewardQuotient := float64(RewardQuotient(dynasty, validators))
	return 1 / rewardQuotient
}

// quadraticPenaltyQuotient is the quotient that will be used to apply penalties to offline
// validators.
func quadraticPenaltyQuotient() uint64 {
	dropTimeFactor := params.GetConfig().SqrtDropTime / params.GetConfig().SlotDuration
	return dropTimeFactor * dropTimeFactor
}

// QuadraticPenalty returns the penalty that will be applied to an offline validator
// based on the number of slots that they are offline.
func QuadraticPenalty(numberOfSlots uint64) uint64 {
	slotFactor := (numberOfSlots * numberOfSlots) / 2
	penaltyQuotient := quadraticPenaltyQuotient()
	return slotFactor / uint64(penaltyQuotient)
}

// RewardValidatorCrosslink applies rewards to validators part of a shard committee for voting on a shard.
func RewardValidatorCrosslink(totalDeposit uint64, participatedDeposits uint64, rewardQuotient uint64, validator *pb.ValidatorRecord) {
	currentBalance := validator.Balance
	newBalance := int64(currentBalance) + int64(currentBalance/rewardQuotient)*(2*int64(participatedDeposits)-int64(totalDeposit)/int64(totalDeposit))
	validator.Balance = uint64(newBalance)
}

// PenaliseValidatorCrosslink applies penalties to validators part of a shard committee for not voting on a shard.
func PenaliseValidatorCrosslink(timeSinceLastConfirmation uint64, rewardQuotient uint64, validator *pb.ValidatorRecord) {
	newBalance := validator.Balance
	quadraticQuotient := quadraticPenaltyQuotient()
	newBalance -= newBalance/rewardQuotient + newBalance*timeSinceLastConfirmation/quadraticQuotient
	validator.Balance = newBalance
}
