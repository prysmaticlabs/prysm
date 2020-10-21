package kv

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/slasher/detection/attestations/types"
	log "github.com/sirupsen/logrus"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

// Tracks the highest and lowest observed epochs from the validator span maps
// used for attester slashing detection. This value is purely used
// as a cache key and only needs to be maintained in memory.
var highestObservedEpoch uint64
var lowestObservedEpoch = params.BeaconConfig().FarFutureEpoch

var (
	slasherLowestObservedEpoch = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "slasher_lowest_observed_epoch",
		Help: "The lowest epoch number seen by slasher",
	})
	slasherHighestObservedEpoch = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "slasher_highest_observed_epoch",
		Help: "The highest epoch number seen by slasher",
	})
	epochSpansCacheEvictions = promauto.NewCounter(prometheus.CounterOpts{
		Name: "epoch_spans_cache_evictions_total",
		Help: "The number of cache evictions seen by slasher",
	})
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
func (db *Store) EpochSpans(_ context.Context, epoch uint64, fromCache bool) (*types.EpochStore, error) {
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
	if len(es.Bytes())%int(types.SpannerEncodedLength) != 0 {
		return types.ErrWrongSize
	}
	//also prune indexed attestations older then weak subjectivity period
	if err := db.setObservedEpochs(ctx, epoch); err != nil {
		return err
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
	length := db.flatSpanCache.Length()
	log.Debugf("Span cache length %d", length)
	return length
}

// EnableSpanCache used to enable or disable span map cache in tests.
func (db *Store) EnableSpanCache(enable bool) {
	db.spanCacheEnabled = enable
}

func (db *Store) setObservedEpochs(ctx context.Context, epoch uint64) error {
	var err error
	if epoch > highestObservedEpoch {
		slasherHighestObservedEpoch.Set(float64(epoch))
		highestObservedEpoch = epoch
		// Prune block header history every PruneSlasherStoragePeriod epoch.
		if highestObservedEpoch%params.BeaconConfig().PruneSlasherStoragePeriod == 0 {
			if err = db.PruneAttHistory(ctx, epoch, params.BeaconConfig().WeakSubjectivityPeriod); err != nil {
				return errors.Wrap(err, "failed to prune indexed attestations store")
			}
		}
	}
	if epoch < lowestObservedEpoch {
		slasherLowestObservedEpoch.Set(float64(epoch))
		lowestObservedEpoch = epoch
	}
	return err
}
