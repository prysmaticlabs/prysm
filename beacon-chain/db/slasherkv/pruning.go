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

const pruningEpochIncrements = types.Epoch(100) // Prune in increments of 100 epochs worth of data.

// PruneProposals prunes all proposal data older than historyLength.
func (s *Store) PruneProposals(ctx context.Context, currentEpoch types.Epoch, historyLength types.Epoch) error {
	if currentEpoch < historyLength {
		return nil
	}
	// We can prune everything less than the current epoch - history length.
	endEpoch := currentEpoch - historyLength
	endPruneSlot, err := helpers.StartSlot(endEpoch)
	if err != nil {
		return err
	}
	encodedEndSlot := fssz.MarshalUint64([]byte{}, uint64(endPruneSlot))

	// We retrieve the lowest stored slot in the proposals bucket.
	var lowestSlot types.Slot
	if err = s.db.View(func(tx *bolt.Tx) error {
		proposalBkt := tx.Bucket(proposalRecordsBucket)
		c := proposalBkt.Cursor()
		k, _ := c.First()
		lowestSlot = slotFromProposalKey(k)
		return nil
	}); err != nil {
		return err
	}

	// We prune in increments of `pruningEpochIncrements` at a time to prevent
	// a long-running bolt transaction which overwhelms CPU and memory.
	lowestEpoch := helpers.SlotToEpoch(lowestSlot)
	slotCursor := lowestSlot
	var finished bool
	var encodedSlotCursor []byte
	for !finished {
		epochAtCursor := helpers.SlotToEpoch(slotCursor)
		log.Infof("Pruned %d/%d epochs worth of proposals", epochAtCursor-lowestEpoch, endEpoch-lowestEpoch)
		encodedSlotCursor = fssz.MarshalUint64([]byte{}, uint64(slotCursor))
		if err = s.db.Update(func(tx *bolt.Tx) error {
			proposalBkt := tx.Bucket(proposalRecordsBucket)
			c := proposalBkt.Cursor()

			var lastPrunedEpoch, epochsPruned types.Epoch
			// We begin a pruning iteration at starting from the current slot cursor.
			for k, _ := c.Seek(encodedSlotCursor); k != nil; k, _ = c.Next() {
				// If we have hit a slot that is greater than the end slot of the pruning process,
				// we then completely exit the process as we are done.
				if !slotPrefixLessThan(k, encodedEndSlot) {
					finished = true
					return nil
				}

				// We check the slot from the current key in the database.
				slot := slotFromProposalKey(k)
				epoch := helpers.SlotToEpoch(slot)
				slotCursor = slot

				// If we have pruned N epochs in this pruning iteration,
				// we exit from the bolt transaction.
				if epochsPruned >= pruningEpochIncrements {
					return nil
				}

				// Proposals in the database look like this:
				//  (slot ++ validatorIndex) => encode(proposal)
				// so it is possible we have a few adjacent objects that have the same slot, such as
				//  (slot = 3 ++ validatorIndex = 0) => ...
				//  (slot = 3 ++ validatorIndex = 1) => ...
				//  (slot = 3 ++ validatorIndex = 2) => ...
				// so we only mark an epoch as pruned if the epoch of the current object
				// under the cursor has changed.
				if epoch != lastPrunedEpoch {
					epochsPruned++
					lastPrunedEpoch = epoch
				}
				slasherProposalsPrunedTotal.Inc()

				// We delete the proposal object from the database.
				if err := proposalBkt.Delete(k); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			return err
		}
	}
	return nil
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

func slotFromProposalKey(key []byte) types.Slot {
	return types.Slot(binary.LittleEndian.Uint64(key[:8]))
}

func slotPrefixLessThan(key, lessThan []byte) bool {
	encSlot := key[:8]
	return bytes.Compare(encSlot, lessThan) < 0
}

func epochPrefixLessThan(key, lessThan []byte) bool {
	encSlot := key[:2]
	return bytes.Compare(encSlot, lessThan) < 0
}
