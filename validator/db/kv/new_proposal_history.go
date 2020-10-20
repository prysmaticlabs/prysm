package kv

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	log "github.com/sirupsen/logrus"
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
			return fmt.Errorf("validator history empty for public key: %#x", publicKey)
		}
		sr := valBucket.Get(bytesutil.Uint64ToBytesBigEndian(slot))
		if len(sr) == 0 {
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
		valBucket, err := bucket.CreateBucketIfNotExists(pubKey)
		if err != nil {
			return fmt.Errorf("could not create bucket for public key %#x", pubKey)
		}
		if err := valBucket.Put(bytesutil.Uint64ToBytesBigEndian(slot), signingRoot); err != nil {
			return err
		}
		return pruneProposalHistoryBySlot(valBucket, slot)
	})
	return err
}

// MigrateV2ProposalFormat accepts a validator public key and returns the corresponding signing root.
// Returns nil if there is no proposal history for the validator at this slot.
func (store *Store) MigrateV2ProposalFormat(ctx context.Context) error {
	ctx, span := trace.StartSpan(ctx, "Validator.MigrateV2ProposalFormat")
	defer span.End()

	var allKeys [][48]byte
	err := store.db.View(func(tx *bolt.Tx) error {
		proposalsBucket := tx.Bucket(historicProposalsBucket)
		if err := proposalsBucket.ForEach(func(pubKey, _ []byte) error {
			var pubKeyCopy [48]byte
			copy(pubKeyCopy[:], pubKey)
			allKeys = append(allKeys, pubKeyCopy)
			return nil
		}); err != nil {
			return errors.Wrapf(err, "could not retrieve proposals for source in %s", store.databasePath)
		}
		return nil
	})
	if err != nil {
		return err
	}
	allKeys = removeDuplicateKeys(allKeys)
	var prs []*pubKeyProposals
	err = store.db.View(func(tx *bolt.Tx) error {
		proposalsBucket := tx.Bucket(historicProposalsBucket)
		for _, pk := range allKeys {
			pr, err := getPubKeyProposals(pk, proposalsBucket)
			prs = append(prs, pr)
			if err != nil {
				return errors.Wrap(err, "could not retrieve public key old proposals format")
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	err = store.db.Update(func(tx *bolt.Tx) error {
		newProposalsBucket := tx.Bucket(newhistoricProposalsBucket)
		for _, pr := range prs {
			valBucket, err := newProposalsBucket.CreateBucketIfNotExists(pr.PubKey[:])
			if err != nil {
				return errors.Wrap(err, "could not could not create bucket for public key")
			}
			for _, epochProposals := range pr.Proposals {
				// Adding an extra byte for the bitlist length.
				slotBitlist := make(bitfield.Bitlist, params.BeaconConfig().SlotsPerEpoch/8+1)
				slotBits := epochProposals.Proposals
				if len(slotBits) == 0 {
					continue
				}
				copy(slotBitlist, slotBits)
				for i := uint64(0); i < params.BeaconConfig().SlotsPerEpoch; i++ {
					if slotBitlist.BitAt(i) {
						ss, err := helpers.StartSlot(bytesutil.FromBytes8(epochProposals.Epoch))
						if err != nil {
							return errors.Wrapf(err, "failed to get start slot of epoch: %d", epochProposals.Epoch)
						}
						if err := valBucket.Put(bytesutil.Uint64ToBytesBigEndian(ss+i), []byte{1}); err != nil {
							return err
						}
					}
				}
			}

		}
		return nil
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

// MigrateV2ProposalsProtectionDb exports old proposal protection data format to the
// new format and save the exported flag to database.
func (store *Store) MigrateV2ProposalsProtectionDb(ctx context.Context) error {
	importProposals, err := store.shouldImportProposals()
	if err != nil {
		return err
	}

	if !importProposals {
		return nil
	}
	log.Info("Starting proposals protection db migration to v2...")
	if err := store.MigrateV2ProposalFormat(ctx); err != nil {
		return err
	}
	err = store.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(historicProposalsBucket)
		if bucket != nil {
			if err := bucket.Put([]byte(proposalExported), []byte{1}); err != nil {
				return errors.Wrap(err, "failed to set exported proposals flag in db")
			}
		}
		return nil
	})
	log.Info("Finished proposals protection db migration to v2")
	return err
}

func (store *Store) shouldImportProposals() (bool, error) {
	var importProposals bool
	err := store.db.View(func(tx *bolt.Tx) error {
		proposalBucket := tx.Bucket(historicProposalsBucket)
		if proposalBucket != nil && proposalBucket.Stats().KeyN != 0 {
			if exported := proposalBucket.Get([]byte(proposalExported)); exported == nil {
				importProposals = true
			}
		}
		return nil
	})
	return importProposals, err
}
