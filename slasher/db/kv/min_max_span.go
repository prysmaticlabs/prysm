package kv

import (
	"context"
	"encoding/binary"

	"github.com/boltdb/bolt"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"go.opencensus.io/trace"
)

var highestValidatorIdx uint64

//func saveToDB(db *Store) func(uint64, uint64, interface{}, int64) {
//	// Returning the function here so we can access the DB properly from the OnEvict.
//	return func(validatorIdx uint64, _ uint64, value interface{}, cost int64) {
//		log.Tracef("evicting span map for validator id: %d", validatorIdx)
//		err := db.batch(func(tx *bolt.Tx) error {
//			bucket := tx.Bucket(validatorsMinMaxSpanBucket)
//			key := bytesutil.Bytes8(validatorIdx)
//			val, err := proto.Marshal(value.(*slashpb.EpochSpanMap))
//			if err != nil {
//				return errors.Wrap(err, "failed to marshal span map")
//			}
//			if err := bucket.Put(key, val); err != nil {
//				return errors.Wrapf(err, "failed to delete validator id: %d from min max span bucket", validatorIdx)
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
	empty4bytes := make([]byte, 2)
	return binary.LittleEndian.Uint16(append(x[:2], empty4bytes...))
}

// Bytes2 returns integer x to bytes in little-endian format, x.to_bytes(2, 'big').
func Bytes2(x uint16) []byte {
	bytes := make([]byte, 2)
	binary.LittleEndian.PutUint16(bytes, x)
	return bytes[:2]
}

// ValidatorSpansMap accepts validator index and returns the corresponding spans
// map for slashing detection.
// Returns nil if the span map for this validator index does not exist.
func (db *Store) ValidatorSpansMap(ctx context.Context, validatorIdx uint64) (map[uint64][2]uint16, error) {
	ctx, span := trace.StartSpan(ctx, "SlasherDB.ValidatorSpansMap")
	defer span.End()
	var err error
	var spanMap map[uint64][2]uint16
	err = db.view(func(tx *bolt.Tx) error {
		b := tx.Bucket(validatorsMinMaxSpanBucket)
		valBucket := b.Bucket(bytesutil.Bytes8(validatorIdx))
		if valBucket == nil {
			return errors.New("validator id span maps are not in db yet")
		}
		keysLength := valBucket.Stats().KeyN
		spanMap = make(map[uint64][2]uint16, keysLength)
		return valBucket.ForEach(func(k, v []byte) error {
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

// ValidatorEpochSpans accepts validator index and epoch returns the corresponding spans
// for slashing detection.
// Returns error if the spans for this validator index and epoch does not exist.
func (db *Store) ValidatorEpochSpans(ctx context.Context, validatorIdx uint64, epoch uint64) ([2]uint16, error) {
	ctx, span := trace.StartSpan(ctx, "SlasherDB.ValidatorSpansMap")
	defer span.End()
	var err error
	var spans [2]uint16
	err = db.view(func(tx *bolt.Tx) error {
		b := tx.Bucket(validatorsMinMaxSpanBucket)
		valBucket := b.Bucket(bytesutil.Bytes8(validatorIdx))
		if valBucket == nil {
			return errors.New("validator id span maps are not in db yet")
		}
		key := bytesutil.Bytes8(epoch)
		v := valBucket.Get(key)
		if v == nil {
			return errors.New("missing spans for epoch")
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
	ctx, span := trace.StartSpan(ctx, "SlasherDB.ValidatorSpansMap")
	defer span.End()
	return db.update(func(tx *bolt.Tx) error {
		b := tx.Bucket(validatorsMinMaxSpanBucket)
		valBucket := b.Bucket(bytesutil.Bytes8(validatorIdx))
		if valBucket == nil {
			return errors.New("validator id span maps are not in db yet")
		}
		key := bytesutil.Bytes8(epoch)
		value := marshalMinMaxSpan(ctx, spans)
		return valBucket.Put(key, value)
	})
}

// SaveValidatorSpansMap accepts a validator index and span map and writes it to disk.
func (db *Store) SaveValidatorSpansMap(ctx context.Context, validatorIdx uint64, spanMap map[uint64][2]uint16) error {
	ctx, span := trace.StartSpan(ctx, "SlasherDB.SaveValidatorSpansMap")
	defer span.End()
	return db.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(validatorsMinMaxSpanBucket)
		valBucket, err := bucket.CreateBucketIfNotExists(bytesutil.Bytes8(validatorIdx))
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

// DeleteValidatorSpans deletes a validator span map using a validator index as bucket key.
func (db *Store) DeleteValidatorSpans(ctx context.Context, validatorIdx uint64) error {
	ctx, span := trace.StartSpan(ctx, "SlasherDB.DeleteValidatorSpans")
	defer span.End()
	if db.spanCacheEnabled {
		db.spanCache.Del(validatorIdx)
	}
	return db.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(validatorsMinMaxSpanBucket)
		key := bytesutil.Bytes8(validatorIdx)
		return bucket.DeleteBucket(key)
	})
}
