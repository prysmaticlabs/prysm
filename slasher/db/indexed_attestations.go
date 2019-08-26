package db

import (
	"bytes"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
)

func createIndexedAttestation(enc []byte) (*ethpb.IndexedAttestation, error) {
	protoIdxAtt := &ethpb.IndexedAttestation{}
	err := proto.Unmarshal(enc, protoIdxAtt)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal encoding")
	}
	return protoIdxAtt, nil
}

// IndexedAttestation accepts a epoch and validator index and returns a list of
// indexed attestations.
// Returns nil if the indexed attestation does not exist.
func (db *Store) IndexedAttestation(epoch uint64, validatorID uint64) ([]*ethpb.BeaconBlockHeader, error) {
	var bha []*ethpb.BeaconBlockHeader
	err := db.view(func(tx *bolt.Tx) error {
		c := tx.Bucket(historicBlockHeadersBucket).Cursor()
		prefix := encodeEpochValidatorID(epoch, validatorID)
		for k, v := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = c.Next() {
			bh, err := createBlockHeader(v)
			if err != nil {
				return err
			}
			bha = append(bha, bh)
		}
		return nil
	})
	return bha, err
}

// HasIndexedAttestation accepts an epoch and validator id and returns true if the block header exists.
func (db *Store) HasIndexedAttestation(epoch uint64, validatorID uint64) bool {
	prefix := encodeEpochValidatorID(epoch, validatorID)
	var hasAttestation bool
	// #nosec G104
	_ = db.view(func(tx *bolt.Tx) error {
		c := tx.Bucket(historicIndexedAttestationsBucket).Cursor()
		for k, _ := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, _ = c.Next() {
			hasAttestation = true
			return nil
		}
		hasAttestation = false
		return nil
	})

	return hasAttestation
}

// SaveIndexedAttestation accepts epoch  and indexed attestation and writes it to disk.
func (db *Store) SaveIndexedAttestation(epoch uint64, idxAttestation *ethpb.IndexedAttestation) error {
	indices := append(idxAttestation.CustodyBit_0Indices, idxAttestation.CustodyBit_1Indices...)
	key, err := encodeEpochCustodyBitSig(epoch, indices, idxAttestation.Signature)
	if err != nil {
		return errors.Wrap(err, "failed in encoding")
	}
	enc, err := proto.Marshal(idxAttestation)
	if err != nil {
		return errors.Wrap(err, "failed to marshal")
	}

	err = db.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(historicIndexedAttestationsBucket)
		if err := bucket.Put(key, enc); err != nil {
			return errors.Wrap(err, "failed to include the block header in the historic block header bucket")
		}

		return err
	})

	// prune history to max size every 10th epoch
	if epoch%10 == 0 {
		weakSubjectivityPeriod := uint64(54000)
		err = db.pruneHistory(epoch, weakSubjectivityPeriod)
	}
	return err
}

// DeleteIndexedAttestation deletes a block header using the slot and its root as keys in their respective buckets.
func (db *Store) DeleteIndexedAttestation(epoch uint64, idxAttestation *ethpb.IndexedAttestation) error {

	indices := append(idxAttestation.CustodyBit_0Indices, idxAttestation.CustodyBit_1Indices...)
	key, err := encodeEpochCustodyBitSig(epoch, indices, idxAttestation.Signature)
	if err != nil {
		return errors.Wrap(err, "failed in encoding")
	}
	return db.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(historicIndexedAttestationsBucket)
		if err := bucket.Delete(key); err != nil {
			return errors.Wrap(err, "failed to delete the indexed attestation from historic indexed attestation bucket")
		}
		return bucket.Delete(key)
	})
}

func (db *Store) pruneAttHistory(currentEpoch uint64, historySize uint64) error {
	pruneTill := int64(currentEpoch) - int64(historySize)
	if pruneTill <= 0 {
		return nil
	}
	return db.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(historicIndexedAttestationsBucket)
		c := tx.Bucket(historicIndexedAttestationsBucket).Cursor()
		max := bytesutil.Bytes8(uint64(pruneTill))
		for k, _ := c.First(); k != nil && bytes.Compare(k[:8], max) <= 0; k, _ = c.Next() {
			if err := bucket.Delete(k); err != nil {
				return errors.Wrap(err, "failed to delete the block header from historic indexed attestation bucket")
			}
		}
		return nil
	})
}
