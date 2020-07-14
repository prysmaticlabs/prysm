package kv

import (
	"context"
	"encoding/binary"
	"fmt"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/wealdtech/go-bytesutil"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

// ProposalHistoryForEpoch accepts a validator public key and returns the corresponding proposal history.
// Returns nil if there is no proposal history for the validator.
func (store *Store) ProposalHistoryForEpoch(ctx context.Context, publicKey []byte, epoch uint64) (bitfield.Bitlist, error) {
	ctx, span := trace.StartSpan(ctx, "Validator.ProposalHistoryForEpoch")
	defer span.End()

	var err error
	// Adding an extra byte for the bitlist length.
	slotBitlist := make(bitfield.Bitlist, params.BeaconConfig().SlotsPerEpoch/8+1)
	err = store.view(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(historicProposalsBucket)
		valBucket := bucket.Bucket(publicKey)
		if valBucket == nil {
			return fmt.Errorf("validator history empty for public key %#x", publicKey)
		}
		slotBits := valBucket.Get(bytesutil.Bytes8(epoch))
		if slotBits == nil || len(slotBits) == 0 {
			slotBitlist = bitfield.NewBitlist(params.BeaconConfig().SlotsPerEpoch)
			return nil
		}
		copy(slotBitlist, slotBits)
		return nil
	})
	return slotBitlist, err
}

// SaveProposalHistoryForEpoch saves the proposal history for the requested validator public key.
func (store *Store) SaveProposalHistoryForEpoch(ctx context.Context, pubKey []byte, epoch uint64, slotBits bitfield.Bitlist) error {
	ctx, span := trace.StartSpan(ctx, "Validator.SaveProposalHistoryForEpoch")
	defer span.End()

	err := store.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(historicProposalsBucket)
		valBucket := bucket.Bucket(pubKey)
		if valBucket == nil {
			return fmt.Errorf("validator history is empty for validator %#x", pubKey)
		}
		if err := valBucket.Put(bytesutil.Bytes8(epoch), slotBits); err != nil {
			return err
		}
		if err := pruneProposalHistory(valBucket, epoch); err != nil {
			return err
		}
		return nil
	})
	return err
}

func pruneProposalHistory(valBucket *bolt.Bucket, newestEpoch uint64) error {
	c := valBucket.Cursor()
	for k, _ := c.First(); k != nil; k, _ = c.First() {
		epoch := binary.LittleEndian.Uint64(k)
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

func (store *Store) initializeSubBuckets(pubKeys [][48]byte) error {
	return store.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(historicProposalsBucket)
		for _, pubKey := range pubKeys {
			if _, err := bucket.CreateBucketIfNotExists(pubKey[:]); err != nil {
				return errors.Wrap(err, "failed to create proposal history bucket")
			}
		}
		return nil
	})
}
