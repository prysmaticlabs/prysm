package db

import (
	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	"github.com/karlseguin/ccache"
	"github.com/pkg/errors"
	slashpb "github.com/prysmaticlabs/prysm/proto/slashing"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/slasher/flags"
)

const maxCacheSize = 10000

var spanCache *ccache.Cache

func init() {
	var err error
	spanCache = ccache.New(ccache.Configure())
	if err != nil {
		errors.Wrap(err, "failed to start span cache")
		panic(err)
	}
}
func createEpochSpanMap(enc []byte) (*slashpb.EpochSpanMap, error) {
	epochSpanMap := &slashpb.EpochSpanMap{}
	err := proto.Unmarshal(enc, epochSpanMap)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal encoding")
	}
	return epochSpanMap, nil
}

// ValidatorSpansMap accepts validator index and returns the corresponding spans
// map for slashing detection.
// Returns nil if the span map for this validator index does not exist.
func (db *Store) ValidatorSpansMap(validatorIdx uint64) (*slashpb.EpochSpanMap, error) {
	var sm *slashpb.EpochSpanMap
	if db.ctx.GlobalBool(flags.UseSpanCacheFlag.Name) {
		spanCache.
		sm, ok := spanCache.Get(validatorIdx)
		if ok {
			return sm.(*slashpb.EpochSpanMap), nil
		}
	}
	err := db.view(func(tx *bolt.Tx) error {
		b := tx.Bucket(validatorsMinMaxSpanBucket)
		enc := b.Get(bytesutil.Bytes4(validatorIdx))
		var err error
		sm, err = createEpochSpanMap(enc)
		if err != nil {
			return err
		}
		return nil
	})
	if sm.EpochSpanMap == nil {
		sm.EpochSpanMap = make(map[uint64]*slashpb.MinMaxEpochSpan)
	}
	return sm, err
}

// SaveValidatorSpansMap accepts a validator index and span map and writes it to disk.
func (db *Store) SaveValidatorSpansMap(validatorIdx uint64, spanMap *slashpb.EpochSpanMap) error {
	evicted := spanCache.Add(validatorIdx, spanMap)

	err := db.batch(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(validatorsMinMaxSpanBucket)
		key := bytesutil.Bytes4(validatorIdx)
		val, err := proto.Marshal(spanMap)
		if err != nil {
			return errors.Wrap(err, "failed to marshal span map")
		}
		if err := bucket.Put(key, val); err != nil {
			return errors.Wrapf(err, "failed to delete validator id: %d from validators min max span bucket", validatorIdx)
		}
		return err
	})
	return err
}

// DeleteValidatorSpanMap deletes a validator span map using a validator index as bucket key.
func (db *Store) DeleteValidatorSpanMap(validatorIdx uint64) error {
	return db.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(validatorsMinMaxSpanBucket)
		key := bytesutil.Bytes4(validatorIdx)
		enc := bucket.Get(key)
		if enc == nil {
			return nil
		}
		if err := bucket.Delete(key); err != nil {
			tx.Rollback()
			return errors.Wrapf(err, "failed to delete the span map for validator idx: %v from validators min max span bucket", validatorIdx)
		}
		return nil
	})
}
