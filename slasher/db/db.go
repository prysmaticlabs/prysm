package db

import (
	"io"
	"os"
	"path"
	"time"

	"github.com/boltdb/bolt"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "beacondb")

// Database defines the necessary methods for Prysm's eth2 backend which may
// be implemented by any key-value or relational database in practice.
type Database interface {
	io.Closer
	DatabasePath() string

	ClearDB() error
}

// Store defines an implementation of the Prysm Database interface
// using BoltDB as the underlying persistent kv-store for eth2.
type Store struct {
	db           *bolt.DB
	databasePath string
}

// SlasherDB manages the data layer of the beacon chain implementation.
// The exposed methods do not have an opinion of the underlying data engine,
// but instead reflect the beacon chain logic.
// For example, instead of defining get, put, remove
// This defines methods such as getBlock, saveBlocksAndAttestations, etc.
type SlasherDB struct {
	db           *Store
	DatabasePath string
}

// Close closes the underlying boltdb database.
func (db *SlasherDB) Close() error {
	return db.db.Close()
}

func (db *SlasherDB) update(fn func(*bolt.Tx) error) error {
	return db.db.db.Update(fn)
}
func (db *SlasherDB) batch(fn func(*bolt.Tx) error) error {
	return db.db.db.Batch(fn)
}
func (db *SlasherDB) view(fn func(*bolt.Tx) error) error {
	return db.db.db.View(fn)
}

// NewDB initializes a new DB.
func NewDB(dirPath string) (*SlasherDB, error) {
	return NewKVStore(dirPath)
}

// NewKVStore initializes a new boltDB key-value store at the directory
// path specified, creates the kv-buckets based on the schema, and stores
// an open connection db object as a property of the Store struct.
func NewKVStore(dirPath string) (*SlasherDB, error) {
	if err := os.MkdirAll(dirPath, 0700); err != nil {
		return nil, err
	}
	datafile := path.Join(dirPath, "beaconchain.db")
	boltDB, err := bolt.Open(datafile, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		if err == bolt.ErrTimeout {
			return nil, errors.New("cannot obtain database lock, database may be in use by another process")
		}
		return nil, err
	}

	kv := &Store{db: boltDB, databasePath: dirPath}

	if err := kv.db.Update(func(tx *bolt.Tx) error {
		return createBuckets(
			tx, cleanupHistoryBucket,
			historicAttestationsBucket,
			historicBlockHeadersBucket,
		)
	}); err != nil {
		return nil, err
	}

	return kv, err
}

// ClearDB removes the previously stored directory at the data directory.
func (k *Store) ClearDB() error {
	if _, err := os.Stat(k.databasePath); os.IsNotExist(err) {
		return nil
	}
	return os.RemoveAll(k.databasePath)
}

// Close closes the underlying BoltDB database.
func (k *Store) Close() error {
	return k.db.Close()
}

// DatabasePath at which this database writes files.
func (k *Store) DatabasePath() string {
	return k.databasePath
}

func createBuckets(tx *bolt.Tx, buckets ...[]byte) error {
	for _, bucket := range buckets {
		if _, err := tx.CreateBucketIfNotExists(bucket); err != nil {
			return err
		}
	}
	return nil
}
