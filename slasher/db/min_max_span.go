package db

import (
	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
)

func createEpochSpanMap(enc []byte) (*ethpb.EpochSpanMap, error) {
	epochSpanMap := &ethpb.EpochSpanMap{}
	err := proto.Unmarshal(enc, epochSpanMap)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal encoding")
	}
	return epochSpanMap, nil
}

// ValidatorSpansMap accepts validator id and returns the corresponding spans
// map for slashing detection and update.
// Returns nil if the span map for this validator id does not exist.
func (db *Store) ValidatorSpansMap(validatorID uint64) (*ethpb.EpochSpanMap, error) {
	var sm *ethpb.EpochSpanMap
	err := db.view(func(tx *bolt.Tx) error {
		b := tx.Bucket(validatorsMinMaxSpanBucket)
		enc := b.Get(bytesutil.Bytes4(validatorID))
		var err error
		sm, err = createEpochSpanMap(enc)
		if err != nil {
			return err
		}
		return nil
	})
	return sm, err
}

// SaveValidatorSpansMap accepts validator id and span map and writes it to disk.
func (db *Store) SaveValidatorSpansMap(validatorID uint64, spanMap *ethpb.EpochSpanMap) error {
	err := db.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(validatorsMinMaxSpanBucket)
		key := bytesutil.Bytes4(validatorID)
		val, err := proto.Marshal(spanMap)
		if err != nil {
			return errors.Wrap(err, "failed to marshal span map")
		}
		if err := bucket.Put(key, val); err != nil {
			return errors.Wrapf(err, "failed to include the span map for validator id: %v in the validators min max span bucket", validatorID)
		}
		return err
	})
	return err
}

// DeleteValidatorSpanMap deletes a validator span map using the validator id as bucket key.
func (db *Store) DeleteValidatorSpanMap(validatorID uint64) error {
	return db.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(validatorsMinMaxSpanBucket)
		key := bytesutil.Bytes4(validatorID)
		enc := bucket.Get(key)
		if enc == nil {
			return nil
		}
		if err := bucket.Delete(key); err != nil {
			tx.Rollback()
			return errors.Wrapf(err, "failed to delete the span map for validator id: %v from validators min max span bucket", validatorID)
		}
		return nil
	})
}
