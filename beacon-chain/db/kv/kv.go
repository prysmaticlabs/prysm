// Package kv defines a bolt-db, key-value store implementation
// of the Database interface defined by a Prysm beacon node.
package kv

import (
	"context"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/dgraph-io/ristretto"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	prombolt "github.com/prysmaticlabs/prombbolt"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/db/iface"
	"github.com/prysmaticlabs/prysm/v3/config/features"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/io/file"
	bolt "go.etcd.io/bbolt"
)

var _ iface.Database = (*Store)(nil)

const (
	// NumOfValidatorEntries is the size of the validator cache entries.
	// we expect to hold a max of 200K validators, so setting it to 2 million (10x the capacity).
	NumOfValidatorEntries = 1 << 21
	// ValidatorEntryMaxCost is set to ~64Mb to allow 200K validators entries to be cached.
	ValidatorEntryMaxCost = 1 << 26
	// BeaconNodeDbDirName is the name of the directory containing the beacon node database.
	BeaconNodeDbDirName = "beaconchaindata"
	// DatabaseFileName is the name of the beacon node database.
	DatabaseFileName = "beaconchain.db"

	boltAllocSize = 8 * 1024 * 1024
	// The size of hash length in bytes
	hashLength = 32
	// Specifies the initial mmap size of bolt.
	mmapSize = 536870912
)

var (
	// Metrics for the validator cache.
	validatorEntryCacheHit = promauto.NewCounter(prometheus.CounterOpts{
		Name: "validator_entry_cache_hit_total",
		Help: "The total number of cache hits on the validator entry cache.",
	})
	validatorEntryCacheMiss = promauto.NewCounter(prometheus.CounterOpts{
		Name: "validator_entry_cache_miss_total",
		Help: "The total number of cache misses on the validator entry cache.",
	})
	validatorEntryCacheDelete = promauto.NewCounter(prometheus.CounterOpts{
		Name: "validator_entry_cache_delete_total",
		Help: "The total number of cache deletes on the validator entry cache.",
	})
)

// BlockCacheSize specifies 1000 slots worth of blocks cached, which
// would be approximately 2MB
var BlockCacheSize = int64(1 << 21)

// blockedBuckets represents the buckets that we want to restrict
// from our metrics fetching for performance reasons. For a detailed
// summary, it can be read in https://github.com/prysmaticlabs/prysm/issues/8274.
var blockedBuckets = [][]byte{
	blocksBucket,
	stateSummaryBucket,
	blockParentRootIndicesBucket,
	blockSlotIndicesBucket,
	finalizedBlockRootsIndexBucket,
}

// Store defines an implementation of the Prysm Database interface
// using BoltDB as the underlying persistent kv-store for Ethereum Beacon Nodes.
type Store struct {
	db                  *bolt.DB
	databasePath        string
	blockCache          *ristretto.Cache
	validatorEntryCache *ristretto.Cache
	stateSummaryCache   *stateSummaryCache
	ctx                 context.Context
}

// KVStoreDatafilePath is the canonical construction of a full
// database file path from the directory path, so that code outside
// this package can find the full path in a consistent way.
func KVStoreDatafilePath(dirPath string) string {
	return path.Join(dirPath, DatabaseFileName)
}

