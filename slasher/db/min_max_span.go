package db

import (
	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	lru "github.com/hashicorp/golang-lru"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	slashpb "github.com/prysmaticlabs/prysm/proto/slashing"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/slasher/flags"
)

const maxCacheSize = 10000

var (
	spanCache *lru.ARCCache
	// Metrics
	spanCacheMiss = promauto.NewCounter(prometheus.CounterOpts{
		Name: "span_cache_miss",
		Help: "The number of span data requests that aren't present in the cache.",
	})
	//SpanCacheHit cache hits.
	spanCacheHit = promauto.NewCounter(prometheus.CounterOpts{
		Name: "span_cache_hit",
		Help: "The number of span data requests that are present in the cache.",
	})
	spanCacheSize = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "span_cache_size",
		Help: "The number of span map data in the span cache",
	})
)

func init() {
	var err error
	spanCache, err = lru.NewARC(maxCacheSize)
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
		sm, ok := spanCache.Get(validatorIdx)
		if ok {

			spanCacheHit.Inc()
			return sm.(*slashpb.EpochSpanMap), nil
		}
		spanCacheMiss.Inc()
	}
	var enc []byte
	err := db.view(func(tx *bolt.Tx) error {
		b := tx.Bucket(validatorsMinMaxSpanBucket)
		enc = b.Get(bytesutil.Bytes4(validatorIdx))
		return nil
	})
	sm, err = createEpochSpanMap(enc)
	if sm.EpochSpanMap == nil {
		sm.EpochSpanMap = make(map[uint64]*slashpb.MinMaxEpochSpan)
	}
	return sm, err
}

// SaveValidatorSpansMap accepts a validator index and span map and writes it to disk.
func (db *Store) SaveValidatorSpansMap(validatorIdx uint64, spanMap *slashpb.EpochSpanMap) error {
	spanCache.Add(validatorIdx, spanMap)
	spanCacheSize.Set(float64(spanCache.Len()))
	//er := make(chan error)
	//go func(validatorIdx uint64, spanMap *slashpb.EpochSpanMap, errChan chan error) {
	//	val, err := proto.Marshal(spanMap)
	//	if err != nil {
	//		errChan <- errors.Wrap(err, "failed to marshal span map")
	//		close(errChan)
	//		return
	//	}
	//	key := bytesutil.Bytes4(validatorIdx)
	//	err = db.batch(func(tx *bolt.Tx) error {
	//		bucket := tx.Bucket(validatorsMinMaxSpanBucket)
	//		if err := bucket.Put(key, val); err != nil {
	//			errChan <- errors.Wrapf(err, "failed to delete validator id: %d from validators min max span bucket", validatorIdx)
	//
	//		}
	//		return err
	//	})
	//	close(errChan)
	//}(validatorIdx, spanMap, er)
	//return er
	return nil
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
