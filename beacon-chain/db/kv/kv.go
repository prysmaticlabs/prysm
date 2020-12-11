// Package kv defines a bolt-db, key-value store implementation
// of the Database interface defined by a Prysm beacon node.
package kv

import (
	"context"
	"os"
	"path"
	"sync"
	"time"

	"github.com/dgraph-io/ristretto"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	prombolt "github.com/prysmaticlabs/prombbolt"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/iface"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/fileutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

var _ iface.Database = (*Store)(nil)
var defaultHotStateDBInterval uint64 = 128 // slots

const (
	// VotesCacheSize with 1M validators will be 8MB.
	VotesCacheSize = 1 << 23
	// NumOfVotes specifies the vote cache size.
	NumOfVotes = 1 << 20
	// BeaconNodeDbDirName is the name of the directory containing the beacon node database.
	BeaconNodeDbDirName = "beaconchaindata"
	// DatabaseFileName is the name of the beacon node database.
	DatabaseFileName = "beaconchain.db"

	boltAllocSize = 8 * 1024 * 1024
)

// BlockCacheSize specifies 1000 slots worth of blocks cached, which
// would be approximately 2MB
var BlockCacheSize = int64(1 << 21)

// Store defines an implementation of the Prysm Database interface
// using BoltDB as the underlying persistent kv-store for eth2.
type Store struct {
	db                      *bolt.DB
	databasePath            string
	blockCache              *ristretto.Cache
	validatorIndexCache     *ristretto.Cache
	stateSummaryCache       *StateSummaryCache
	slotsPerArchivedPoint   uint64
	hotStateCache           *hotStateCache
	finalizedInfo           *finalizedInfo
	epochBoundaryStateCache *epochBoundaryState
	saveHotStateDB          *saveHotStateDbConfig
}

// This tracks the config in the event of long non-finality,
// how often does the node save hot states to db? what are
// the saved hot states in db?... etc
type saveHotStateDbConfig struct {
	enabled         bool
	lock            sync.Mutex
	duration        uint64
	savedStateRoots [][32]byte
}

// This tracks the finalized point. It's also the point where slot and the block root of
// cold and hot sections of the DB splits.
type finalizedInfo struct {
	slot  uint64
	root  [32]byte
	state *state.BeaconState
	lock  sync.RWMutex
}

// NewKVStore initializes a new boltDB key-value store at the directory
// path specified, creates the kv-buckets based on the schema, and stores
// an open connection db object as a property of the Store struct.
func NewKVStore(dirPath string, stateSummaryCache *StateSummaryCache) (*Store, error) {
	hasDir, err := fileutil.HasDir(dirPath)
	if err != nil {
		return nil, err
	}
	if !hasDir {
		if err := fileutil.MkdirAll(dirPath); err != nil {
			return nil, err
		}
	}
	datafile := path.Join(dirPath, DatabaseFileName)
	boltDB, err := bolt.Open(datafile, params.BeaconIoConfig().ReadWritePermissions, &bolt.Options{Timeout: 1 * time.Second, InitialMmapSize: 10e6})
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
		NumCounters: NumOfVotes,     // number of keys to track frequency of (1M).
		MaxCost:     VotesCacheSize, // maximum cost of cache (8MB).
		BufferItems: 64,             // number of keys per Get buffer.
	})
	if err != nil {
		return nil, err
	}

	kv := &Store{
		db:                      boltDB,
		databasePath:            dirPath,
		blockCache:              blockCache,
		validatorIndexCache:     validatorCache,
		stateSummaryCache:       stateSummaryCache,
		hotStateCache:           newHotStateCache(),
		finalizedInfo:           &finalizedInfo{slot: 0, root: params.BeaconConfig().ZeroHash},
		slotsPerArchivedPoint:   params.BeaconConfig().SlotsPerArchivedPoint,
		epochBoundaryStateCache: newBoundaryStateCache(),
		saveHotStateDB: &saveHotStateDbConfig{
			duration: defaultHotStateDBInterval,
		},
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
			// New State Management service bucket.
			newStateServiceCompatibleBucket,
			// Migrations
			migrationsBucket,
		)
	}); err != nil {
		return nil, err
	}

	err = prometheus.Register(createBoltCollector(kv.db))

	return kv, err
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
	return s.db.Close()
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

// createBoltCollector returns a prometheus collector specifically configured for boltdb.
func createBoltCollector(db *bolt.DB) prometheus.Collector {
	return prombolt.New("boltDB", db)
}

// Resume resumes a new state management object from previously saved finalized check point in DB.
func (s *Store) Resume(ctx context.Context) (*state.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "stateGen.Resume")
	defer span.End()

	c, err := s.FinalizedCheckpoint(ctx)
	if err != nil {
		return nil, err
	}
	fRoot := bytesutil.ToBytes32(c.Root)
	// Resume as genesis state if last finalized root is zero hashes.
	if fRoot == params.BeaconConfig().ZeroHash {
		return s.GenesisState(ctx)
	}
	fState, err := s.StateByRoot(ctx, fRoot)
	if err != nil {
		return nil, err
	}
	if fState == nil {
		return nil, errors.New("finalized state not found in disk")
	}

	s.finalizedInfo = &finalizedInfo{slot: fState.Slot(), root: fRoot, state: fState.Copy()}

	return fState, nil
}

// SaveFinalizedState saves the finalized slot, root and state into memory to be used by state gen service.
// This used for migration at the correct start slot and used for hot state play back to ensure
// lower bound to start is always at the last finalized state.
func (s *Store) SaveFinalizedState(fSlot uint64, fRoot [32]byte, fState *state.BeaconState) {
	s.finalizedInfo.lock.Lock()
	defer s.finalizedInfo.lock.Unlock()
	s.finalizedInfo.root = fRoot
	s.finalizedInfo.state = fState.Copy()
	s.finalizedInfo.slot = fSlot
}

// Returns true if input root equals to cached finalized root.
func (s *Store) isFinalizedRoot(r [32]byte) bool {
	s.finalizedInfo.lock.RLock()
	defer s.finalizedInfo.lock.RUnlock()
	return r == s.finalizedInfo.root
}

// Returns the cached and copied finalized state.
func (s *Store) finalizedState() *state.BeaconState {
	s.finalizedInfo.lock.RLock()
	defer s.finalizedInfo.lock.RUnlock()
	return s.finalizedInfo.state.Copy()
}
