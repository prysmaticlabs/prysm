package kv

import (
	"bytes"
	"context"
	"encoding/hex"
	"path/filepath"

	"github.com/pkg/errors"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

var errFailedToCloseSource = errors.New("failed to close the source")
var errFailedToCloseManySources = errors.New("failed to close one or more stores")

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
	return createMergeTargetStore(targetDirectory, allProposals, allAttestations)
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
	return createSplitTargetStores(targetDirectory, allProposals, allAttestations)
}

func getPubKeyProposals(pubKey []byte, proposalsBucket *bolt.Bucket) (*pubKeyProposals, error) {
	pubKeyProposals := pubKeyProposals{
		PubKey:    pubKey,
		Proposals: []epochProposals{},
	}

	pubKeyBucket := proposalsBucket.Bucket(pubKey)
	if pubKeyBucket == nil {
		return &pubKeyProposals, nil
	}

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
		return nil, errors.Wrapf(err, "could not retrieve proposals for public key %x", pubKey[:12])
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
			if err != nil {
				err = errors.Wrap(err, errFailedToCloseSource.Error())
			} else {
				err = errors.Wrap(deferErr, errFailedToCloseSource.Error())
			}

		}
	}()
	if err != nil {
		return errors.Wrapf(err, "could not initialize a new database in %s", targetDirectory)
	}

	err = newStore.update(func(tx *bolt.Tx) error {
		allProposalsBucket := tx.Bucket(historicProposalsBucket)
		for _, pubKeyProposals := range allProposals {
			proposalsBucket, err := createProposalsBucket(allProposalsBucket, pubKeyProposals.PubKey)
			if err != nil {
				return err
			}
			if err := addEpochProposals(proposalsBucket, pubKeyProposals.Proposals); err != nil {
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
	defer func() {
		failedToClose := false
		for _, store := range storesToClose {
			if deferErr := store.Close(); deferErr != nil {
				failedToClose = true
			}
		}
		if failedToClose {
			if err != nil {
				err = errors.Wrapf(err, errFailedToCloseManySources.Error())
			} else {
				err = errFailedToCloseManySources
			}
		}
	}()

	for _, pubKeyProposals := range allProposals {
		dirName := hex.EncodeToString(pubKeyProposals.PubKey)[:12]
		path := filepath.Join(targetDirectory, dirName)
		newStore, err := NewKVStore(path, [][48]byte{})
		if err != nil {
			return errors.Wrapf(err, "could not create a validator database in %s", path)
		}
		storesToClose = append(storesToClose, newStore)

		if err := newStore.update(func(tx *bolt.Tx) error {
			allProposalsBucket := tx.Bucket(historicProposalsBucket)
			proposalsBucket, err := createProposalsBucket(allProposalsBucket, pubKeyProposals.PubKey)
			if err != nil {
				return err
			}
			if err := addEpochProposals(proposalsBucket, pubKeyProposals.Proposals); err != nil {
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
				return errors.Wrapf(err, "could not create a validator database in %s", path)
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
		// Storing keys upfront will allow using several short transactions (one for every key)
		// instead of one long-running transaction for all keys.
		var allKeys [][]byte

		if err := store.db.View(func(tx *bolt.Tx) error {
			proposalsBucket := tx.Bucket(historicProposalsBucket)
			if err := proposalsBucket.ForEach(func(pubKey, _ []byte) error {
				pubKeyCopy := make([]byte, len(pubKey))
				copy(pubKeyCopy, pubKey)
				allKeys = append(allKeys, pubKeyCopy)
				return nil
			}); err != nil {
				return errors.Wrapf(err, "could not retrieve proposals for source in %s", store.databasePath)
			}

			attestationsBucket := tx.Bucket(historicAttestationsBucket)
			if err := attestationsBucket.ForEach(func(pubKey, _ []byte) error {
				pubKeyCopy := make([]byte, len(pubKey))
				copy(pubKeyCopy, pubKey)
				allKeys = append(allKeys, pubKeyCopy)
				return nil
			}); err != nil {
				return errors.Wrapf(err, "could not retrieve attestations for source in %s", store.databasePath)
			}

			return nil
		}); err != nil {
			return nil, nil, err
		}

		allKeys = removeDuplicateKeys(allKeys)

		for _, pubKey := range allKeys {
			if err := store.db.View(func(tx *bolt.Tx) error {
				proposalsBucket := tx.Bucket(historicProposalsBucket)
				pubKeyProposals, err := getPubKeyProposals(pubKey, proposalsBucket)
				if err != nil {
					return err
				}
				allProposals = append(allProposals, *pubKeyProposals)

				attestationsBucket := tx.Bucket(historicAttestationsBucket)
				v := attestationsBucket.Get(pubKey)
				if v != nil {
					attestations := pubKeyAttestations{
						PubKey:       pubKey,
						Attestations: make([]byte, len(v)),
					}
					copy(attestations.Attestations, v)
					allAttestations = append(allAttestations, attestations)
				}

				return nil
			}); err != nil {
				return nil, nil, errors.Wrapf(err, "could not retrieve data for public key %x", pubKey[:12])
			}
		}
	}

	return allProposals, allAttestations, nil
}

func createProposalsBucket(topLevelBucket *bolt.Bucket, pubKey []byte) (*bolt.Bucket, error) {
	var bucket, err = topLevelBucket.CreateBucket(pubKey)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create proposals bucket for public key %x", pubKey[:12])
	}
	return bucket, nil
}

func addEpochProposals(bucket *bolt.Bucket, proposals []epochProposals) error {
	for _, singleProposal := range proposals {
		if err := bucket.Put(singleProposal.Epoch, singleProposal.Proposals); err != nil {
			return errors.Wrapf(err, "could not add epoch proposals for epoch %v", singleProposal.Epoch)
		}
	}
	return nil
}

func addAttestations(bucket *bolt.Bucket, attestations pubKeyAttestations) error {
	if err := bucket.Put(attestations.PubKey, attestations.Attestations); err != nil {
		return errors.Wrapf(
			err,
			"could not add public key attestations for public key %x",
			attestations.PubKey[:12])
	}
	return nil
}

func removeDuplicateKeys(keys [][]byte) [][]byte {
	last := 0

next:
	for _, k1 := range keys {
		for _, k2 := range keys[:last] {
			if bytes.Equal(k1, k2) {
				continue next
			}
		}
		keys[last] = k1
		last++
	}

	return keys[:last]
}
