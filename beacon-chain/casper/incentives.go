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
func CalculateRewards(attestations []*pb.AggregatedAttestation, validators []*pb.ValidatorRecord, dynasty uint64, totalDeposit uint64) ([]*pb.ValidatorRecord, error) {
	activeValidators := ActiveValidatorIndices(validators, dynasty)
	attesterDeposits := GetAttestersTotalDeposit(activeValidators)
	attesterBitfield := attestations[len(attestations)-1].AttesterBitfield
	attesterFactor := attesterDeposits * 3
	totalFactor := uint64(totalDeposit * 2)

	// Pad 0s to the rest of attesterBitfield bitfield up to default validator length.
	for len(attesterBitfield)*8 < params.BootstrappedValidatorsCount {
		attesterBitfield = append(attesterBitfield, byte(0))
	}

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
