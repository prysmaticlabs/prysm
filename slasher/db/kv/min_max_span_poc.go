package kv

import (
	"context"
	"encoding/binary"
	"fmt"

	"github.com/boltdb/bolt"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"go.opencensus.io/trace"
)

var highestValidatorIdx uint64

//func saveToDB(db *Store) func(uint64, uint64, interface{}, int64) {
//	// Returning the function here so we can access the DB properly from the OnEvict.
//	return func(epoch uint64, _ uint64, value interface{}, cost int64) {
//		log.Tracef("evicting span map for validator id: %d", epoch)
//		err := db.batch(func(tx *bolt.Tx) error {
//			bucket := tx.Bucket(validatorsMinMaxSpanBucket)
//			key := bytesutil.Bytes8(epoch)
//			val, err := proto.Marshal(value.(*slashpb.EpochSpanMap))
//			if err != nil {
//				return errors.Wrap(err, "failed to marshal span map")
//			}
//			if err := bucket.Put(key, val); err != nil {
//				return errors.Wrapf(err, "failed to delete validator id: %d from min max span bucket", epoch)
//			}
//			return err
//		})
//		if err != nil {
//			log.Errorf("failed to save span map to db on cache eviction: %v", err)
//		}
//	}
//}

func unmarshalMinMaxSpan(ctx context.Context, enc []byte) ([2]uint16, error) {
	ctx, span := trace.StartSpan(ctx, "SlasherDB.unmarshalMinMaxSpan")
	defer span.End()
	r := [2]uint16{}
	if len(enc) != 4 {
		return r, errors.New("wrong data length for min max span")
	}
	r[0] = FromBytes2(enc[:2])
	r[1] = FromBytes2(enc[2:4])
	return r, nil
}

func marshalMinMaxSpan(ctx context.Context, spans [2]uint16) []byte {
	ctx, span := trace.StartSpan(ctx, "SlasherDB.marshalMinMaxSpan")
	defer span.End()
	return append(Bytes2(spans[0]), Bytes2(spans[1])...)
}

// FromBytes2 returns an integer which is stored in the little-endian format(4, 'little')
// from a byte array.
func FromBytes2(x []byte) uint16 {
	return binary.LittleEndian.Uint16(x[:2])
}

// Bytes2 returns integer x to bytes in little-endian format, x.to_bytes(2, 'big').
func Bytes2(x uint16) []byte {
	bytes := make([]byte, 2)
	binary.LittleEndian.PutUint16(bytes, x)
	return bytes[:2]
}

// EpochSpansMap accepts epoch and returns the corresponding spans map epoch=>spans
// for slashing detection.
// Returns nil if the span map for this validator index does not exist.
func (db *Store) EpochSpansMap(ctx context.Context, epoch uint64) (map[uint64][2]uint16, error) {
	ctx, span := trace.StartSpan(ctx, "SlasherDB.EpochSpansMap")
	defer span.End()
	var err error
	var spanMap map[uint64][2]uint16
	err = db.view(func(tx *bolt.Tx) error {
		b := tx.Bucket(validatorsMinMaxSpanBucket)
		epochBucket := b.Bucket(bytesutil.Bytes8(epoch))
		if epochBucket == nil {
			return nil
		}
		keysLength := epochBucket.Stats().KeyN
		spanMap = make(map[uint64][2]uint16, keysLength)
		return epochBucket.ForEach(func(k, v []byte) error {
			key := bytesutil.FromBytes8(k)
			value, err := unmarshalMinMaxSpan(ctx, v)
			if err != nil {
				return err
			}
			spanMap[key] = value
			return nil
		})
	})
	if spanMap == nil {
		spanMap = make(map[uint64][2]uint16)
	}
	return spanMap, err
}

// EpochSpanByValidatorIndex accepts validator index and epoch returns the corresponding spans
// for slashing detection.
// Returns error if the spans for this validator index and epoch does not exist.
func (db *Store) EpochSpanByValidatorIndex(ctx context.Context, validatorIdx uint64, epoch uint64) ([2]uint16, error) {
	ctx, span := trace.StartSpan(ctx, "SlasherDB.EpochSpansMap")
	defer span.End()
	var err error
	if db.spanCacheEnabled {
		v, ok := db.spanCache.Get(validatorIdx)
		spanMap := make(map[uint64][2]uint16)
		if ok {
			spanMap = v.(map[uint64][2]uint16)
			spans, ok := spanMap[epoch]
			if ok {
				return spans, nil
			}
		}
	}
	var spans [2]uint16
	err = db.view(func(tx *bolt.Tx) error {
		b := tx.Bucket(validatorsMinMaxSpanBucket)
		epochBucket := b.Bucket(bytesutil.Bytes8(epoch))
		if epochBucket == nil {
			return nil
		}
		key := bytesutil.Bytes8(validatorIdx)
		v := epochBucket.Get(key)
		if v == nil {
			return nil
		}
		value, err := unmarshalMinMaxSpan(ctx, v)
		if err != nil {
			return err
		}
		spans = value
		return nil
	})
	return spans, err
}

