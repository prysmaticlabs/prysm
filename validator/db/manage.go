package db

import (
	"context"

	"github.com/pkg/errors"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

type epochProposals struct {
	Epoch     []byte
	Proposals []byte
}

type pubKeyProposals struct {
	PubKey    []byte
	Proposals []epochProposals
}

type pubKeyAttestations struct {
	PubKey       []byte
	Attestations []byte
}

// Merge merges data from sourceStores into a new store, which is created in targetDirectory.
func Merge(ctx context.Context, sourceStores []*Store, targetDirectory string) error {
	ctx, span := trace.StartSpan(ctx, "Validator.Db.Manage")
	defer span.End()

	var allProposals []pubKeyProposals
	var allAttestations []pubKeyAttestations

	for _, store := range sourceStores {
		if err := store.db.View(func(tx *bolt.Tx) error {
			proposalsBucket := tx.Bucket(historicProposalsBucket)
			if err := proposalsBucket.ForEach(func(pubKey, _ []byte) error {
				pubKeyProposals, err := getPubKeyProposals(pubKey, proposalsBucket)
				if err != nil {
					return errors.Wrapf(err, "Could not retrieve proposals for database in %s", store.databasePath)
				}
				allProposals = append(allProposals, *pubKeyProposals)
				return nil
			}); err != nil {
				return errors.Wrapf(err, "Could not retrieve proposals for database in %s", store.databasePath)
			}

			attestationsBucket := tx.Bucket(historicAttestationsBucket)
			if err := attestationsBucket.ForEach(func(pubKey, v []byte) error {
				attestations := pubKeyAttestations{
					PubKey:       make([]byte, len(pubKey)),
					Attestations: make([]byte, len(v)),
				}
				copy(attestations.PubKey, pubKey)
				copy(attestations.Attestations, v)
				allAttestations = append(allAttestations, attestations)
				return nil
			}); err != nil {
				return errors.Wrapf(err, "Could not retrieve attestations for database in %s", store.databasePath)
			}

			return nil
		}); err != nil {
			return err
		}
	}

	if err := createTargetStore(targetDirectory, allProposals, allAttestations); err != nil {
		return errors.Wrapf(err, "Could not create target store")
	}

	return nil
}

func getPubKeyProposals(pubKey []byte, proposalsBucket *bolt.Bucket) (*pubKeyProposals, error) {
	pubKeyProposals := pubKeyProposals{
		PubKey:    pubKey,
		Proposals: []epochProposals{},
	}

	pubKeyBucket := proposalsBucket.Bucket(pubKey)
	if err := pubKeyBucket.ForEach(func(epoch, v []byte) error {
		epochProposals := epochProposals{
			Epoch:     make([]byte, len(epoch)),
			Proposals: make([]byte, len(v)),
		}
		copy(epochProposals.Epoch, epoch)
		copy(epochProposals.Proposals, v)
		pubKeyProposals.Proposals = append(pubKeyProposals.Proposals, epochProposals)
		return nil
	}); err != nil {
		return nil, err
	}

	return &pubKeyProposals, nil
}

func createTargetStore(
	targetDirectory string,
	allProposals []pubKeyProposals,
	allAttestations []pubKeyAttestations) (err error) {

	newStore, err := NewKVStore(targetDirectory, [][48]byte{})
	defer func() {
		if deferErr := newStore.Close(); deferErr != nil {
			err = errors.Wrap(err, "Could not close the merged database")
		}
	}()
	if err != nil {
		return errors.Wrapf(err, "Could not initialize a new database in %s", targetDirectory)
	}

	err = newStore.update(func(tx *bolt.Tx) error {
		proposalsBucket := tx.Bucket(historicProposalsBucket)
		for _, pubKeyProposals := range allProposals {
			pubKeyBucket, err := proposalsBucket.CreateBucket(pubKeyProposals.PubKey)
			if err != nil {
				return errors.Wrapf(err,
					"Could not create proposals bucket for public key %x",
					pubKeyProposals.PubKey[:12])
			}
			for _, epochProposals := range pubKeyProposals.Proposals {
				if err := pubKeyBucket.Put(epochProposals.Epoch, epochProposals.Proposals); err != nil {
					return errors.Wrapf(err, "Could not add epoch proposals for epoch %v", epochProposals.Epoch)
				}
			}
		}
		attestationsBucket := tx.Bucket(historicAttestationsBucket)
		for _, attestations := range allAttestations {
			if err := attestationsBucket.Put(attestations.PubKey, attestations.Attestations); err != nil {
				return errors.Wrapf(
					err,
					"Could not add public key attestations for public key %x",
					attestations.PubKey[:12])
			}
		}
		return nil
	})

	return err
}
