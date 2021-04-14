package slasherkv

import (
	"bytes"
	"context"
	"encoding/binary"

	fssz "github.com/ferranbt/fastssz"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	bolt "go.etcd.io/bbolt"
)

// PruneProposals prunes all proposal data older than currentEpoch - historyLength in
// specified epoch increments. BoltDB cannot handle long-running transactions, so we instead
// use a cursor-based mechanism to prune at pruningEpochIncrements by opening individual bolt
// transactions for each pruning iteration.
func (s *Store) PruneProposals(
	ctx context.Context, currentEpoch, pruningEpochIncrements, historyLength types.Epoch,
) error {
	if currentEpoch < historyLength {
		log.Debugf("Current epoch %d < history length %d, nothing to prune", currentEpoch, historyLength)
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

	// If the lowest slot is greater than or equal to the end pruning slot,
	// there is nothing to prune, so we return early.
	if lowestSlot >= endPruneSlot {
		log.Debugf("Lowest slot %d is >= pruning slot %d, nothing to prune", lowestSlot, endPruneSlot)
		return nil
	}

	// We prune in increments of `pruningEpochIncrements` at a time to prevent
	// a long-running bolt transaction which overwhelms CPU and memory.
	slotCursor := lowestSlot
	var encodedSlotCursor []byte

	// While we still have epochs to prune based on a cursor, we continue the pruning process.
	epochAtCursor := helpers.SlotToEpoch(slotCursor)
	for endEpoch-epochAtCursor > 0 {
		// Each pruning iteration involves a unique bolt transaction. Given pruning can be
		// a very expensive process which puts pressure on the database, we perform
		// the process in a batch-based method using a cursor to proceed to the next batch.
		log.Debugf("Pruned %d/%d epochs worth of proposals", endEpoch-epochAtCursor, endEpoch)
		encodedSlotCursor = fssz.MarshalUint64([]byte{}, uint64(slotCursor))
		if err = s.db.Update(func(tx *bolt.Tx) error {
			proposalBkt := tx.Bucket(proposalRecordsBucket)
			c := proposalBkt.Cursor()

			var lastPrunedEpoch, epochsPruned types.Epoch
			// We begin a pruning iteration at starting from the current slot cursor.
			for k, _ := c.Seek(encodedSlotCursor); k != nil; k, _ = c.Next() {
				// We check the slot from the current key in the database.
				// If we have hit a slot that is greater than the end slot of the pruning process,
				// we then completely exit the process as we are done.
				if !slotPrefixLessThan(k, encodedEndSlot) {
					// We don't want to unmarshal the key bytes every time
					// into the cursor value, so we only do it if needed here.
					slotCursor = slotFromProposalKey(k)
					return nil
				}
				slot := slotFromProposalKey(k)
				slotCursor = slot
				encodedSlotCursor = fssz.MarshalUint64([]byte{}, uint64(slotCursor))
				epochAtCursor = helpers.SlotToEpoch(slot)

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
				if epochAtCursor > lastPrunedEpoch {
					epochsPruned++
					lastPrunedEpoch = epochAtCursor
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
func (s *Store) PruneAttestations(ctx context.Context, currentEpoch, pruningEpochIncrements, historyLength types.Epoch) error {
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
