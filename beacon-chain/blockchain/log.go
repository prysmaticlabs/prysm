package blockchain

import (
	"encoding/hex"
	"fmt"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "blockchain")

// logs state transition related data every slot.
func logStateTransitionData(b *ethpb.BeaconBlock) {
	log.WithFields(logrus.Fields{
		"attestations":      len(b.Body.Attestations),
		"deposits":          len(b.Body.Deposits),
		"attesterSlashings": len(b.Body.AttesterSlashings),
		"proposerSlashings": len(b.Body.ProposerSlashings),
		"voluntaryExits":    len(b.Body.VoluntaryExits),
	}).Info("Finished applying state transition")
}

func logBlockSyncStatus(block *ethpb.BeaconBlock, blockRoot [32]byte, finalized *ethpb.Checkpoint) {
	log.WithFields(logrus.Fields{
		"slot":           block.Slot,
		"block":          fmt.Sprintf("0x%s...", hex.EncodeToString(blockRoot[:])[:8]),
		"epoch":          helpers.SlotToEpoch(block.Slot),
		"finalizedEpoch": finalized.Epoch,
		"finalizedRoot":  fmt.Sprintf("0x%s...", hex.EncodeToString(finalized.Root[:])[:8]),
	}).Info("Synced new block")
}

func logEpochData(beaconState *stateTrie.BeaconState) {
	log.WithFields(logrus.Fields{
		"epoch":                  helpers.CurrentEpoch(beaconState),
		"finalizedEpoch":         beaconState.FinalizedCheckpointEpoch(),
		"justifiedEpoch":         beaconState.CurrentJustifiedCheckpoint().Epoch,
		"previousJustifiedEpoch": beaconState.PreviousJustifiedCheckpoint().Epoch,
	}).Info("Starting next epoch")
	activeVals, err := helpers.ActiveValidatorIndices(beaconState, helpers.CurrentEpoch(beaconState))
	if err != nil {
		log.WithError(err).Error("Could not get active validator indices")
		return
	}
	log.WithFields(logrus.Fields{
		"totalValidators":  len(beaconState.Validators()),
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
