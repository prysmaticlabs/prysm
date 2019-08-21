package kv

import (
	"os"
	"path"
	"sync"
	"time"

	"github.com/boltdb/bolt"
	"github.com/pkg/errors"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
)

// Store defines an implementation of the Prysm Database interface
// using BoltDB as the underlying persistent kv-store for eth2.
type Store struct {
	db           *bolt.DB
	databasePath string

	// Caching layer properties.
	blocksLock  sync.RWMutex
	votesLock   sync.RWMutex
	blocks      map[[32]byte]*ethpb.BeaconBlock
	latestVotes map[uint64]*pb.ValidatorLatestVote
}

// NewKVStore initializes a new boltDB key-value store at the directory
// path specified, creates the kv-buckets based on the schema, and stores
// an open connection db object as a property of the Store struct.
func NewKVStore(dirPath string) (*Store, error) {
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

	kv := &Store{
		db:           boltDB,
		databasePath: dirPath,
		blocks:       make(map[[32]byte]*ethpb.BeaconBlock),
		latestVotes:  make(map[uint64]*pb.ValidatorLatestVote),
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
			// Indices buckets.
			attestationShardIndicesBucket,
			attestationParentRootIndicesBucket,
			attestationStartEpochIndicesBucket,
			attestationEndEpochIndicesBucket,
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
