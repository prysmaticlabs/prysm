package blockchain

import (
	"encoding/hex"
	"fmt"
	"time"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/timeutils"
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

func logBlockSyncStatus(block *ethpb.BeaconBlock, blockRoot [32]byte, finalized *ethpb.Checkpoint, receivedTime time.Time, genesisTime uint64) error {
	startTime, err := helpers.SlotToTime(genesisTime, block.Slot)
	if err != nil {
		return err
	}
	log.WithFields(logrus.Fields{
		"slot":                      block.Slot,
		"slotInEpoch":               block.Slot % params.BeaconConfig().SlotsPerEpoch,
		"block":                     fmt.Sprintf("0x%s...", hex.EncodeToString(blockRoot[:])[:8]),
		"epoch":                     helpers.SlotToEpoch(block.Slot),
		"finalizedEpoch":            finalized.Epoch,
		"finalizedRoot":             fmt.Sprintf("0x%s...", hex.EncodeToString(finalized.Root)[:8]),
		"sinceSlotStartTime":        timeutils.Now().Sub(startTime),
		"chainServiceProcessedTime": timeutils.Now().Sub(receivedTime),
	}).Info("Synced new block")
	return nil
}
