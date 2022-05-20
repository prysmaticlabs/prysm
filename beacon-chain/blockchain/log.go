package blockchain

import (
	"encoding/hex"
	"fmt"
	"time"

	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/runtime/version"
	prysmTime "github.com/prysmaticlabs/prysm/time"
	"github.com/prysmaticlabs/prysm/time/slots"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "blockchain")

// logs state transition related data every slot.
func logStateTransitionData(b interfaces.BeaconBlock) error {
	log := log.WithField("slot", b.Slot())
	if len(b.Body().Attestations()) > 0 {
		log = log.WithField("attestations", len(b.Body().Attestations()))
	}
	if len(b.Body().Deposits()) > 0 {
		log = log.WithField("deposits", len(b.Body().Deposits()))
	}
	if len(b.Body().AttesterSlashings()) > 0 {
		log = log.WithField("attesterSlashings", len(b.Body().AttesterSlashings()))
	}
	if len(b.Body().ProposerSlashings()) > 0 {
		log = log.WithField("proposerSlashings", len(b.Body().ProposerSlashings()))
	}
	if len(b.Body().VoluntaryExits()) > 0 {
		log = log.WithField("voluntaryExits", len(b.Body().VoluntaryExits()))
	}
	if b.Version() == version.Altair || b.Version() == version.Bellatrix {
		agg, err := b.Body().SyncAggregate()
		if err != nil {
			return err
		}
		log = log.WithField("syncBitsCount", agg.SyncCommitteeBits.Count())
	}
	if b.Version() == version.Bellatrix {
		p, err := b.Body().ExecutionPayload()
		if err != nil {
			return err
		}
		log = log.WithField("payloadHash", fmt.Sprintf("%#x", bytesutil.Trunc(p.BlockHash)))
		log = log.WithField("txCount", len(p.Transactions))
	}
	log.Info("Finished applying state transition")
	return nil
}

func logBlockSyncStatus(block interfaces.BeaconBlock, blockRoot [32]byte, justified, finalized *ethpb.Checkpoint, receivedTime time.Time, genesisTime uint64) error {
	startTime, err := slots.ToTime(genesisTime, block.Slot())
	if err != nil {
		return err
	}
	level := log.Logger.GetLevel()
	if level >= logrus.DebugLevel {
		log.WithFields(logrus.Fields{
			"slot":                      block.Slot(),
			"slotInEpoch":               block.Slot() % params.BeaconConfig().SlotsPerEpoch,
			"block":                     fmt.Sprintf("0x%s...", hex.EncodeToString(blockRoot[:])[:8]),
			"epoch":                     slots.ToEpoch(block.Slot()),
			"justifiedEpoch":            justified.Epoch,
			"justifiedRoot":             fmt.Sprintf("0x%s...", hex.EncodeToString(justified.Root)[:8]),
			"finalizedEpoch":            finalized.Epoch,
			"finalizedRoot":             fmt.Sprintf("0x%s...", hex.EncodeToString(finalized.Root)[:8]),
			"parentRoot":                fmt.Sprintf("0x%s...", hex.EncodeToString(block.ParentRoot())[:8]),
			"version":                   version.String(block.Version()),
			"sinceSlotStartTime":        prysmTime.Now().Sub(startTime),
			"chainServiceProcessedTime": prysmTime.Now().Sub(receivedTime),
		}).Debug("Synced new block")
	} else {
		log.WithFields(logrus.Fields{
			"slot":  block.Slot(),
			"block": fmt.Sprintf("0x%s...", hex.EncodeToString(blockRoot[:])[:8]),
			"epoch": slots.ToEpoch(block.Slot()),
		}).Info("Synced new block")
	}
	return nil
}
