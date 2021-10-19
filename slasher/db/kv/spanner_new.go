package kv

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	slashertypes "github.com/prysmaticlabs/prysm/slasher/detection/attestations/types"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

// Tracks the highest and lowest observed epochs from the validator span maps
// used for attester slashing detection. This value is purely used
// as a cache key and only needs to be maintained in memory.
var highestObservedEpoch types.Epoch
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
			epoch, keyOK := key.(types.Epoch)
			epochStore, valueOK := value.(*slashertypes.EpochStore)
			if !keyOK || !valueOK {
				return errors.New("could not cast key and value into needed types")
			}

			bucket := tx.Bucket(validatorsMinMaxSpanBucketNew)
			if err := bucket.Put(bytesutil.Bytes8(uint64(epoch)), epochStore.Bytes()); err != nil {
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
func (s *Store) EpochSpans(_ context.Context, epoch types.Epoch, fromCache bool) (*slashertypes.EpochStore, error) {
	// Get from the cache if it exists or is requested, if not, go to DB.
	if fromCache && s.flatSpanCache.Has(epoch) || s.flatSpanCache.Has(epoch) {
		spans, _ := s.flatSpanCache.Get(epoch)
		return spans, nil
	}

	var copiedSpans []byte
	err := s.view(func(tx *bolt.Tx) error {
		b := tx.Bucket(validatorsMinMaxSpanBucketNew)
		if b == nil {
			return nil
		}
		spans := b.Get(bytesutil.Bytes8(uint64(epoch)))
		copiedSpans = make([]byte, len(spans))
		copy(copiedSpans, spans)
		return nil
	})
	if err != nil {
		return &slashertypes.EpochStore{}, err
	}
	if copiedSpans == nil {
		copiedSpans = []byte{}
	}
	return slashertypes.NewEpochStore(copiedSpans)
}

// SaveEpochSpans accepts a epoch and span byte array and writes it to disk.
func (s *Store) SaveEpochSpans(ctx context.Context, epoch types.Epoch, es *slashertypes.EpochStore, toCache bool) error {
	if len(es.Bytes())%int(slashertypes.SpannerEncodedLength) != 0 {
		return slashertypes.ErrWrongSize
	}
	// Also prune indexed attestations older then weak subjectivity period.
	if err := s.setObservedEpochs(ctx, epoch); err != nil {
		return err
	}
	// Saving to the cache if it exists so cache and DB never conflict.
	if toCache || s.flatSpanCache.Has(epoch) {
		s.flatSpanCache.Set(epoch, es)
	}
	if toCache {
		return nil
	}

	return s.update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(validatorsMinMaxSpanBucketNew)
		if err != nil {
			return err
		}
		return b.Put(bytesutil.Bytes8(uint64(epoch)), es.Bytes())
	})
}

// CacheLength returns the number of cached items.
func (s *Store) CacheLength(ctx context.Context) int {
	ctx, span := trace.StartSpan(ctx, "slasherDB.CacheLength")
	defer span.End()
	length := s.flatSpanCache.Length()
	log.Debugf("Span cache length %d", length)
	return length
}

// EnableSpanCache used to enable or disable span map cache in tests.
func (s *Store) EnableSpanCache(enable bool) {
	s.spanCacheEnabled = enable
}

func (s *Store) setObservedEpochs(ctx context.Context, epoch types.Epoch) error {
	var err error
	if epoch > highestObservedEpoch {
		slasherHighestObservedEpoch.Set(float64(epoch))
		highestObservedEpoch = epoch
		// Prune block header history every PruneSlasherStoragePeriod epoch.
		if highestObservedEpoch%params.BeaconConfig().PruneSlasherStoragePeriod == 0 {
			if err = s.PruneAttHistory(ctx, epoch, params.BeaconConfig().WeakSubjectivityPeriod); err != nil {
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
