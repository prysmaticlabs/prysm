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
	encodedEndPruneSlot := fssz.MarshalUint64([]byte{}, uint64(endPruneSlot))

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
	// While we still have epochs to prune based on a cursor, we continue the pruning process.
	epochAtCursor := helpers.SlotToEpoch(lowestSlot)
	for epochAtCursor < endEpoch {
		// Each pruning iteration involves a unique bolt transaction. Given pruning can be
		// a very expensive process which puts pressure on the database, we perform
		// the process in a batch-based method using a cursor to proceed to the next batch.
		log.Debugf("Pruned %d/%d epochs worth of proposals", epochAtCursor, endEpoch-1)
		if err = s.db.Update(func(tx *bolt.Tx) error {
			proposalBkt := tx.Bucket(proposalRecordsBucket)
			c := proposalBkt.Cursor()

			var lastPrunedEpoch, epochsPruned types.Epoch
			// We begin a pruning iteration at starting from the first item in the bucket.
			for k, _ := c.First(); k != nil; k, _ = c.Next() {
				// We check the slot from the current key in the database.
				// If we have hit a slot that is greater than the end slot of the pruning process,
				// we then completely exit the process as we are done.
				if !uint64PrefixLessThan(k, encodedEndPruneSlot) {
					epochAtCursor = endEpoch
					return nil
				}
				epochAtCursor = helpers.SlotToEpoch(slotFromProposalKey(k))

				// Proposals in the database look like this:
				//  (slot ++ validatorIndex) => encode(proposal)
				// so it is possible we have a few adjacent objects that have the same slot, such as
				//  (slot = 3 ++ validatorIndex = 0) => ...
				//  (slot = 3 ++ validatorIndex = 1) => ...
				//  (slot = 3 ++ validatorIndex = 2) => ...
				// so we only mark an epoch as pruned if the epoch of the current object
				// under the cursor has changed.
				if err := proposalBkt.Delete(k); err != nil {
					return err
				}
				if epochAtCursor == 0 {
					epochsPruned = 1
				} else if epochAtCursor > lastPrunedEpoch {
					epochsPruned++
					lastPrunedEpoch = epochAtCursor
				}
				slasherProposalsPrunedTotal.Inc()

				// If we have pruned N epochs in this pruning iteration,
				// we exit from the bolt transaction.
				if epochsPruned >= pruningEpochIncrements {
					return nil
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
func (s *Store) PruneAttestations(
	ctx context.Context, currentEpoch, pruningEpochIncrements, historyLength types.Epoch,
) error {
	if currentEpoch < historyLength {
		log.Debugf("Current epoch %d < history length %d, nothing to prune", currentEpoch, historyLength)
		return nil
	}
	// We can prune everything less than the current epoch - history length.
	endPruneEpoch := currentEpoch - historyLength
	encodedEndPruneEpoch := fssz.MarshalUint64([]byte{}, uint64(endPruneEpoch))

	// We retrieve the lowest stored epoch in the proposals bucket.
	var lowestEpoch types.Epoch
	if err := s.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(attestationDataRootsBucket)
		c := bkt.Cursor()
		k, _ := c.First()
		lowestEpoch = types.Epoch(binary.LittleEndian.Uint64(k))
		return nil
	}); err != nil {
		return err
	}

	// If the lowest slot is greater than or equal to the end pruning slot,
	// there is nothing to prune, so we return early.
	if lowestEpoch >= endPruneEpoch {
		log.Debugf("Lowest epoch %d is >= pruning epoch %d, nothing to prune", lowestEpoch, endPruneEpoch)
		return nil
	}

	// We prune in increments of `pruningEpochIncrements` at a time to prevent
	// a long-running bolt transaction which overwhelms CPU and memory.
	// While we still have epochs to prune based on a cursor, we continue the pruning process.
	epochAtCursor := lowestEpoch
	for epochAtCursor < endPruneEpoch {
		// Each pruning iteration involves a unique bolt transaction. Given pruning can be
		// a very expensive process which puts pressure on the database, we perform
		// the process in a batch-based method using a cursor to proceed to the next batch.
		log.Debugf("Pruned %d/%d epochs worth of attestations", epochAtCursor, endPruneEpoch-1)
		if err := s.db.Update(func(tx *bolt.Tx) error {
			rootsBkt := tx.Bucket(attestationDataRootsBucket)
			attsBkt := tx.Bucket(attestationRecordsBucket)
			c := rootsBkt.Cursor()

			var lastPrunedEpoch, epochsPruned types.Epoch
			// We begin a pruning iteration at starting from the first item in the bucket.
			for k, v := c.First(); k != nil; k, v = c.Next() {
				// We check the epoch from the current key in the database.
				// If we have hit an epoch that is greater than the end epoch of the pruning process,
				// we then completely exit the process as we are done.
				if !uint64PrefixLessThan(k, encodedEndPruneEpoch) {
					epochAtCursor = endPruneEpoch
					return nil
				}

				epochAtCursor = types.Epoch(binary.LittleEndian.Uint64(k))

				// Attestation in the database look like this:
				//  (target_epoch ++ _) => encode(attestation)
				// so it is possible we have a few adjacent objects that have the same slot, such as
				//  (target_epoch = 3 ++ _) => encode(attestation)
				// so we only mark an epoch as pruned if the epoch of the current object
				// under the cursor has changed.
				if err := rootsBkt.Delete(k); err != nil {
					return err
				}
				if err := attsBkt.Delete(v); err != nil {
					return err
				}
				if epochAtCursor == 0 {
					epochsPruned = 1
				} else if epochAtCursor > lastPrunedEpoch {
					epochsPruned++
					lastPrunedEpoch = epochAtCursor
				}
				slasherAttestationsPrunedTotal.Inc()

				// If we have pruned N epochs in this pruning iteration,
				// we exit from the bolt transaction.
				if epochsPruned >= pruningEpochIncrements {
					return nil
				}
			}
			return nil
		}); err != nil {
			return err
		}
	}
	return nil
}

func slotFromProposalKey(key []byte) types.Slot {
	return types.Slot(binary.LittleEndian.Uint64(key[:8]))
}

func uint64PrefixLessThan(key, lessThan []byte) bool {
	enc := key[:8]
	return bytes.Compare(enc, lessThan) < 0
}
