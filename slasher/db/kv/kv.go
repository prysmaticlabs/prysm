// Package kv defines a bolt-db, key-value store implementation of
// the slasher database interface.
package kv

import (
	"context"
	"os"
	"path"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/io/file"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/slasher/cache"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

const (
	// SlasherDbDirName is the name of the directory containing the slasher database.
	SlasherDbDirName = "slasherdata"
	// DatabaseFileName is the name of the slasher database.
	DatabaseFileName = "slasher.db"
)

// Store defines an implementation of the slasher Database interface
// using BoltDB as the underlying persistent kv-store for Ethereum.
type Store struct {
	highestAttCacheEnabled  bool
	spanCacheEnabled        bool
	highestAttestationCache *cache.HighestAttestationCache
	flatSpanCache           *cache.EpochFlatSpansCache
	db                      *bolt.DB
	databasePath            string
}

// Config options for the slasher db.
type Config struct {
	// SpanCacheSize determines the span map cache size.
	SpanCacheSize               int
	HighestAttestationCacheSize int
}

// Close closes the underlying boltdb database.
func (s *Store) Close() error {
	s.flatSpanCache.Purge()
	s.highestAttestationCache.Purge()
	return s.db.Close()
}

// RemoveOldestFromCache clears the oldest key out of the cache only if the cache is at max capacity.
func (s *Store) RemoveOldestFromCache(ctx context.Context) uint64 {
	ctx, span := trace.StartSpan(ctx, "slasherDB.removeOldestFromCache")
	defer span.End()
	epochRemoved := s.flatSpanCache.PruneOldest()
	return epochRemoved
}

// ClearSpanCache clears the spans cache.
func (s *Store) ClearSpanCache() {
	s.flatSpanCache.Purge()
}

func (s *Store) update(fn func(*bolt.Tx) error) error {
	return s.db.Update(fn)
}
func (s *Store) view(fn func(*bolt.Tx) error) error {
	return s.db.View(fn)
}

// ClearDB removes any previously stored data at the configured data directory.
func (s *Store) ClearDB() error {
	if _, err := os.Stat(s.databasePath); os.IsNotExist(err) {
		return nil
	}
	return os.Remove(filepath.Join(s.databasePath, DatabaseFileName))
}

// DatabasePath at which this database writes files.
func (s *Store) DatabasePath() string {
	return s.databasePath
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
	hasDir, err := file.HasDir(dirPath)
	if err != nil {
		return nil, err
	}
	if !hasDir {
		if err := file.MkdirAll(dirPath); err != nil {
			return nil, err
		}
	}

	datafile := path.Join(dirPath, DatabaseFileName)
	boltDB, err := bolt.Open(datafile, params.BeaconIoConfig().ReadWritePermissions, &bolt.Options{Timeout: params.BeaconIoConfig().BoltTimeout})
	if err != nil {
		if errors.Is(err, bolt.ErrTimeout) {
			return nil, errors.New("cannot obtain database lock, database may be in use by another process")
		}
		return nil, err
	}
	kv := &Store{db: boltDB, databasePath: dirPath}
	kv.EnableSpanCache(true)
	kv.EnableHighestAttestationCache(true)
	flatSpanCache, err := cache.NewEpochFlatSpansCache(cfg.SpanCacheSize, persistFlatSpanMapsOnEviction(kv))
	if err != nil {
		return nil, errors.Wrap(err, "could not create new flat cache")
	}
	kv.flatSpanCache = flatSpanCache
	highestAttCache, err := cache.NewHighestAttestationCache(cfg.HighestAttestationCacheSize, persistHighestAttestationCacheOnEviction(kv))
	kv.highestAttestationCache = highestAttCache

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
			highestAttestationBucket,
		)
	}); err != nil {
		return nil, err
	}

	return kv, err
}

// Size returns the db size in bytes.
func (s *Store) Size() (int64, error) {
	var size int64
	err := s.db.View(func(tx *bolt.Tx) error {
		size = tx.Size()
		return nil
	})
	return size, err
}
