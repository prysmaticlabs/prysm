// Package kv defines a bolt-db, key-value store implementation of
// the slasher database interface.
package kv

import (
	"context"
	"os"
	"path"
	"time"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/slasher/cache"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

var databaseFileName = "slasher.db"

// Store defines an implementation of the slasher Database interface
// using BoltDB as the underlying persistent kv-store for eth2.
type Store struct {
	db               *bolt.DB
	databasePath     string
	spanCache        *cache.EpochSpansCache
	flatSpanCache    *cache.EpochFlatSpansCache
	spanCacheEnabled bool
}

// Config options for the slasher db.
type Config struct {
	// SpanCacheSize determines the span map cache size.
	SpanCacheSize int
}

// Close closes the underlying boltdb database.
func (db *Store) Close() error {
	db.flatSpanCache.Purge()
	return db.db.Close()
}

// RemoveOldestFromCache clears the oldest key out of the cache only if the cache is at max capacity.
func (db *Store) RemoveOldestFromCache(ctx context.Context) uint64 {
	ctx, span := trace.StartSpan(ctx, "slasherDB.removeOldestFromCache")
	defer span.End()
	epochRemoved := db.flatSpanCache.PruneOldest()
	return epochRemoved
}

// ClearSpanCache clears the spans cache.
func (db *Store) ClearSpanCache() {
	db.flatSpanCache.Purge()
}

func (db *Store) update(fn func(*bolt.Tx) error) error {
	return db.db.Update(fn)
}
func (db *Store) batch(fn func(*bolt.Tx) error) error {
	return db.db.Batch(fn)
}
func (db *Store) view(fn func(*bolt.Tx) error) error {
	return db.db.View(fn)
}

// ClearDB removes any previously stored data at the configured data directory.
func (db *Store) ClearDB() error {
	if _, err := os.Stat(db.databasePath); os.IsNotExist(err) {
		return nil
	}
	return os.Remove(db.databasePath)
}

// DatabasePath at which this database writes files.
func (db *Store) DatabasePath() string {
	return db.databasePath
}

func createBuckets(tx *bolt.Tx, buckets ...[]byte) error {
	for _, bucket := range buckets {
		if _, err := tx.CreateBucketIfNotExists(bucket); err != nil {
			return err
		}
	}
	return nil
}

// NewKVStore initializes a new boltDB key-value store at the directory
// path specified, creates the kv-buckets based on the schema, and stores
// an open connection db object as a property of the Store struct.
func NewKVStore(dirPath string, cfg *Config) (*Store, error) {
	if err := os.MkdirAll(dirPath, 0700); err != nil {
		return nil, err
	}
	datafile := path.Join(dirPath, databaseFileName)
	boltDB, err := bolt.Open(datafile, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		if err == bolt.ErrTimeout {
			return nil, errors.New("cannot obtain database lock, database may be in use by another process")
		}
		return nil, err
	}
	kv := &Store{db: boltDB, databasePath: datafile}
	kv.EnableSpanCache(true)
	spanCache, err := cache.NewEpochSpansCache(cfg.SpanCacheSize, persistSpanMapsOnEviction(kv))
	if err != nil {
		return nil, errors.Wrap(err, "could not create new cache")
	}
	kv.spanCache = spanCache
	flatSpanCache, err := cache.NewEpochFlatSpansCache(cfg.SpanCacheSize, persistFlatSpanMapsOnEviction(kv))
	if err != nil {
		return nil, errors.Wrap(err, "could not create new flat cache")
	}
	kv.flatSpanCache = flatSpanCache

	if err := kv.db.Update(func(tx *bolt.Tx) error {
		return createBuckets(
			tx,
			indexedAttestationsBucket,
			indexedAttestationsRootsByTargetBucket,
			historicIndexedAttestationsBucket,
			historicBlockHeadersBucket,
			compressedIdxAttsBucket,
			validatorsPublicKeysBucket,
			validatorsMinMaxSpanBucket,
			validatorsMinMaxSpanBucketNew,
			slashingBucket,
			chainDataBucket,
		)
	}); err != nil {
		return nil, err
	}

	return kv, err
}

// Size returns the db size in bytes.
func (db *Store) Size() (int64, error) {
	var size int64
	err := db.db.View(func(tx *bolt.Tx) error {
		size = tx.Size()
		return nil
	})
	return size, err
}
