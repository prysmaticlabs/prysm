package forkchoice

import (
	"fmt"

	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "forkchoice")

// logs epoch related data during epoch boundary.
func logEpochData(beaconState *stateTrie.BeaconState) {
	log.WithFields(logrus.Fields{
		"epoch":                  helpers.CurrentEpoch(beaconState),
		"finalizedEpoch":         beaconState.FinalizedCheckpoint().Epoch,
		"justifiedEpoch":         beaconState.CurrentJustifiedCheckpoint().Epoch,
		"previousJustifiedEpoch": beaconState.PreviousJustifiedCheckpoint().Epoch,
	}).Info("Starting next epoch")
	activeVals, err := helpers.ActiveValidatorIndices(beaconState, helpers.CurrentEpoch(beaconState))
	if err != nil {
		log.WithError(err).Error("Could not get active validator indices")
		return
	}
	log.WithFields(logrus.Fields{
		"totalValidators":  beaconState.NumofValidators(),
		"activeValidators": len(activeVals),
		"averageBalance":   fmt.Sprintf("%.5f ETH", averageBalance(beaconState.Balances())),
	}).Info("Validator registry information")
}

func averageBalance(balances []uint64) float64 {
	total := uint64(0)
	for i := 0; i < len(balances); i++ {
		total += balances[i]
	}
	return float64(total) / float64(len(balances)) / float64(params.BeaconConfig().GweiPerEth)
}
