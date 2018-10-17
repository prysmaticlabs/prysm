package casper

import (
	"github.com/prysmaticlabs/prysm/beacon-chain/params"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
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
	totalParticipatedDeposit uint64,
	timeSinceFinality uint64) (newValidators []*pb.ValidatorRecord) {
	newValidators = validators
	totalDeposit := TotalActiveValidatorDeposit(validators)
	activeValidators := ActiveValidatorIndices(validators)
	rewardQuotient := uint64(RewardQuotient(validators))
	penaltyQuotient := uint64(quadraticPenaltyQuotient())

	log.Debugf("Applying rewards and penalties for the validators for slot %d", slot)
	if timeSinceFinality <= 3*params.GetConfig().CycleLength {
		for _, validatorIndex := range activeValidators {
			var voted bool

			for _, voterIndex := range voterIndices {
				if voterIndex == validatorIndex {
					voted = true
					balance := newValidators[validatorIndex].GetBalance()
					newBalance := int64(balance) + int64(balance/rewardQuotient)*(2*int64(totalParticipatedDeposit)-int64(totalDeposit))/int64(totalDeposit)
					newValidators[validatorIndex].Balance = uint64(newBalance)
					break
				}
			}

			if !voted {
				newBalance := newValidators[validatorIndex].GetBalance()
				newBalance -= newBalance / rewardQuotient
				newValidators[validatorIndex].Balance = newBalance
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
				newBalance := newValidators[validatorIndex].GetBalance()
				newBalance -= newBalance/rewardQuotient + newBalance*timeSinceFinality/penaltyQuotient
				newValidators[validatorIndex].Balance = newBalance
			}
		}
	}

	return newValidators
}

// RewardQuotient returns the reward quotient for validators which will be used to
// reward validators for voting on blocks, or penalise them for being offline.
func RewardQuotient(validators []*pb.ValidatorRecord) uint64 {
	totalDepositETH := TotalActiveValidatorDepositInEth(validators)
	return params.GetConfig().BaseRewardQuotient * mathutil.IntegerSquareRoot(totalDepositETH)
}

// SlotMaxInterestRate returns the interest rate for a validator in a slot, the interest
// rate is targeted for a compunded annual rate of 3.88%.
func SlotMaxInterestRate(validators []*pb.ValidatorRecord) float64 {
	rewardQuotient := float64(RewardQuotient(validators))
	return 1 / rewardQuotient
}

// quadraticPenaltyQuotient is the quotient that will be used to apply penalties to offline
// validators.
func quadraticPenaltyQuotient() uint64 {
	dropTimeFactor := params.GetConfig().SqrtExpDropTime / params.GetConfig().SlotDuration
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
// TODO(#538): Change this to big.Int as tests using 64 bit integers fail due to integer overflow.
func RewardValidatorCrosslink(totalDeposit uint64, participatedDeposits uint64, rewardQuotient uint64, validator *pb.ValidatorRecord) (newValidator *pb.ValidatorRecord) {
	newValidator = validator
	currentBalance := newValidator.Balance
	newValidator.Balance += uint64(int64(currentBalance) / int64(rewardQuotient) * (2*int64(participatedDeposits) - int64(totalDeposit)) / int64(totalDeposit))
	return newValidator
}

// PenaliseValidatorCrosslink applies penalties to validators part of a shard committee for not voting on a shard.
func PenaliseValidatorCrosslink(timeSinceLastConfirmation uint64, rewardQuotient uint64, validator *pb.ValidatorRecord) (newValidator *pb.ValidatorRecord) {
	newValidator = validator
	currentBalance := newValidator.Balance
	quadraticQuotient := quadraticPenaltyQuotient()
	newValidator.Balance -= currentBalance/rewardQuotient + currentBalance*timeSinceLastConfirmation/quadraticQuotient
	return newValidator
}
