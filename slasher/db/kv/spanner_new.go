package kv

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/slasher/detection/attestations/types"
	log "github.com/sirupsen/logrus"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

// This function defines a function which triggers upon a span map being
// evicted from the cache. It allows us to persist the span map by the epoch value
// to the database itself in the validatorsMinMaxSpanBucket.
func persistFlatSpanMapsOnEviction(db *Store) func(key interface{}, value interface{}) {
	// We use a closure here so we can access the database itself
	// on the eviction of a span map from the cache. The function has the signature
	// required by the ristretto cache OnEvict method.
	// See https://godoc.org/github.com/dgraph-io/ristretto#Config.
	return func(key interface{}, value interface{}) {
		log.Tracef("Evicting flat span map for epoch: %d", key)
		err := db.update(func(tx *bolt.Tx) error {
			epoch, keyOK := key.(uint64)
			epochStore, valueOK := value.(*types.EpochStore)
			if !keyOK || !valueOK {
				return errors.New("could not cast key and value into needed types")
			}

			bucket := tx.Bucket(validatorsMinMaxSpanBucketNew)
			if err := bucket.Put(bytesutil.Bytes8(epoch), epochStore.Bytes()); err != nil {
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

// EpochSpans accepts epoch and returns the corresponding spans byte array
// for slashing detection.
// Returns span byte array, and error in case of db error.
// returns empty byte array if no entry for this epoch exists in db.
func (db *Store) EpochSpans(ctx context.Context, epoch uint64, fromCache bool) (*types.EpochStore, error) {
	ctx, span := trace.StartSpan(ctx, "slasherDB.EpochSpans")
	defer span.End()

	// Get from the cache if it exists or is requested, if not, go to DB.
	if fromCache && db.flatSpanCache.Has(epoch) || db.flatSpanCache.Has(epoch) {
		spans, _ := db.flatSpanCache.Get(epoch)
		return spans, nil
	}

	var copiedSpans []byte
	err := db.view(func(tx *bolt.Tx) error {
		b := tx.Bucket(validatorsMinMaxSpanBucketNew)
		if b == nil {
			return nil
		}
		spans := b.Get(bytesutil.Bytes8(epoch))
		copiedSpans = make([]byte, len(spans))
		copy(copiedSpans, spans)
		return nil
	})
	if err != nil {
		return &types.EpochStore{}, err
	}
	if copiedSpans == nil {
		copiedSpans = []byte{}
	}
	return types.NewEpochStore(copiedSpans)
}

// SaveEpochSpans accepts a epoch and span byte array and writes it to disk.
func (db *Store) SaveEpochSpans(ctx context.Context, epoch uint64, es *types.EpochStore, toCache bool) error {
	ctx, span := trace.StartSpan(ctx, "slasherDB.SaveEpochSpans")
	defer span.End()

	if len(es.Bytes())%int(types.SpannerEncodedLength) != 0 {
		return types.ErrWrongSize
	}

	// Saving to the cache if it exists so cache and DB never conflict.
	if toCache || db.flatSpanCache.Has(epoch) {
		db.flatSpanCache.Set(epoch, es)
	}
	if toCache {
		return nil
	}

	return db.update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(validatorsMinMaxSpanBucketNew)
		if err != nil {
			return err
		}
		return b.Put(bytesutil.Bytes8(epoch), es.Bytes())
	})
}

// CacheLength returns the number of cached items.
func (db *Store) CacheLength(ctx context.Context) int {
	ctx, span := trace.StartSpan(ctx, "slasherDB.CacheLength")
	defer span.End()
	len := db.flatSpanCache.Length()
	log.Debugf("Span cache length %d", len)
	return len
}
