package kv

import (
	"os"
	"path"
	"time"

	"github.com/boltdb/bolt"
	"github.com/pkg/errors"
)

type KVStore struct {
	db           *bolt.DB
	DatabasePath string
}

// NewKVStore initializes a new boltDB key-value store at the directory
// path specified, creates the kv-buckets based on the schema, and stores
// an open connection db object as a property of the KVStore struct.
func NewKVStore(dirPath string) (*KVStore, error) {
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

	kv := &KVStore{db: boltDB, DatabasePath: dirPath}

	if err := kv.db.Update(func(tx *bolt.Tx) error {
		//return createBuckets(tx, blockBucket, attestationBucket, attestationTargetBucket, mainChainBucket,
		//	histStateBucket, chainInfoBucket, cleanupHistoryBucket, blockOperationsBucket, validatorBucket)
		// TODO: Update buckets.
		return nil
	}); err != nil {
		return nil, err
	}

	return kv, err
}

// ClearDB removes the previously stored directory at the data directory.
func (k *KVStore) ClearDB() error {
	if _, err := os.Stat(k.DatabasePath); os.IsNotExist(err) {
		return nil
	}
	return os.RemoveAll(k.DatabasePath)
}

// Close closes the underlying boltdb database.
func (k *KVStore) Close() error {
	return k.db.Close()
}

func createBuckets(tx *bolt.Tx, buckets ...[]byte) error {
	for _, bucket := range buckets {
		if _, err := tx.CreateBucketIfNotExists(bucket); err != nil {
			return err
		}
	}
	return nil
}
