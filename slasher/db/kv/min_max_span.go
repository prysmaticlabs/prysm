package kv

import (
	"context"
	"fmt"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	slashpb "github.com/prysmaticlabs/prysm/proto/slashing"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	log "github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

var highestValidatorIdx uint64

func saveToDB(db *Store) func(uint64, uint64, interface{}, int64) {
	// Returning the function here so we can access the DB properly from the OnEvict.
	return func(validatorIdx uint64, _ uint64, value interface{}, cost int64) {
		log.Tracef("evicting span map for validator id: %d", validatorIdx)
		err := db.batch(func(tx *bolt.Tx) error {
			bucket := tx.Bucket(validatorsMinMaxSpanBucket)
			key := bytesutil.Bytes4(validatorIdx)
			val, err := proto.Marshal(value.(*slashpb.EpochSpanMap))
			if err != nil {
				return errors.Wrap(err, "failed to marshal span map")
			}
			if err := bucket.Put(key, val); err != nil {
				return errors.Wrapf(err, "failed to delete validator id: %d from min max span bucket", validatorIdx)
			}
			return err
		})
		if err != nil {
			log.Errorf("failed to save span map to db on cache eviction: %v", err)
		}
	}
}

func unmarshalEpochSpanMap(ctx context.Context, enc []byte) (*slashpb.EpochSpanMap, error) {
	ctx, span := trace.StartSpan(ctx, "SlasherDB.unmarshalEpochSpanMap")
	defer span.End()
	epochSpanMap := &slashpb.EpochSpanMap{}
	err := proto.Unmarshal(enc, epochSpanMap)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal epoch span map")
	}
	return epochSpanMap, nil
}

// ValidatorSpansMap accepts validator index and returns the corresponding spans
// map for slashing detection.
// Returns nil if the span map for this validator index does not exist.
func (db *Store) ValidatorSpansMap(ctx context.Context, validatorIdx uint64) (*slashpb.EpochSpanMap, error) {
	ctx, span := trace.StartSpan(ctx, "SlasherDB.ValidatorSpansMap")
	defer span.End()
	var err error
	var spanMap *slashpb.EpochSpanMap
	if db.spanCacheEnabled {
		spanMap, ok := db.spanCache.Get(validatorIdx)
		if ok {
			return spanMap.(*slashpb.EpochSpanMap), nil
		}
	}

	err = db.view(func(tx *bolt.Tx) error {
		b := tx.Bucket(validatorsMinMaxSpanBucket)
		enc := b.Get(bytesutil.Bytes4(validatorIdx))
		spanMap, err = unmarshalEpochSpanMap(ctx, enc)
		if err != nil {
			return err
		}
		return nil
	})
	if spanMap.EpochSpanMap == nil {
		spanMap.EpochSpanMap = make(map[uint64]*slashpb.MinMaxEpochSpan)
	}
	return spanMap, err
}

// SaveValidatorSpansMap accepts a validator index and span map and writes it to disk.
func (db *Store) SaveValidatorSpansMap(ctx context.Context, validatorIdx uint64, spanMap *slashpb.EpochSpanMap) error {
	ctx, span := trace.StartSpan(ctx, "SlasherDB.SaveValidatorSpansMap")
	defer span.End()
	if db.spanCacheEnabled {
		if validatorIdx > highestValidatorIdx {
			highestValidatorIdx = validatorIdx
		}
		saved := db.spanCache.Set(validatorIdx, spanMap, 1)
		if !saved {
			return fmt.Errorf("failed to save span map to cache")
		}
		return nil
	}
	err := db.batch(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(validatorsMinMaxSpanBucket)
		key := bytesutil.Bytes4(validatorIdx)
		val, err := proto.Marshal(spanMap)
		if err != nil {
			return errors.Wrap(err, "failed to marshal span map")
		}
		if err := bucket.Put(key, val); err != nil {
			return errors.Wrapf(err, "failed to delete validator id: %d from min max span bucket", validatorIdx)
		}
		return nil
	})
	return err
}

// SaveCachedSpansMaps saves all span map from cache to disk
// if no span maps are in db or cache is disabled it returns nil.
func (db *Store) SaveCachedSpansMaps(ctx context.Context) error {
	ctx, span := trace.StartSpan(ctx, "SlasherDB.SaveCachedSpansMaps")
	defer span.End()
	if db.spanCacheEnabled {
		err := db.update(func(tx *bolt.Tx) error {
			bucket := tx.Bucket(validatorsMinMaxSpanBucket)
			for i := uint64(0); i <= highestValidatorIdx; i++ {
				spanMap, ok := db.spanCache.Get(i)
				if ok {
					key := bytesutil.Bytes4(i)
					val, err := proto.Marshal(spanMap.(*slashpb.EpochSpanMap))
					if err != nil {
						return errors.Wrap(err, "failed to marshal span map")
					}
					if err := bucket.Put(key, val); err != nil {
						return errors.Wrapf(err, "failed to save validator id: %d from validators min max span cache", i)
					}
				}
			}
			return nil
		})
		return err
	}
	return nil
}

// DeleteValidatorSpanMap deletes a validator span map using a validator index as bucket key.
func (db *Store) DeleteValidatorSpanMap(ctx context.Context, validatorIdx uint64) error {
	ctx, span := trace.StartSpan(ctx, "SlasherDB.DeleteValidatorSpanMap")
	defer span.End()
	if db.spanCacheEnabled {
		db.spanCache.Del(validatorIdx)
	}
	return db.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(validatorsMinMaxSpanBucket)
		key := bytesutil.Bytes4(validatorIdx)
		enc := bucket.Get(key)
		if enc == nil {
			return nil
		}
		if err := bucket.Delete(key); err != nil {
			tx.Rollback()
			return errors.Wrapf(err, "failed to delete the span map for validator idx: %v from min max span bucket", validatorIdx)
		}
		return nil
	})
}
