package casper

import (
	"math"

	"github.com/prysmaticlabs/prysm/beacon-chain/params"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
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
	timeSinceFinality uint64) ([]*pb.ValidatorRecord, error) {
	totalDeposit := TotalActiveValidatorDeposit(dynasty, validators)
	activeValidators := ActiveValidatorIndices(validators, dynasty)
	rewardQuotient := uint64(RewardQuotient(dynasty, validators))
	PenaltyQuotient := uint64(QuadraticPenaltyQuotient())
	depositFactor := (totalParticipatedDeposit - totalDeposit) / totalDeposit

	log.Debugf("Applying rewards and penalties for the validators for slot %d", slot)
	if timeSinceFinality <= (params.CycleLength) {
		for _, validatorIndice := range activeValidators {
			var voted bool

			for _, voterIndice := range voterIndices {
				if voterIndice == validatorIndice {
					voted = true
					balance := validators[validatorIndice].GetBalance()
					newbalance := uint64(balance + (balance/rewardQuotient)*depositFactor)
					validators[voterIndice].Balance = newbalance
					break
				}
			}

			if !voted {
				newbalance := validators[validatorIndice].GetBalance()
				newbalance -= newbalance / rewardQuotient
				validators[validatorIndice].Balance = newbalance
			}
		}

	} else {
		for _, validatorIndice := range activeValidators {
			var voted bool

			for _, voterIndice := range voterIndices {
				if voterIndice == validatorIndice {
					voted = true
					break
				}
			}

			if !voted {
				newbalance := validators[validatorIndice].GetBalance()
				newbalance -= newbalance/rewardQuotient + newbalance*timeSinceFinality/PenaltyQuotient
				validators[validatorIndice].Balance = newbalance
			}
		}

	}

	return validators, nil
}

// RewardQuotient returns the reward quotient for validators which will be used to
// reward validators for voting on blocks, or penalise them for being offline.
func RewardQuotient(dynasty uint64, validators []*pb.ValidatorRecord) uint64 {
	totalDepositETH := TotalActiveValidatorDepositInEth(dynasty, validators)
	return params.BaseRewardQuotient * uint64(math.Pow(float64(totalDepositETH), 0.5))
}

// SlotMaxInterestRate returns the interest rate for a validator in a slot, the interest
// rate is targeted for a compunded annual rate of 3.88%.
func SlotMaxInterestRate(dynasty uint64, validators []*pb.ValidatorRecord) float64 {
	rewardQuotient := float64(RewardQuotient(dynasty, validators))
	return 1 / rewardQuotient
}

// QuadraticPenaltyQuotient is the quotient that will be used to apply penalties to offline
// validators.
func QuadraticPenaltyQuotient() uint64 {
	dropTimeFactor := float64(params.SqrtDropTime / params.SlotDuration)
	return uint64(math.Pow(dropTimeFactor, 0.5))
}

// QuadraticPenalty returns the penalty that will be applied to an offline validator
// based on the number of slots that they are offline.
func QuadraticPenalty(numberOfSlots uint64) uint64 {
	slotFactor := (numberOfSlots * numberOfSlots) / 2
	penaltyQuotient := QuadraticPenaltyQuotient()
	return slotFactor / uint64(penaltyQuotient)
}
