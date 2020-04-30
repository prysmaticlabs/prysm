package db

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

// ProposalHistoriesForEpoch accepts an array of validator public keys with an epoch and returns the corresponding proposal histories.
// Returns nil if there is no proposal history for the validator.
func (db *Store) ProposalHistoriesForEpoch(ctx context.Context, publicKeys [][48]byte, epoch uint64) (map[[48]byte]bitfield.Bitlist, error) {
	ctx, span := trace.StartSpan(ctx, "Validator.ProposalHistoryForEpoch")
	defer span.End()

	var err error
	proposalHistoriesForEpoch := make(map[[48]byte]bitfield.Bitlist)
	// Using 5 here since a bitfield length of 32 is always 5 bytes long.
	err = db.view(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(historicProposalsBucket)
		for _, pubKey := range publicKeys {
			valBucket := bucket.Bucket(pubKey[:])
			if valBucket == nil {
				return fmt.Errorf("validator history empty for public key %#x", pubKey)
			}
			slotBits := valBucket.Get(bytesutil.Bytes8(epoch))

			slotBitlist := make(bitfield.Bitlist, params.BeaconConfig().SlotsPerEpoch/8+1)
			if slotBits == nil || len(slotBits) == 0 {
				slotBitlist = bitfield.NewBitlist(params.BeaconConfig().SlotsPerEpoch)
			} else {
				copy(slotBitlist, slotBits)
			}
			proposalHistoriesForEpoch[pubKey] = slotBitlist
		}
		return nil
	})
	return proposalHistoriesForEpoch, err
}

// SaveProposalHistoriesForEpoch saves the provided proposal histories to the indicated validator public keys.
func (db *Store) SaveProposalHistoriesForEpoch(ctx context.Context, epoch uint64, proposalHistoriesForEpoch map[[48]byte]bitfield.Bitlist) error {
	ctx, span := trace.StartSpan(ctx, "Validator.SaveProposalHistoryForEpoch")
	defer span.End()

	err := db.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(historicProposalsBucket)
		for pubKey, history := range proposalHistoriesForEpoch {
			valBucket := bucket.Bucket(pubKey[:])
			if valBucket == nil {
				return fmt.Errorf("validator history is empty for validator %#x", pubKey)
			}
			if err := valBucket.Put(bytesutil.Bytes8(epoch), history); err != nil {
				return err
			}
			if err := pruneProposalHistory(valBucket, epoch); err != nil {
				return err
			}
		}
		return nil
	})
	return err
}

// DeleteProposalHistory deletes the proposal history for the corresponding validator public key.
func (db *Store) DeleteProposalHistory(ctx context.Context, pubkey []byte) error {
	ctx, span := trace.StartSpan(ctx, "Validator.DeleteProposalHistory")
	defer span.End()

	return db.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(historicProposalsBucket)
		if err := bucket.DeleteBucket(pubkey); err != nil {
			return errors.Wrap(err, "failed to delete the proposal history")
		}
		return nil
	})
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
			// If starting from the oldest, we stop finding anything prunable, stop pruning.
			break
		}
	}
	return nil
}

func (db *Store) initializeSubBuckets(pubKeys [][48]byte) error {
	return db.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(historicProposalsBucket)
		for _, pubKey := range pubKeys {
			if _, err := bucket.CreateBucketIfNotExists(pubKey[:]); err != nil {
				return errors.Wrap(err, "failed to create proposal history bucket")
			}
		}
		return nil
	})
}
