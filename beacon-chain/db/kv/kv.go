package kv

import (
	"os"
	"path"
	"time"

	"github.com/boltdb/bolt"
	"github.com/dgraph-io/ristretto"
	"github.com/mdlayher/prombolt"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	// VotesCacheSize with 1M validators will only be around 50Mb.
	VotesCacheSize   = 1000000
	databaseFileName = "beaconchain.db"
)

// BlockCacheSize specifies 4 epochs worth of blocks cached.
var BlockCacheSize = int64(256)

// Store defines an implementation of the Prysm Database interface
// using BoltDB as the underlying persistent kv-store for eth2.
type Store struct {
	db                  *bolt.DB
	databasePath        string
	blockCache          *ristretto.Cache
	votesCache          *ristretto.Cache
	validatorIndexCache *ristretto.Cache
}

// NewKVStore initializes a new boltDB key-value store at the directory
// path specified, creates the kv-buckets based on the schema, and stores
// an open connection db object as a property of the Store struct.
func NewKVStore(dirPath string) (*Store, error) {
	if err := os.MkdirAll(dirPath, 0700); err != nil {
		return nil, err
	}
	datafile := path.Join(dirPath, databaseFileName)
	boltDB, err := bolt.Open(datafile, 0600, &bolt.Options{Timeout: 1 * time.Second, InitialMmapSize: 10e6})
	if err != nil {
		if err == bolt.ErrTimeout {
			return nil, errors.New("cannot obtain database lock, database may be in use by another process")
		}
		return nil, err
	}
	blockCache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: BlockCacheSize, // number of keys to track frequency of (1M).
		MaxCost:     1 << 23,        // maximum cost of cache (10MB).
		BufferItems: 64,             // number of keys per Get buffer.
	})
	if err != nil {
		return nil, err
	}

	votesCache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: VotesCacheSize, // number of keys to track frequency of (1M).
		MaxCost:     1 << 23,        // maximum cost of cache (10MB).
		BufferItems: 64,             // number of keys per Get buffer.
	})
	if err != nil {
		return nil, err
	}

	validatorCache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: VotesCacheSize, // number of keys to track frequency of (1M).
		MaxCost:     1 << 23,        // maximum cost of cache (10MB).
		BufferItems: 64,             // number of keys per Get buffer.
	})
	if err != nil {
		return nil, err
	}

	kv := &Store{
		db:                  boltDB,
		databasePath:        dirPath,
		blockCache:          blockCache,
		votesCache:          votesCache,
		validatorIndexCache: validatorCache,
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
			// Indices buckets.
			attestationHeadBlockRootBucket,
			attestationSourceRootIndicesBucket,
			attestationSourceEpochIndicesBucket,
			attestationTargetRootIndicesBucket,
			attestationTargetEpochIndicesBucket,
			blockSlotIndicesBucket,
			blockParentRootIndicesBucket,
			finalizedBlockRootsIndexBucket,
		)
	}); err != nil {
		return nil, err
	}
	err = prometheus.Register(createBoltCollector(kv.db))

	return kv, err
}

// ClearDB removes the previously stored database in the data directory.
func (k *Store) ClearDB() error {
	if _, err := os.Stat(k.databasePath); os.IsNotExist(err) {
		return nil
	}
	prometheus.Unregister(createBoltCollector(k.db))
	return os.Remove(path.Join(k.databasePath, databaseFileName))
}

// Close closes the underlying BoltDB database.
func (k *Store) Close() error {
	prometheus.Unregister(createBoltCollector(k.db))
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

// createBoltCollector returns a prometheus collector specifically configured for boltdb.
func createBoltCollector(db *bolt.DB) prometheus.Collector {
	return prombolt.New("boltDB", db)
}
