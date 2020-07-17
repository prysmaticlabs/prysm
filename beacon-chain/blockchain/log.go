package blockchain

import (
	"encoding/hex"
	"fmt"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
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
