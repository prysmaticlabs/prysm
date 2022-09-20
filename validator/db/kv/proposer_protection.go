package kv

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

// ProposalHistoryForPubkey for a validator public key.
type ProposalHistoryForPubkey struct {
	Proposals []Proposal
}

// Proposal representation for a validator public key.
type Proposal struct {
	Slot        types.Slot `json:"slot"`
	SigningRoot []byte     `json:"signing_root"`
}

// ProposedPublicKeys retrieves all public keys in our proposals history bucket.
func (s *Store) ProposedPublicKeys(ctx context.Context) ([][fieldparams.BLSPubkeyLength]byte, error) {
	ctx, span := trace.StartSpan(ctx, "Validator.ProposedPublicKeys")
	defer span.End()
	var err error
	proposedPublicKeys := make([][fieldparams.BLSPubkeyLength]byte, 0)
	err = s.view(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(historicProposalsBucket)
		return bucket.ForEach(func(key []byte, _ []byte) error {
			pubKeyBytes := [fieldparams.BLSPubkeyLength]byte{}
			copy(pubKeyBytes[:], key)
			proposedPublicKeys = append(proposedPublicKeys, pubKeyBytes)
			return nil
		})
	})
	return proposedPublicKeys, err
}

// ProposalHistoryForSlot accepts a validator public key and returns the corresponding signing root as well
// as a boolean that tells us if we have a proposal history stored at the slot. It is possible we have proposed
// a slot but stored a nil signing root, so the boolean helps give full information.
func (s *Store) ProposalHistoryForSlot(ctx context.Context, publicKey [fieldparams.BLSPubkeyLength]byte, slot types.Slot) ([32]byte, bool, error) {
	ctx, span := trace.StartSpan(ctx, "Validator.ProposalHistoryForSlot")
	defer span.End()

	var err error
	var proposalExists bool
	signingRoot := [32]byte{}
	err = s.view(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(historicProposalsBucket)
		valBucket := bucket.Bucket(publicKey[:])
		if valBucket == nil {
			return nil
		}
		signingRootBytes := valBucket.Get(bytesutil.SlotToBytesBigEndian(slot))
		if signingRootBytes == nil {
			return nil
		}
		proposalExists = true
		copy(signingRoot[:], signingRootBytes)
		return nil
	})
	return signingRoot, proposalExists, err
}

// ProposalHistoryForPubKey returns the entire proposal history for a given public key.
func (s *Store) ProposalHistoryForPubKey(ctx context.Context, publicKey [fieldparams.BLSPubkeyLength]byte) ([]*Proposal, error) {
	ctx, span := trace.StartSpan(ctx, "Validator.ProposalHistoryForPubKey")
	defer span.End()

	proposals := make([]*Proposal, 0)
	err := s.view(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(historicProposalsBucket)
		valBucket := bucket.Bucket(publicKey[:])
		if valBucket == nil {
			return nil
		}
		return valBucket.ForEach(func(slotKey, signingRootBytes []byte) error {
			slot := bytesutil.BytesToSlotBigEndian(slotKey)
			sr := make([]byte, fieldparams.RootLength)
			copy(sr, signingRootBytes)
			proposals = append(proposals, &Proposal{
				Slot:        slot,
				SigningRoot: sr,
			})
			return nil
		})
	})
	return proposals, err
}

