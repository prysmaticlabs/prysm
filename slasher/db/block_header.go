package db

import (
	"bytes"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func createBlockHeader(enc []byte) (*ethpb.BeaconBlockHeader, error) {
	protoBlockHeader := &ethpb.BeaconBlockHeader{}
	err := proto.Unmarshal(enc, protoBlockHeader)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal encoding")
	}
	return protoBlockHeader, nil
}

// BlockHeader accepts an epoch and validator id and returns the corresponding block header array.
// Returns nil if the block header for those values does not exist.
func (db *Store) BlockHeader(epoch uint64, validatorID uint64) ([]*ethpb.BeaconBlockHeader, error) {
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

// HasBlockHeader accepts an epoch and validator id and returns true if the block header exists.
func (db *Store) HasBlockHeader(epoch uint64, validatorID uint64) bool {
	prefix := encodeEpochValidatorID(epoch, validatorID)
	var hasBlockHeader bool
	// #nosec G104
	_ = db.view(func(tx *bolt.Tx) error {
		c := tx.Bucket(historicBlockHeadersBucket).Cursor()
		for k, _ := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, _ = c.Next() {
			hasBlockHeader = true
			return nil
		}
		hasBlockHeader = false
		return nil
	})

	return hasBlockHeader
}

// SaveBlockHeader accepts a block header and writes it to disk.
func (db *Store) SaveBlockHeader(epoch uint64, validatorID uint64, blockHeader *ethpb.BeaconBlockHeader) error {
	key := encodeEpochValidatorIDSig(epoch, validatorID, blockHeader.Signature)
	enc, err := proto.Marshal(blockHeader)
	if err != nil {
		return errors.Wrap(err, "failed to encode block")
	}

	err = db.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(historicBlockHeadersBucket)
		if err := bucket.Put(key, enc); err != nil {
			return errors.Wrap(err, "failed to include the block header in the historic block header bucket")
		}

		return err
	})

	// prune history to max size every 10th epoch
	if epoch%params.BeaconConfig().PruneSlasherStoragePeriod == 0 {
		err = db.pruneHistory(epoch, params.BeaconConfig().WeakSubjectivityPeriod)
	}
	return err
}

// DeleteBlockHeader deletes a block header using the epoch and validator id.
func (db *Store) DeleteBlockHeader(epoch uint64, validatorID uint64, blockHeader *ethpb.BeaconBlockHeader) error {

	key := encodeEpochValidatorIDSig(epoch, validatorID, blockHeader.Signature)

	return db.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(historicBlockHeadersBucket)
		if err := bucket.Delete(key); err != nil {
			return errors.Wrap(err, "failed to delete the block header from historic block header bucket")
		}
		return bucket.Delete(key)
	})
}

func (db *Store) pruneHistory(currentEpoch uint64, historySize uint64) error {
	pruneTill := int64(currentEpoch) - int64(historySize)
	if pruneTill <= 0 {
		return nil
	}
	return db.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(historicBlockHeadersBucket)
		c := tx.Bucket(historicBlockHeadersBucket).Cursor()
		for k, _ := c.First(); k != nil && bytesutil.FromBytes8(k[:8]) <= uint64(pruneTill); k, _ = c.Next() {
			if err := bucket.Delete(k); err != nil {
				return errors.Wrap(err, "failed to delete the block header from historic block header bucket")
			}
		}
		return nil
	})
}