// NewKVStore initializes a new boltDB key-value store at the directory
// path specified, creates the kv-buckets based on the schema, and stores
// an open connection db object as a property of the Store struct.
func NewKVStore(ctx context.Context, dirPath string) (*Store, error) {
	hasDir, err := file.HasDir(dirPath)
	if err != nil {
		return nil, err
	}
	if !hasDir {
		if err := file.MkdirAll(dirPath); err != nil {
			return nil, err
		}
	}
	datafile := KVStoreDatafilePath(dirPath)
	log.Infof("Opening Bolt DB at %s", datafile)
	boltDB, err := bolt.Open(
		datafile,
		params.BeaconIoConfig().ReadWritePermissions,
		&bolt.Options{
			Timeout:         1 * time.Second,
			InitialMmapSize: mmapSize,
		},
	)
	if err != nil {
		if errors.Is(err, bolt.ErrTimeout) {
			return nil, errors.New("cannot obtain database lock, database may be in use by another process")
		}
		return nil, err
	}
	boltDB.AllocSize = boltAllocSize
	blockCache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1000,           // number of keys to track frequency of (1000).
		MaxCost:     BlockCacheSize, // maximum cost of cache (1000 Blocks).
		BufferItems: 64,             // number of keys per Get buffer.
	})
	if err != nil {
		return nil, err
	}

	validatorCache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: NumOfValidatorEntries, // number of entries in cache (2 Million).
		MaxCost:     ValidatorEntryMaxCost, // maximum size of the cache (64Mb)
		BufferItems: 64,                    // number of keys per Get buffer.
	})
	if err != nil {
		return nil, err
	}

	kv := &Store{
		db:                  boltDB,
		databasePath:        dirPath,
		blockCache:          blockCache,
		validatorEntryCache: validatorCache,
		stateSummaryCache:   newStateSummaryCache(),
		ctx:                 ctx,
	}
	if err := kv.db.Update(func(tx *bolt.Tx) error {
		return createBuckets(
			tx,
			attestationsBucket,
			blocksBucket,
			stateBucket,
			proposerSlashingsBucket,
			attesterSlashingsBucket,
			voluntaryExitsBucket,
			chainMetadataBucket,
			checkpointBucket,
			powchainBucket,
			stateSummaryBucket,
			stateValidatorsBucket,
			// Indices buckets.
			attestationHeadBlockRootBucket,
			attestationSourceRootIndicesBucket,
			attestationSourceEpochIndicesBucket,
			attestationTargetRootIndicesBucket,
			attestationTargetEpochIndicesBucket,
			blockSlotIndicesBucket,
			stateSlotIndicesBucket,
			blockParentRootIndicesBucket,
			finalizedBlockRootsIndexBucket,
			blockRootValidatorHashesBucket,
			// State management service bucket.
			newStateServiceCompatibleBucket,
			// Migrations
			migrationsBucket,

			feeRecipientBucket,
			registrationBucket,
		)
	}); err != nil {
		return nil, err
	}
	if err = prometheus.Register(createBoltCollector(kv.db)); err != nil {
		return nil, err
	}
	if err = kv.checkNeedsResync(); err != nil {
		return nil, err
	}
	return kv, nil
}

// ClearDB removes the previously stored database in the data directory.
func (s *Store) ClearDB() error {
	if _, err := os.Stat(s.databasePath); os.IsNotExist(err) {
		return nil
	}
	prometheus.Unregister(createBoltCollector(s.db))
	if err := os.Remove(path.Join(s.databasePath, DatabaseFileName)); err != nil {
		return errors.Wrap(err, "could not remove database file")
	}
	return nil
}

// Close closes the underlying BoltDB database.
func (s *Store) Close() error {
	prometheus.Unregister(createBoltCollector(s.db))

	// Before DB closes, we should dump the cached state summary objects to DB.
	if err := s.saveCachedStateSummariesDB(s.ctx); err != nil {
		return err
	}

	return s.db.Close()
}

// DatabasePath at which this database writes files.
func (s *Store) DatabasePath() string {
	return s.databasePath
}

func (s *Store) checkNeedsResync() error {
	return s.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(migrationsBucket)
		hasDisabledFeature := !features.Get().EnableOnlyBlindedBeaconBlocks
		if hasDisabledFeature && bkt.Get(migrationBlindedBeaconBlocksKey) != nil {
			return fmt.Errorf(
				"you have disabled the flag %s, and your node must resync to ensure your "+
					"database is compatible. If you do not want to resync, please re-enable the %s flag",
				features.EnableOnlyBlindedBeaconBlocks.Name,
				features.EnableOnlyBlindedBeaconBlocks.Name,
			)
		}
		return nil
	})

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
	return prombolt.New("boltDB", db, blockedBuckets...)
}
