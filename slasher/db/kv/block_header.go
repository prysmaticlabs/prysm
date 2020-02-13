package kv

import (
	"bytes"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func unmarshalBlockHeader(enc []byte) (*ethpb.SignedBeaconBlockHeader, error) {
	protoBlockHeader := &ethpb.SignedBeaconBlockHeader{}
	err := proto.Unmarshal(enc, protoBlockHeader)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal encoding")
	}
	return protoBlockHeader, nil
}

// BlockHeaders accepts an epoch and validator id and returns the corresponding block header array.
// Returns nil if the block header for those values does not exist.
func (db *Store) BlockHeaders(epoch uint64, validatorID uint64) ([]*ethpb.SignedBeaconBlockHeader, error) {
	var blockHeaders []*ethpb.SignedBeaconBlockHeader
	err := db.view(func(tx *bolt.Tx) error {
		c := tx.Bucket(historicBlockHeadersBucket).Cursor()
		prefix := encodeEpochValidatorID(epoch, validatorID)
		for k, v := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = c.Next() {
			bh, err := unmarshalBlockHeader(v)
			if err != nil {
				return err
			}
			blockHeaders = append(blockHeaders, bh)
		}
		return nil
	})
	return blockHeaders, err
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
func (db *Store) SaveBlockHeader(epoch uint64, validatorID uint64, blockHeader *ethpb.SignedBeaconBlockHeader) error {
	key := encodeEpochValidatorIDSig(epoch, validatorID, blockHeader.Signature)
	enc, err := proto.Marshal(blockHeader)
	if err != nil {
		return errors.Wrap(err, "failed to encode block")
	}

	err = db.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(historicBlockHeadersBucket)
		if err := bucket.Put(key, enc); err != nil {
			return errors.Wrap(err, "failed to include block header in the historical bucket")
		}

		return err
	})

	// Prune block header history every 10th epoch.
	if epoch%params.BeaconConfig().PruneSlasherStoragePeriod == 0 {
		err = db.PruneBlockHistory(epoch, params.BeaconConfig().WeakSubjectivityPeriod)
	}
	return err
}

// DeleteBlockHeader deletes a block header using the epoch and validator id.
func (db *Store) DeleteBlockHeader(epoch uint64, validatorID uint64, blockHeader *ethpb.SignedBeaconBlockHeader) error {
	key := encodeEpochValidatorIDSig(epoch, validatorID, blockHeader.Signature)

	return db.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(historicBlockHeadersBucket)
		if err := bucket.Delete(key); err != nil {
			return errors.Wrap(err, "failed to delete the block header from historical bucket")
		}
		return bucket.Delete(key)
	})
}

// PruneBlockHistory removes all blocks from the DB older than the pruning epoch age.
func (db *Store) PruneBlockHistory(currentEpoch uint64, pruningEpochAge uint64) error {
	pruneTill := int64(currentEpoch) - int64(pruningEpochAge)
	if pruneTill <= 0 {
		return nil
	}
	return db.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(historicBlockHeadersBucket)
		c := tx.Bucket(historicBlockHeadersBucket).Cursor()
		for k, _ := c.First(); k != nil && bytesutil.FromBytes8(k[:8]) <= uint64(pruneTill); k, _ = c.Next() {
			if err := bucket.Delete(k); err != nil {
				return errors.Wrap(err, "failed to delete the block header from historical bucket")
			}
		}
		return nil
	})
}