// SaveValidatorEpochSpans accepts validator index epoch and spans returns.
// Returns error if the spans for this validator index and epoch does not exist.
func (db *Store) SaveValidatorEpochSpans(ctx context.Context, validatorIdx uint64, epoch uint64, spans [2]uint16) error {
	ctx, span := trace.StartSpan(ctx, "SlasherDB.EpochSpansMap")
	defer span.End()
	defer span.End()
	if db.spanCacheEnabled {
		if validatorIdx > highestValidatorIdx {
			highestValidatorIdx = validatorIdx
		}
		v, ok := db.spanCache.Get(epoch)
		spanMap := make(map[uint64][2]uint16)
		if ok {
			spanMap = v.(map[uint64][2]uint16)
		}
		spanMap[validatorIdx] = spans
		saved := db.spanCache.Set(epoch, spanMap, 1)
		if !saved {
			return fmt.Errorf("failed to save span map to cache")
		}
		return nil
	}
	return db.update(func(tx *bolt.Tx) error {
		b := tx.Bucket(validatorsMinMaxSpanBucket)
		epochBucket, err := b.CreateBucketIfNotExists(bytesutil.Bytes8(epoch))
		if err != nil {
			return err
		}
		key := bytesutil.Bytes8(validatorIdx)
		value := marshalMinMaxSpan(ctx, spans)
		return epochBucket.Put(key, value)
	})
}

// SaveEpochSpansMap accepts a epoch and span map epoch=>spans and writes it to disk.
func (db *Store) SaveEpochSpansMap(ctx context.Context, epoch uint64, spanMap map[uint64][2]uint16) error {
	ctx, span := trace.StartSpan(ctx, "SlasherDB.SaveEpochSpansMap")
	defer span.End()
	return db.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(validatorsMinMaxSpanBucket)
		valBucket, err := bucket.CreateBucketIfNotExists(bytesutil.Bytes8(epoch))
		if err != nil {
			return err
		}
		for k, v := range spanMap {
			err = valBucket.Put(bytesutil.Bytes8(k), marshalMinMaxSpan(ctx, v))
			if err != nil {
				return err
			}
		}
		return nil
	})
}

// TODO: bring back caching
//// SaveCachedSpansMaps saves all span map from cache to disk
//// if no span maps are in db or cache is disabled it returns nil.
//func (db *Store) SaveCachedSpansMaps(ctx context.Context) error {
//	ctx, span := trace.StartSpan(ctx, "SlasherDB.SaveCachedSpansMaps")
//	defer span.End()
//	if db.spanCacheEnabled {
//		err := db.update(func(tx *bolt.Tx) error {
//			bucket := tx.Bucket(validatorsMinMaxSpanBucket)
//			for i := uint64(0); i <= highestValidatorIdx; i++ {
//				spanMap, ok := db.spanCache.Get(i)
//				if ok {
//					key := bytesutil.Bytes8(i)
//					val, err := proto.Marshal(spanMap.(*slashpb.EpochSpanMap))
//					if err != nil {
//						return errors.Wrap(err, "failed to marshal span map")
//					}
//					if err := bucket.Put(key, val); err != nil {
//						return errors.Wrapf(err, "failed to save validator id: %d from validators min max span cache", i)
//					}
//				}
//			}
//			return nil
//		})
//		return err
//	}
//	return nil
//}

// DeleteEpochSpans deletes a epochs validators span map using a epoch index as bucket key.
func (db *Store) DeleteEpochSpans(ctx context.Context, epoch uint64) error {
	ctx, span := trace.StartSpan(ctx, "SlasherDB.DeleteEpochSpans")
	defer span.End()
	if db.spanCacheEnabled {
		db.spanCache.Del(epoch)
	}
	return db.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(validatorsMinMaxSpanBucket)
		key := bytesutil.Bytes8(epoch)
		return bucket.DeleteBucket(key)
	})
}

// DeleteValidatorSpanByEpoch deletes a validator span for a certain epoch
// using a validator index as bucket key.
func (db *Store) DeleteValidatorSpanByEpoch(ctx context.Context, validatorIdx uint64, epoch uint64) error {
	ctx, span := trace.StartSpan(ctx, "SlasherDB.DeleteEpochSpans")
	defer span.End()
	if db.spanCacheEnabled {
		v, ok := db.spanCache.Get(epoch)
		spanMap := make(map[uint64][2]uint16)
		if ok {
			spanMap = v.(map[uint64][2]uint16)
		}
		delete(spanMap, validatorIdx)
		saved := db.spanCache.Set(epoch, spanMap, 1)
		if !saved {
			return fmt.Errorf("failed to save span map to cache")
		}
		return nil
	}
	return db.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(validatorsMinMaxSpanBucket)
		e := bytesutil.Bytes8(epoch)
		epochBucket := bucket.Bucket(e)
		v := bytesutil.Bytes8(validatorIdx)
		return epochBucket.Delete(v)
	})
}