// SaveProposalHistoryForSlot saves the proposal history for the requested validator public key.
// We also check if the incoming proposal slot is lower than the lowest signed proposal slot
// for the validator and override its value on disk.
func (s *Store) SaveProposalHistoryForSlot(ctx context.Context, pubKey [fieldparams.BLSPubkeyLength]byte, slot types.Slot, signingRoot []byte) error {
	ctx, span := trace.StartSpan(ctx, "Validator.SaveProposalHistoryForEpoch")
	defer span.End()

	err := s.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(historicProposalsBucket)
		valBucket, err := bucket.CreateBucketIfNotExists(pubKey[:])
		if err != nil {
			return fmt.Errorf("could not create bucket for public key %#x", pubKey)
		}

		// If the incoming slot is lower than the lowest signed proposal slot, override.
		lowestSignedBkt := tx.Bucket(lowestSignedProposalsBucket)
		lowestSignedProposalBytes := lowestSignedBkt.Get(pubKey[:])
		var lowestSignedProposalSlot types.Slot
		if len(lowestSignedProposalBytes) >= 8 {
			lowestSignedProposalSlot = bytesutil.BytesToSlotBigEndian(lowestSignedProposalBytes)
		}
		if len(lowestSignedProposalBytes) == 0 || slot < lowestSignedProposalSlot {
			if err := lowestSignedBkt.Put(pubKey[:], bytesutil.SlotToBytesBigEndian(slot)); err != nil {
				return err
			}
		}

		// If the incoming slot is higher than the highest signed proposal slot, override.
		highestSignedBkt := tx.Bucket(highestSignedProposalsBucket)
		highestSignedProposalBytes := highestSignedBkt.Get(pubKey[:])
		var highestSignedProposalSlot types.Slot
		if len(highestSignedProposalBytes) >= 8 {
			highestSignedProposalSlot = bytesutil.BytesToSlotBigEndian(highestSignedProposalBytes)
		}
		if len(highestSignedProposalBytes) == 0 || slot > highestSignedProposalSlot {
			if err := highestSignedBkt.Put(pubKey[:], bytesutil.SlotToBytesBigEndian(slot)); err != nil {
				return err
			}
		}

		if err := valBucket.Put(bytesutil.SlotToBytesBigEndian(slot), signingRoot); err != nil {
			return err
		}
		return pruneProposalHistoryBySlot(valBucket, slot)
	})
	return err
}

// LowestSignedProposal returns the lowest signed proposal slot for a validator public key.
// If no data exists, a boolean of value false is returned.
func (s *Store) LowestSignedProposal(ctx context.Context, publicKey [fieldparams.BLSPubkeyLength]byte) (types.Slot, bool, error) {
	ctx, span := trace.StartSpan(ctx, "Validator.LowestSignedProposal")
	defer span.End()

	var err error
	var lowestSignedProposalSlot types.Slot
	var exists bool
	err = s.view(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(lowestSignedProposalsBucket)
		lowestSignedProposalBytes := bucket.Get(publicKey[:])
		// 8 because bytesutil.BytesToUint64BigEndian will return 0 if input is less than 8 bytes.
		if len(lowestSignedProposalBytes) < 8 {
			return nil
		}
		exists = true
		lowestSignedProposalSlot = bytesutil.BytesToSlotBigEndian(lowestSignedProposalBytes)
		return nil
	})
	return lowestSignedProposalSlot, exists, err
}

// HighestSignedProposal returns the highest signed proposal slot for a validator public key.
// If no data exists, a boolean of value false is returned.
func (s *Store) HighestSignedProposal(ctx context.Context, publicKey [fieldparams.BLSPubkeyLength]byte) (types.Slot, bool, error) {
	ctx, span := trace.StartSpan(ctx, "Validator.HighestSignedProposal")
	defer span.End()

	var err error
	var highestSignedProposalSlot types.Slot
	var exists bool
	err = s.view(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(highestSignedProposalsBucket)
		highestSignedProposalBytes := bucket.Get(publicKey[:])
		// 8 because bytesutil.BytesToUint64BigEndian will return 0 if input is less than 8 bytes.
		if len(highestSignedProposalBytes) < 8 {
			return nil
		}
		exists = true
		highestSignedProposalSlot = bytesutil.BytesToSlotBigEndian(highestSignedProposalBytes)
		return nil
	})
	return highestSignedProposalSlot, exists, err
}

func pruneProposalHistoryBySlot(valBucket *bolt.Bucket, newestSlot types.Slot) error {
	c := valBucket.Cursor()
	for k, _ := c.First(); k != nil; k, _ = c.First() {
		slot := bytesutil.BytesToSlotBigEndian(k)
		epoch := slots.ToEpoch(slot)
		newestEpoch := slots.ToEpoch(newestSlot)
		// Only delete epochs that are older than the weak subjectivity period.
		if epoch+params.BeaconConfig().WeakSubjectivityPeriod <= newestEpoch {
			if err := c.Delete(); err != nil {
				return errors.Wrapf(err, "could not prune epoch %d in proposal history", epoch)
			}
		} else {
			// If starting from the oldest, we don't find anything prunable, stop pruning.
			break
		}
	}
	return nil
}
