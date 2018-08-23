package casper

import (
	"fmt"

	"github.com/prysmaticlabs/prysm/beacon-chain/params"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "casper")

// CalculateRewards adjusts validators balances by applying rewards or penalties
// based on FFG incentive structure.
func CalculateRewards(active *types.ActiveState, crystallized *types.CrystallizedState, block *types.Block) error {
	latestPendingAtt := active.LatestPendingAttestation()
	if latestPendingAtt == nil {
		return nil
	}
	validators := crystallized.Validators()
	activeValidators := ActiveValidatorIndices(crystallized)
	attesterDeposits := GetAttestersTotalDeposit(active)
	totalDeposit := crystallized.TotalDeposits()

	attesterFactor := attesterDeposits * 3
	totalFactor := uint64(totalDeposit * 2)
	if attesterFactor >= totalFactor {
		log.Debugf("Setting justified slot to current slot number: %v", block.SlotNumber())
		crystallized.UpdateJustifiedSlot(block.SlotNumber())

		log.Debug("Applying rewards and penalties for the validators from last cycle")
		for i, attesterIndex := range activeValidators {
			voted, err := utils.CheckBit(latestPendingAtt.AttesterBitfield, attesterIndex)
			if err != nil {
				return fmt.Errorf("exiting calculate rewards FFG due to %v", err)
			}
			if voted {
				validators[i].Balance += params.AttesterReward
			} else {
				validators[i].Balance -= params.AttesterReward
			}
		}

		log.Debug("Resetting attester bit field to all zeros")
		active.ClearPendingAttestations()

		crystallized.SetValidators(validators)
	}
	return nil
}
