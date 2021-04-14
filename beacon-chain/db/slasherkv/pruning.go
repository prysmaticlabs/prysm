package slasherkv

import (
	"bytes"
	"context"
	"encoding/binary"

	fssz "github.com/ferranbt/fastssz"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	log "github.com/sirupsen/logrus"
	bolt "go.etcd.io/bbolt"
)

// PruneProposals prunes all proposal data older than historyLength.
func (s *Store) PruneProposals(ctx context.Context, currentEpoch types.Epoch, historyLength types.Epoch) error {
	if currentEpoch < historyLength {
		return nil
	}
	// + 1 here so we can prune everything less than this, but not equal.
	endEpoch := currentEpoch - historyLength
	endPruneSlot, err := helpers.StartSlot(endEpoch)
	if err != nil {
		return err
	}
	encodedEndSlot := fssz.MarshalUint64([]byte{}, uint64(endPruneSlot))

	// We retrieve the lowest stored epoch.
	var lowestEpoch types.Epoch
	if err = s.db.View(func(tx *bolt.Tx) error {
		proposalBkt := tx.Bucket(proposalRecordsBucket)
		c := proposalBkt.Cursor()
		k, _ := c.First()
		encSlot := k[:8]
		lowestEpoch = helpers.SlotToEpoch(types.Slot(binary.LittleEndian.Uint64(encSlot)))
		return nil
	}); err != nil {
		return err
	}

	// We prune in increments of 100 epochs at a time to prevent
	// a long-running bolt transaction which overwhelms CPU and memory.
	pruningIncrements := types.Epoch(100)
	epochCursor := lowestEpoch
	var slotCursor types.Slot
	var finished bool
	var encodedSlotCursor []byte
	for !finished {
		log.Infof("Pruned %d/%d epochs worth of proposals", epochCursor-lowestEpoch, endEpoch-lowestEpoch)
		encodedSlotCursor = fssz.MarshalUint64([]byte{}, uint64(slotCursor))
		if err = s.db.Update(func(tx *bolt.Tx) error {
			proposalBkt := tx.Bucket(proposalRecordsBucket)
			c := proposalBkt.Cursor()
			epochsPruned := types.Epoch(0)
			var lastPrunedEpoch types.Epoch
			for k, _ := c.Seek(encodedSlotCursor); k != nil; k, _ = c.Next() {
				if !slotPrefixLessThan(k, encodedEndSlot) {
					finished = true
					return nil
				}
				slot := types.Slot(binary.LittleEndian.Uint64(k[:8]))
				epoch := helpers.SlotToEpoch(slot)
				if epochsPruned == pruningIncrements {
					epochCursor = epoch
					return nil
				}
				if epoch != lastPrunedEpoch {
					epochsPruned++
					lastPrunedEpoch = epoch
				}
				slasherProposalsPrunedTotal.Inc()
				if err := proposalBkt.Delete(k); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			return err
		}
	}
}

// PruneAttestations prunes all proposal data older than historyLength.
func (s *Store) PruneAttestations(ctx context.Context, currentEpoch types.Epoch, historyLength types.Epoch) error {
	if currentEpoch < historyLength {
		return nil
	}
	// + 1 here so we can prune everything less than this, but not equal.
	endPruneEpoch := currentEpoch - historyLength
	epochEnc := encodeTargetEpoch(endPruneEpoch, historyLength)
	return s.db.Update(func(tx *bolt.Tx) error {
		signingRootsBkt := tx.Bucket(attestationDataRootsBucket)
		attRecordsBkt := tx.Bucket(attestationRecordsBucket)
		c := signingRootsBkt.Cursor()
		for k, v := c.Seek(epochEnc); k != nil; k, v = c.Prev() {
			if !epochPrefixLessThan(k, epochEnc) {
				continue
			}
			slasherAttestationsPrunedTotal.Inc()
			if err := signingRootsBkt.Delete(k); err != nil {
				return err
			}
			if err := attRecordsBkt.Delete(v); err != nil {
				return err
			}
		}
		return nil
	})
}

func slotPrefixLessThan(key, lessThan []byte) bool {
	encSlot := key[:8]
	return bytes.Compare(encSlot, lessThan) < 0
}

func epochPrefixLessThan(key, lessThan []byte) bool {
	encSlot := key[:2]
	return bytes.Compare(encSlot, lessThan) < 0
}
