package db

import (
	"os"
	"path"
	"time"

	"github.com/boltdb/bolt"
	"github.com/dgraph-io/ristretto"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "slasherDB")

var databaseFileName = "slasher.db"

var d *Store

// Store defines an implementation of the Prysm Database interface
// using BoltDB as the underlying persistent kv-store for eth2.
type Store struct {
	db               *bolt.DB
	databasePath     string
	spanCache        *ristretto.Cache
	spanCacheEnabled bool
}

// Config options for the slasher db.
type Config struct {
	// SpanCacheEnabled use span cache to detect surround slashing.
	SpanCacheEnabled bool
	cacheItems       int64
	maxCacheSize     int64
}

// Close closes the underlying boltdb database.
func (db *Store) Close() error {
	return db.db.Close()
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

// NewDB initializes a new DB.
func NewDB(dirPath string, cfg *Config) (*Store, error) {
	var err error
	d, err = NewKVStore(dirPath, cfg)
	return d, err
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
	if cfg.cacheItems == 0 {
		cfg.cacheItems = 20000
	}
	if cfg.maxCacheSize == 0 {
		cfg.maxCacheSize = 2 << 30 //(2GB)
	}
	spanCache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: cfg.cacheItems,   // number of keys to track frequency of (10M).
		MaxCost:     cfg.maxCacheSize, // maximum cost of cache.
		BufferItems: 64,               // number of keys per Get buffer.
		OnEvict:     saveToDB,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to start span cache")
	}
	kv := &Store{db: boltDB, databasePath: datafile, spanCache: spanCache, spanCacheEnabled: cfg.SpanCacheEnabled}

	if err := kv.db.Update(func(tx *bolt.Tx) error {
		return createBuckets(
			tx,
			historicIndexedAttestationsBucket,
			historicBlockHeadersBucket,
			compressedIdxAttsBucket,
			validatorsPublicKeysBucket,
			validatorsMinMaxSpanBucket,
			slashingBucket,
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
