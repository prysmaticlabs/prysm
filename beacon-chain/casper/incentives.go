package casper

import (
	"math"

	"github.com/prysmaticlabs/prysm/beacon-chain/params"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "casper")

// CalculateRewards adjusts validators balances by applying rewards or penalties
// based on FFG incentive structure.
func CalculateRewards(
	attestations []*pb.AggregatedAttestation,
	validators []*pb.ValidatorRecord,
	dynasty uint64,
	totalDeposit uint64,
	totalParticipatedDeposit uint64) ([]*pb.ValidatorRecord, error) {
	activeValidators := ActiveValidatorIndices(validators, dynasty)
	attesterBitfield := attestations[len(attestations)-1].AttesterBitfield
	attesterFactor := totalParticipatedDeposit * 3
	totalFactor := totalDeposit * 2

	if attesterFactor >= totalFactor {
		log.Debug("Applying rewards and penalties for the validators from last cycle")
		for i, attesterIndex := range activeValidators {
			voted := shared.CheckBit(attesterBitfield, int(attesterIndex))
			if voted {
				validators[i].Balance += params.AttesterReward
			} else {
				validators[i].Balance -= params.AttesterReward
			}
		}
	}
	return validators, nil
}

func CalculateReward(attestations []*pb.AggregatedAttestation, validators []*pb.ValidatorRecord, dynasty uint64, totalDeposit uint64) {

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
