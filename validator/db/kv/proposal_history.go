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

// ProposalHistoryForPubkey for a validator public key.
type ProposalHistoryForPubkey struct {
	Proposals []Proposal
}

// Proposal representation as a simple combination of a slot and signing root.
type Proposal struct {
	Slot        uint64 `json:"slot"`
	SigningRoot []byte `json:"signing_root"`
}

// ProposalHistoryForSlot accepts a validator public key and returns the corresponding signing root.
// Returns nil if there is no proposal history for the validator at this slot.
func (store *Store) ProposalHistoryForSlot(ctx context.Context, publicKey []byte, slot uint64) ([]byte, uint64, error) {
	ctx, span := trace.StartSpan(ctx, "Validator.ProposalHistoryForSlot")
	defer span.End()

	var err error
	noDataFound := false
	var minimalSlot uint64
	signingRoot := make([]byte, 32)
	err = store.view(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(newhistoricProposalsBucket)
		valBucket := bucket.Bucket(publicKey)
		if valBucket == nil {
			return fmt.Errorf("validator history empty for public key: %#x", publicKey)
		}
		sr := valBucket.Get(bytesutil.Uint64ToBytesBigEndian(slot))
		min := valBucket.Get(minimalProposalSlotKey)
		minimalSlot = bytesutil.BytesToUint64BigEndian(min)
		if len(sr) == 0 {
			noDataFound = true
			return nil
		}
		copy(signingRoot, sr)
		return nil
	})
	if noDataFound {
		return nil, minimalSlot, nil
	}
	return signingRoot, minimalSlot, err
}

// SaveProposalHistoryForPubKeys saves the proposal histories for the provided validator public keys.
func (store *Store) SaveProposalHistoryForPubKeys(
	ctx context.Context,
	historyByPubKeys map[[48]byte]ProposalHistoryForPubkey,
) error {
	ctx, span := trace.StartSpan(ctx, "Validator.SaveProposalHistoryForPubKeys")
	defer span.End()

	minimalProposalSlot := make(map[[48]byte]uint64)
	err := store.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(newhistoricProposalsBucket)
		for pubKey, history := range historyByPubKeys {
			valBucket, err := bucket.CreateBucketIfNotExists(pubKey[:])
			if err != nil {
				return fmt.Errorf("could not create bucket for public key %#x", pubKey)
			}
			for _, proposal := range history.Proposals {
				minimalSlot, ok := minimalProposalSlot[pubKey]
				if !ok || (ok && proposal.Slot < minimalSlot) {
					minimalProposalSlot[pubKey] = proposal.Slot
					if err = valBucket.Put(minimalProposalSlotKey, bytesutil.Uint64ToBytesBigEndian(proposal.Slot)); err != nil {
						return err
					}
				}
				if err := valBucket.Put(bytesutil.Uint64ToBytesBigEndian(proposal.Slot), proposal.SigningRoot); err != nil {
					return err
				}
			}
		}
		return nil
	})
	return err
}

// SaveProposalHistoryForSlot saves the proposal history for the requested validator public key.
func (store *Store) SaveProposalHistoryForSlot(ctx context.Context, pubKey []byte, slot uint64, signingRoot []byte) error {
	ctx, span := trace.StartSpan(ctx, "Validator.SaveProposalHistoryForSlot")
	defer span.End()

	err := store.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(newhistoricProposalsBucket)
		valBucket, err := bucket.CreateBucketIfNotExists(pubKey)
		if err != nil {
			return fmt.Errorf("could not create bucket for public key %#x", pubKey)
		}
		enc := valBucket.Get(minimalProposalSlotKey)
		var minSlot uint64
		if len(enc) != 0 {
			minSlot = bytesutil.BytesToUint64BigEndian(enc)
		}
		if len(enc) == 0 || slot < minSlot {
			if err := valBucket.Put(minimalProposalSlotKey, bytesutil.Uint64ToBytesBigEndian(slot)); err != nil {
				return err
			}
		}
		if err := valBucket.Put(bytesutil.Uint64ToBytesBigEndian(slot), signingRoot); err != nil {
			return err
		}
		return pruneProposalHistoryBySlot(valBucket, slot)
	})
	return err
}

// UpdatePublicKeysBuckets for a specified list of keys.
func (store *Store) UpdatePublicKeysBuckets(pubKeys [][48]byte) error {
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
