package db

import (
	"errors"
	"os"
	"path"
	"sync"
	"time"

	"github.com/boltdb/bolt"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "validatordb")

// ValidatorDB manages the data layer of the validator implementation.
type ValidatorDB struct {
	stateLock              sync.RWMutex
	db                     *bolt.DB
	DatabasePath           string
	lastProposedBlockEpoch map[bls.PublicKey]uint64 //TODO test work with cache
	lastAttestationEpoch   map[bls.PublicKey]uint64
}

// Close closes the underlying boltdb database.
func (db *ValidatorDB) Close() error {
	return db.db.Close()
}

func (db *ValidatorDB) update(fn func(*bolt.Tx) error) error {
	return db.db.Update(fn)
}
func (db *ValidatorDB) batch(fn func(*bolt.Tx) error) error {
	return db.db.Batch(fn)
}
func (db *ValidatorDB) view(fn func(*bolt.Tx) error) error {
	return db.db.View(fn)
}

func createBuckets(tx *bolt.Tx, buckets ...[]byte) error {
	for _, bucket := range buckets {
		if _, err := tx.CreateBucketIfNotExists(bucket); err != nil {
			return err
		}
	}

	return nil
}

// NewDB initializes a new DB.
// Using buckets for seperate validator (by pubkey), and subbuckets for save propose block and attests
func NewDB(dirPath string) (*ValidatorDB, error) {
	if err := os.MkdirAll(dirPath, 0700); err != nil {
		return nil, err
	}
	datafile := path.Join(dirPath, "validator.db")
	boltDB, err := bolt.Open(datafile, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		if err == bolt.ErrTimeout {
			return nil, errors.New("cannot obtain database lock, database may be in use by another process")
		}
		return nil, err
	}

	db := &ValidatorDB{
		db:                     boltDB,
		DatabasePath:           dirPath,
		lastProposedBlockEpoch: make(map[bls.PublicKey]uint64),
		lastAttestationEpoch:   make(map[bls.PublicKey]uint64),
	}

	return db, err
}

// ClearDB removes the previously stored directory at the data directory.
func ClearDB(dirPath string) error {
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		return nil
	}
	return os.RemoveAll(dirPath) // TODO remove only database file, not keys file
}
