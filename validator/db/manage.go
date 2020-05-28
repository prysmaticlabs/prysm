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

	var allProposals []pubKeyProposals
	var allAttestations []pubKeyAttestations

	for _, store := range sourceStores {
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
			return err
		}
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

	var allProposals []pubKeyProposals
	var allAttestations []pubKeyAttestations

	if err := sourceStore.db.View(func(tx *bolt.Tx) error {
		proposalsBucket := tx.Bucket(historicProposalsBucket)
		if err := proposalsBucket.ForEach(func(pubKey, _ []byte) error {
			pubKeyProposals, err := getPubKeyProposals(pubKey, proposalsBucket)
			if err != nil {
				return errors.Wrapf(err, "Could not retrieve proposals for source in %s", sourceStore.databasePath)
			}
			allProposals = append(allProposals, *pubKeyProposals)
			return nil
		}); err != nil {
			return errors.Wrapf(err, "Could not retrieve proposals for source in %s", sourceStore.databasePath)
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
			return errors.Wrapf(err, "Could not retrieve attestations for source in %s", sourceStore.databasePath)
		}

		return nil
	}); err != nil {
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
	allAttestations []pubKeyAttestations) error {

	newStore, err := NewKVStore(targetDirectory)
	defer func() {
		if e := newStore.Close(); e != nil {
			err = errors.Wrap(err, "Could not close the merged database")
		}
	}()
	if err != nil {
		return errors.Wrapf(err, "Could not initialize a new database in %s", targetDirectory)
	}

	if err := newStore.update(func(tx *bolt.Tx) error {
		proposalsBucket := tx.Bucket(historicProposalsBucket)
		for _, pubKeyProposals := range allProposals {
			pubKeyBucket, err := proposalsBucket.CreateBucket(pubKeyProposals.PubKey)
			if err != nil {
				return errors.Wrapf(err, "Could not create proposals bucket for public key %v", pubKeyProposals.PubKey)
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
				return errors.Wrapf(err, "Could not add public key attestations for public key %v", attestations.PubKey)
			}
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func createSplitTargetStores(
	targetDirectory string,
	allProposals []pubKeyProposals,
	allAttestations []pubKeyAttestations) (err error) {

	var storesToClose []*Store
	defer(func(){
		for _, store := range storesToClose {
			if err := store.Close(); err != nil {
				err = errors.Wrap(err, "Closing store failed")
			}
		}
	})()

	for _, pubKeyProposals := range allProposals {
		dirName := hex.EncodeToString(pubKeyProposals.PubKey)[:12]
		path := filepath.Join(targetDirectory, dirName)
		newStore, err := NewKVStore(path)
		if err != nil {
			return errors.Wrapf(err, "Could not create a validator database in %s", path)
		}
		storesToClose = append(storesToClose, newStore)

		if err := newStore.update(func(tx *bolt.Tx) error {
			proposalsBucket := tx.Bucket(historicProposalsBucket)
			var proposalsPubKeyBucket, err = proposalsBucket.CreateBucket(pubKeyProposals.PubKey)
			if err != nil {
				return errors.Wrapf(err, "Could not create proposals bucket for public key %v", pubKeyProposals.PubKey)
			}
			for _, epochProposals := range pubKeyProposals.Proposals {
				if err := proposalsPubKeyBucket.Put(epochProposals.Epoch, epochProposals.Proposals); err != nil {
					return errors.Wrapf(err, "Could not add epoch proposals for epoch %v", epochProposals.Epoch)
				}
			}

			attestationsBucket := tx.Bucket(historicAttestationsBucket)
			for _, pubKeyAttestations := range allAttestations {
				if string(pubKeyAttestations.PubKey) == string(pubKeyProposals.PubKey) {
					if err := attestationsBucket.Put(pubKeyAttestations.PubKey, pubKeyAttestations.Attestations); err != nil {
						return errors.Wrapf(err, "Could not add public key attestations for public key %v", pubKeyAttestations.PubKey)
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
		if !hasMatchingProposals{
			dirName := hex.EncodeToString(pubKeyAttestations.PubKey)[:12]
			path := filepath.Join(targetDirectory, dirName)
			newStore, err := NewKVStore(path)
			if err != nil {
				return errors.Wrapf(err, "Could not create a validator database in %s", path)
			}
			storesToClose = append(storesToClose, newStore)

			if err := newStore.update(func(tx *bolt.Tx) error {
				attestationsBucket := tx.Bucket(historicAttestationsBucket)
				if err := attestationsBucket.Put(pubKeyAttestations.PubKey, pubKeyAttestations.Attestations); err != nil {
					return errors.Wrapf(err, "Could not add public key attestations for public key %v", pubKeyAttestations.PubKey)
				}

				return nil
			}); err != nil {
				return err
			}
		}
	}

	return nil
}
