// Package kv defines a persistent backend for the validator service.
package kv

import (
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/params"
	bolt "go.etcd.io/bbolt"
)

var databaseFileName = "validator.db"

// Store defines an implementation of the Prysm Database interface
// using BoltDB as the underlying persistent kv-store for eth2.
type Store struct {
	db           *bolt.DB
	databasePath string
}

// Close closes the underlying boltdb database.
func (store *Store) Close() error {
	return store.db.Close()
}

func (store *Store) update(fn func(*bolt.Tx) error) error {
	return store.db.Update(fn)
}
func (store *Store) batch(fn func(*bolt.Tx) error) error {
	return store.db.Batch(fn)
}
func (store *Store) view(fn func(*bolt.Tx) error) error {
	return store.db.View(fn)
}

// ClearDB removes any previously stored data at the configured data directory.
func (store *Store) ClearDB() error {
	if _, err := os.Stat(store.databasePath); os.IsNotExist(err) {
		return nil
	}
	return os.Remove(filepath.Join(store.databasePath, databaseFileName))
}

// DatabasePath at which this database writes files.
func (store *Store) DatabasePath() string {
	return store.databasePath
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
func NewKVStore(dirPath string, pubKeys [][48]byte) (*Store, error) {
	if err := os.MkdirAll(dirPath, params.BeaconIoConfig().ReadWriteExecutePermissions); err != nil {
		return nil, err
	}
	datafile := filepath.Join(dirPath, databaseFileName)
	boltDB, err := bolt.Open(datafile, params.BeaconIoConfig().ReadWritePermissions, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		if err == bolt.ErrTimeout {
			return nil, errors.New("cannot obtain database lock, database may be in use by another process")
		}
		return nil, err
	}

	kv := &Store{db: boltDB, databasePath: dirPath}

	if err := kv.db.Update(func(tx *bolt.Tx) error {
		return createBuckets(
			tx,
			historicProposalsBucket,
			historicAttestationsBucket,
		)
	}); err != nil {
		return nil, err
	}

	// Initialize the required public keys into the DB to ensure they're not empty.
	if err := kv.initializeSubBuckets(pubKeys); err != nil {
		return nil, err
	}

	return kv, err
}

// GetKVStore returns the validator boltDB key-value store from directory. Returns nil if no such store exists.
func GetKVStore(directory string) (*Store, error) {
	fileName := filepath.Join(directory, databaseFileName)
	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		return nil, nil
	}
	boltDb, err := bolt.Open(fileName, params.BeaconIoConfig().ReadWritePermissions, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		if err == bolt.ErrTimeout {
			return nil, errors.New("cannot obtain database lock, database may be in use by another process")
		}
		return nil, err
	}

	return &Store{db: boltDb, databasePath: directory}, nil
}

// Size returns the db size in bytes.
func (store *Store) Size() (int64, error) {
	var size int64
	err := store.db.View(func(tx *bolt.Tx) error {
		size = tx.Size()
		return nil
	})
	return size, err
}
