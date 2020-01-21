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
var d *Store

// Store defines an implementation of the Prysm Database interface
// using BoltDB as the underlying persistent kv-store for eth2.
type Store struct {
	db               *bolt.DB
	databasePath     string
	spanCache        *ristretto.Cache
	spanCacheEnabled bool
}
type Config struct {
	spanCacheEnabled bool
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

// ClearDB removes the previously stored directory at the data directory.
func (db *Store) ClearDB() error {
	if _, err := os.Stat(db.databasePath); os.IsNotExist(err) {
		return nil
	}
	return os.RemoveAll(db.databasePath)
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
	datafile := path.Join(dirPath, "slasher.db")
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
		errors.Wrap(err, "failed to start span cache")
		panic(err)
	}
	kv := &Store{db: boltDB, databasePath: dirPath, spanCache: spanCache, spanCacheEnabled: cfg.spanCacheEnabled}

	if err := kv.db.Update(func(tx *bolt.Tx) error {
		return createBuckets(
			tx,
			historicIndexedAttestationsBucket,
			historicBlockHeadersBucket,
			indexedAttestationsIndicesBucket,
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
