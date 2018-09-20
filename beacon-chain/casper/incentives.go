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
func CalculateRewards(
	slot uint64,
	voterIndices []uint32,
	validators []*pb.ValidatorRecord,
	dynasty uint64,
	totalParticipatedDeposit uint64,
	timeSinceFinality uint64) ([]*pb.ValidatorRecord, error) {
	totalDeposit := TotalActiveValidatorDeposit(dynasty, validators)
	activeValidators := ActiveValidatorIndices(validators, dynasty)
	rewardQuotient := RewardQuotient(dynasty, validators)
	PenaltyQuotient := QuadraticPenaltyQuotient()
	depositFactor := int64(2*totalParticipatedDeposit-totalDeposit) / int64(totalDeposit)

	log.Debugf("Applying rewards and penalties for the validators for slot %d", slot)
	if timeSinceFinality <= (params.CycleLength) {
		for _, validatorIndice := range activeValidators {
			var voted bool

			for _, voterIndice := range voterIndices {
				if voterIndice == validatorIndice {
					voted = true
					balance := validators[voterIndice].GetBalance()
					newbalance := int64(balance) + int64(balance/rewardQuotient)*depositFactor
					validators[voterIndice].Balance = uint64(newbalance)
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

func RewardQuotient(dynasty uint64, validators []*pb.ValidatorRecord) uint64 {
	totalDepositETH := TotalActiveValidatorDepositInEth(dynasty, validators)
	return params.BaseRewardQuotient * uint64(math.Pow(float64(totalDepositETH), 0.5))
}

func SlotMaxInterestRate(dynasty uint64, validators []*pb.ValidatorRecord) float64 {
	rewardQuotient := float64(RewardQuotient(dynasty, validators))
	return 1 / rewardQuotient
}

func QuadraticPenaltyQuotient() uint64 {
	dropTimeFactor := float64(params.SqrtDropTime / params.SlotDuration)
	return uint64(math.Pow(dropTimeFactor, 0.5))
}

func QuadraticPenalty(numberOfSlots uint64) uint64 {
	slotFactor := (numberOfSlots * numberOfSlots) / 2
	penaltyQuotient := QuadraticPenaltyQuotient()
	return slotFactor / penaltyQuotient
}
