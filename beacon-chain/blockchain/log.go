package blockchain

import (
	"encoding/hex"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	consensusBlocks "github.com/prysmaticlabs/prysm/v3/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/runtime/version"
	prysmTime "github.com/prysmaticlabs/prysm/v3/time"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
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
		p, err := b.Body().Execution()
		if err != nil {
			return err
		}
		log = log.WithField("payloadHash", fmt.Sprintf("%#x", bytesutil.Trunc(p.BlockHash())))
		txs, err := p.Transactions()
		switch {
		case errors.Is(err, consensusBlocks.ErrUnsupportedGetter):
		case err != nil:
			return err
		default:
			log = log.WithField("txCount", len(txs))
			txsPerSlotCount.Set(float64(len(txs)))
		}

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
		parentRoot := block.ParentRoot()
		log.WithFields(logrus.Fields{
			"slot":                      block.Slot(),
			"slotInEpoch":               block.Slot() % params.BeaconConfig().SlotsPerEpoch,
			"block":                     fmt.Sprintf("0x%s...", hex.EncodeToString(blockRoot[:])[:8]),
			"epoch":                     slots.ToEpoch(block.Slot()),
			"justifiedEpoch":            justified.Epoch,
			"justifiedRoot":             fmt.Sprintf("0x%s...", hex.EncodeToString(justified.Root)[:8]),
			"finalizedEpoch":            finalized.Epoch,
			"finalizedRoot":             fmt.Sprintf("0x%s...", hex.EncodeToString(finalized.Root)[:8]),
			"parentRoot":                fmt.Sprintf("0x%s...", hex.EncodeToString(parentRoot[:])[:8]),
			"version":                   version.String(block.Version()),
			"sinceSlotStartTime":        prysmTime.Now().Sub(startTime),
			"chainServiceProcessedTime": prysmTime.Now().Sub(receivedTime),
		}).Debug("Synced new block")
	} else {
		log.WithFields(logrus.Fields{
			"slot":           block.Slot(),
			"block":          fmt.Sprintf("0x%s...", hex.EncodeToString(blockRoot[:])[:8]),
			"finalizedEpoch": finalized.Epoch,
			"finalizedRoot":  fmt.Sprintf("0x%s...", hex.EncodeToString(finalized.Root)[:8]),
			"epoch":          slots.ToEpoch(block.Slot()),
		}).Info("Synced new block")
	}
	return nil
}

// logs payload related data every slot.
func logPayload(block interfaces.BeaconBlock) error {
	isExecutionBlk, err := blocks.IsExecutionBlock(block.Body())
	if err != nil {
		return errors.Wrap(err, "could not determine if block is execution block")
	}
	if !isExecutionBlk {
		return nil
	}
	payload, err := block.Body().Execution()
	if err != nil {
		return err
	}
	if payload.GasLimit() == 0 {
		return errors.New("gas limit should not be 0")
	}
	gasUtilized := float64(payload.GasUsed()) / float64(payload.GasLimit())

	log.WithFields(logrus.Fields{
		"blockHash":   fmt.Sprintf("%#x", bytesutil.Trunc(payload.BlockHash())),
		"parentHash":  fmt.Sprintf("%#x", bytesutil.Trunc(payload.ParentHash())),
		"blockNumber": payload.BlockNumber,
		"gasUtilized": fmt.Sprintf("%.2f", gasUtilized),
	}).Debug("Synced new payload")
	return nil
}
