package kv

import (
	"os"
	"path"
	"time"

	"github.com/boltdb/bolt"
	"github.com/karlseguin/ccache"
	"github.com/pkg/errors"
)

// BlockCacheSize specifies 4 epochs worth of blocks cached.
const BlockCacheSize = 256

// VotesCacheSize with 1M validators will only be around 50Mb.
const VotesCacheSize = 1000000

// Store defines an implementation of the Prysm Database interface
// using BoltDB as the underlying persistent kv-store for eth2.
type Store struct {
	db           *bolt.DB
	databasePath string
	blockCache   *ccache.Cache
	votesCache   *ccache.Cache
}

// NewKVStore initializes a new boltDB key-value store at the directory
// path specified, creates the kv-buckets based on the schema, and stores
// an open connection db object as a property of the Store struct.
func NewKVStore(dirPath string) (*Store, error) {
	if err := os.MkdirAll(dirPath, 0700); err != nil {
		return nil, err
	}
	datafile := path.Join(dirPath, "beaconchain.db")
	boltDB, err := bolt.Open(datafile, 0600, &bolt.Options{Timeout: 1 * time.Second, InitialMmapSize: 10e6})
	if err != nil {
		if err == bolt.ErrTimeout {
			return nil, errors.New("cannot obtain database lock, database may be in use by another process")
		}
		return nil, err
	}

	kv := &Store{
		db:           boltDB,
		databasePath: dirPath,
		blockCache:   ccache.New(ccache.Configure().MaxSize(BlockCacheSize)),
		votesCache:   ccache.New(ccache.Configure().MaxSize(VotesCacheSize)),
	}

	if err := kv.db.Update(func(tx *bolt.Tx) error {
		return createBuckets(
			tx,
			attestationsBucket,
			blocksBucket,
			stateBucket,
			validatorsBucket,
			proposerSlashingsBucket,
			attesterSlashingsBucket,
			voluntaryExitsBucket,
			chainMetadataBucket,
			checkpointBucket,
			archivedValidatorSetChangesBucket,
			archivedCommitteeInfoBucket,
			archivedBalancesBucket,
			archivedValidatorParticipationBucket,
			archivedActiveIndicesBucket,
			// Indices buckets.
			attestationHeadBlockRootBucket,
			attestationSourceRootIndicesBucket,
			attestationSourceEpochIndicesBucket,
			attestationTargetRootIndicesBucket,
			attestationTargetEpochIndicesBucket,
			blockSlotIndicesBucket,
			blockParentRootIndicesBucket,
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
