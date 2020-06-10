package kv

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	log "github.com/sirupsen/logrus"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

// EpochSpans accepts epoch and returns the corresponding spans byte array
// for slashing detection.
// Returns span byte array, and error in case of db error.
// returns empty byte array if no entry for this epoch exists in db.
func (db *Store) EpochSpans(ctx context.Context, epoch uint64) (EpochStore, error) {
	ctx, span := trace.StartSpan(ctx, "slasherDB.EpochSpans")
	defer span.End()

	var err error
	var spans []byte
	err = db.view(func(tx *bolt.Tx) error {
		b := tx.Bucket(validatorsMinMaxSpanBucketNew)
		if b == nil {
			return nil
		}
		spans = b.Get(bytesutil.Bytes8(epoch))
		return nil
	})
	if spans == nil {
		spans = []byte{}
	}
	return spans, err
}

// SaveEpochSpans accepts a epoch and span byte array and writes it to disk.
func (db *Store) SaveEpochSpans(ctx context.Context, epoch uint64, es EpochStore) error {
	ctx, span := trace.StartSpan(ctx, "slasherDB.SaveEpochSpans")
	defer span.End()

	if len(es)%spannerEncodedLength != 0 {
		return ErrWrongSize
	}
	return db.update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(validatorsMinMaxSpanBucketNew)
		if err != nil {
			return err
		}
		return b.Put(bytesutil.Bytes8(epoch), es)
	})
}

// This function defines a function which triggers upon a span map being
// evicted from the cache. It allows us to persist the span map by the epoch value
// to the database itself in the validatorsMinMaxSpanBucket.
func persistEpochSpansOnEviction(db *Store) func(key interface{}, value interface{}) {
	// We use a closure here so we can access the database itself
	// on the eviction of a span map from the cache. The function has the signature
	// required by the ristretto cache OnEvict method.
	// See https://godoc.org/github.com/dgraph-io/ristretto#Config.
	return func(key interface{}, value interface{}) {
		log.Tracef("Evicting flat span map for epoch: %d", key)
		err := db.update(func(tx *bolt.Tx) error {
			epoch, keyOK := key.(uint64)
			spans, valueOK := value.(EpochStore)
			if !keyOK || !valueOK {
				return errors.New("could not cast key and value into needed types")
			}
			bucket, err := tx.CreateBucketIfNotExists(validatorsMinMaxSpanBucketNew)
			if err != nil {
				return err
			}
			if err := bucket.Put(bytesutil.Bytes8(epoch), spans); err != nil {
				return err
			}
			epochSpansCacheEvictions.Inc()
			return nil
		})
		if err != nil {
			log.Errorf("Failed to save span map to db on cache eviction: %v", err)
		}
	}
}
