package db

import (
	"context"
	"encoding/hex"
	"path/filepath"

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

	allProposals, allAttestations, err := getAllProposalsAndAllAttestations(sourceStores)
	if err != nil {
		return err
	}
	if err := createMergeTargetStore(targetDirectory, allProposals, allAttestations); err != nil {
		return errors.Wrapf(err, "Could not create target store")
	}
	return nil
}

// Split splits data from sourceStore into several stores, one for each public key in sourceStore.
// Each new store is created in its own subdirectory inside targetDirectory.
func Split(ctx context.Context, sourceStore *Store, targetDirectory string) error {
	ctx, span := trace.StartSpan(ctx, "Validator.Db.Manage")
	defer span.End()

	allProposals, allAttestations, err := getAllProposalsAndAllAttestations([]*Store{sourceStore})
	if err != nil {
		return err
	}
	if err := createSplitTargetStores(targetDirectory, allProposals, allAttestations); err != nil {
		return errors.Wrapf(err, "Could not create split target store")
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

func createMergeTargetStore(
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
			if err := addProposals(proposalsBucket, pubKeyProposals); err != nil {
				return err
			}
		}
		attestationsBucket := tx.Bucket(historicAttestationsBucket)
		for _, attestations := range allAttestations {
			if err := addAttestations(attestationsBucket, attestations); err != nil {
				return err
			}
		}
		return nil
	})

	return err
}

func createSplitTargetStores(
	targetDirectory string,
	allProposals []pubKeyProposals,
	allAttestations []pubKeyAttestations) (err error) {

	var storesToClose []*Store
	defer (func() {
		for _, store := range storesToClose {
			if deferErr := store.Close(); deferErr != nil {
				err = errors.Wrap(deferErr, "Closing store failed")
			}
		}
	})()

	for _, pubKeyProposals := range allProposals {
		dirName := hex.EncodeToString(pubKeyProposals.PubKey)[:12]
		path := filepath.Join(targetDirectory, dirName)
		newStore, err := NewKVStore(path, [][48]byte{})
		if err != nil {
			return errors.Wrapf(err, "Could not create a validator database in %s", path)
		}
		storesToClose = append(storesToClose, newStore)

		if err := newStore.update(func(tx *bolt.Tx) error {
			proposalsBucket := tx.Bucket(historicProposalsBucket)
			if err := addProposals(proposalsBucket, pubKeyProposals); err != nil {
				return err
			}

			attestationsBucket := tx.Bucket(historicAttestationsBucket)
			for _, pubKeyAttestations := range allAttestations {
				if string(pubKeyAttestations.PubKey) == string(pubKeyProposals.PubKey) {
					if err := addAttestations(attestationsBucket, pubKeyAttestations); err != nil {
						return err
					}
					break
				}
			}

			return nil
		}); err != nil {
			return err
		}
	}

	// Create stores for attestations belonging to public keys that do not have proposals.
	for _, pubKeyAttestations := range allAttestations {
		var hasMatchingProposals = false
		for _, pubKeyProposals := range allProposals {
			if string(pubKeyAttestations.PubKey) == string(pubKeyProposals.PubKey) {
				hasMatchingProposals = true
				break
			}
		}
		if !hasMatchingProposals {
			dirName := hex.EncodeToString(pubKeyAttestations.PubKey)[:12]
			path := filepath.Join(targetDirectory, dirName)
			newStore, err := NewKVStore(path, [][48]byte{})
			if err != nil {
				return errors.Wrapf(err, "Could not create a validator database in %s", path)
			}
			storesToClose = append(storesToClose, newStore)

			if err := newStore.update(func(tx *bolt.Tx) error {
				attestationsBucket := tx.Bucket(historicAttestationsBucket)
				if err := addAttestations(attestationsBucket, pubKeyAttestations); err != nil {
					return err
				}

				return nil
			}); err != nil {
				return err
			}
		}
	}

	return nil
}

func getAllProposalsAndAllAttestations(stores []*Store) ([]pubKeyProposals, []pubKeyAttestations, error) {
	var allProposals []pubKeyProposals
	var allAttestations []pubKeyAttestations

	for _, store := range stores {
		if err := store.db.View(func(tx *bolt.Tx) error {
			proposalsBucket := tx.Bucket(historicProposalsBucket)
			if err := proposalsBucket.ForEach(func(pubKey, _ []byte) error {
				pubKeyProposals, err := getPubKeyProposals(pubKey, proposalsBucket)
				if err != nil {
					return errors.Wrapf(err, "Could not retrieve proposals for source in %s", store.databasePath)
				}
				allProposals = append(allProposals, *pubKeyProposals)
				return nil
			}); err != nil {
				return errors.Wrapf(err, "Could not retrieve proposals for source in %s", store.databasePath)
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
				return errors.Wrapf(err, "Could not retrieve attestations for source in %s", store.databasePath)
			}

			return nil
		}); err != nil {
			return nil, nil, err
		}
	}

	return allProposals, allAttestations, nil
}

func addProposals(bucket *bolt.Bucket, proposals pubKeyProposals) error {
	var proposalsPubKeyBucket, err = bucket.CreateBucket(proposals.PubKey)
	if err != nil {
		return errors.Wrapf(err, "Could not create proposals bucket for public key %x", proposals.PubKey[:12])
	}
	for _, epochProposals := range proposals.Proposals {
		if err := proposalsPubKeyBucket.Put(epochProposals.Epoch, epochProposals.Proposals); err != nil {
			return errors.Wrapf(err, "Could not add epoch proposals for epoch %v", epochProposals.Epoch)
		}
	}
	return nil
}

func addAttestations(bucket *bolt.Bucket, attestations pubKeyAttestations) error {
	if err := bucket.Put(attestations.PubKey, attestations.Attestations); err != nil {
		return errors.Wrapf(err, "Could not add public key attestations for public key %x", attestations.PubKey[:12])
	}
	return nil
}
