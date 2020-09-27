package kv

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

// ProposalHistoryForSlot accepts a validator public key and returns the corresponding signing root.
// Returns nil if there is no proposal history for the validator at this slot.
func (store *Store) ProposalHistoryForSlot(ctx context.Context, publicKey []byte, slot uint64) ([]byte, error) {
	ctx, span := trace.StartSpan(ctx, "Validator.ProposalHistoryForSlot")
	defer span.End()

	var err error
	signingRoot := make([]byte, 32)
	err = store.view(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(newhistoricProposalsBucket)
		valBucket := bucket.Bucket(publicKey)
		if valBucket == nil {
			return fmt.Errorf("validator history empty for public key %#x", publicKey)
		}
		sr := valBucket.Get(bytesutil.Uint64ToBytesBigEndian(slot))
		if sr == nil || len(sr) == 0 {
			return nil
		}
		copy(signingRoot, sr)
		return nil
	})
	return signingRoot, err
}

// SaveProposalHistoryForSlot saves the proposal history for the requested validator public key.
func (store *Store) SaveProposalHistoryForSlot(ctx context.Context, pubKey []byte, slot uint64, signingRoot []byte) error {
	ctx, span := trace.StartSpan(ctx, "Validator.SaveProposalHistoryForEpoch")
	defer span.End()

	err := store.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(newhistoricProposalsBucket)
		valBucket := bucket.Bucket(pubKey)
		if valBucket == nil {
			return fmt.Errorf("validator history is empty for validator %#x", pubKey)
		}
		if err := valBucket.Put(bytesutil.Uint64ToBytesBigEndian(slot), signingRoot); err != nil {
			return err
		}
		if err := pruneProposalHistoryBySlot(valBucket, slot); err != nil {
			return err
		}
		return nil
	})
	return err
}

// UpdatePublicKeysNewBuckets for a specified list of keys.
func (store *Store) UpdatePublicKeysNewBuckets(pubKeys [][48]byte) error {
	return store.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(newhistoricProposalsBucket)
		for _, pubKey := range pubKeys {
			if _, err := bucket.CreateBucketIfNotExists(pubKey[:]); err != nil {
				return errors.Wrap(err, "failed to create proposal history bucket")
			}
		}
		return nil
	})
}

func pruneProposalHistoryBySlot(valBucket *bolt.Bucket, newestSlot uint64) error {
	c := valBucket.Cursor()
	for k, _ := c.First(); k != nil; k, _ = c.First() {
		slot := bytesutil.BytesToUint64BigEndian(k)
		epoch := helpers.SlotToEpoch(slot)
		newestEpoch := helpers.SlotToEpoch(newestSlot)
		// Only delete epochs that are older than the weak subjectivity period.
		if epoch+params.BeaconConfig().WeakSubjectivityPeriod <= newestEpoch {
			if err := c.Delete(); err != nil {
				return errors.Wrapf(err, "could not prune epoch %d in proposal history", epoch)
			}
		} else {
			// If starting from the oldest, we dont find anything prunable, stop pruning.
			break
		}
	}
	return nil
}
