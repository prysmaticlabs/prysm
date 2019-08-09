package db

import (
	"context"
	"os"
	"path"
	"sync"
	"time"

	"github.com/boltdb/bolt"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
)

var log = logrus.WithField("prefix", "beacondb")

// Database defines the necessary methods for Prysm's eth2 backend which may
// be implemented by any key-value or relational database in practice.
type Database interface {
	ClearDB() error
	Attestation(attRoot [32]byte) (*ethpb.Attestation, error)
	HasAttestation(attRoot [32]byte) bool
	SaveAttestation(ctx context.Context, att *ethpb.Attestation) error
	SaveAttestations(ctx context.Context, atts []*ethpb.Attestation) error
	Block(filter QueryFilter) (*ethpb.BeaconBlock, error)
	HasBlock(blockRoot [32]byte) bool
	ChildBlockRootsByParent(parentRoot [32]byte, filter QueryFilter) ([][]byte, error)
	SaveBlock(ctx context.Context, block *ethpb.BeaconBlock) error
	SaveBlocks(ctx context.Context, blocks []*ethpb.BeaconBlock) error
	LatestMessage(validatorIdx uint64) (*pb.LatestMessage, error)
	HasLatestMessage(validatorIdx uint64) bool
	SaveLatestMessage(ctx context.Context, validatorIdx uint64, msg *pb.LatestMessage) error
	State(ctx context.Context, filter QueryFilter) (*pb.BeaconState, error)
	HeadState(ctx context.Context) (*pb.BeaconState, error)
	SaveState(ctx context.Context, state *pb.BeaconState, blockRoot [32]byte) error
}

// QueryFilter defines a generic interface for type-asserting
// specific filters to use in querying DB objects.
type QueryFilter interface{}

// SlotFilter provides capabilities for filtering objects by slot.
type SlotFilter struct {
	startSlot uint64
	endSlot   uint64
}

// RootFilter provides capabilities for filtering objects by root.
type RootFilter struct {
	hash [32]byte
}

// EpochFilter provides capabilities for filtering objects by epoch.
type EpochFilter struct {
	startEpoch uint64
	endEpoch   uint64
}

// BeaconDB manages the data layer of the beacon chain implementation.
// The exposed methods do not have an opinion of the underlying data engine,
// but instead reflect the beacon chain logic.
// For example, instead of defining get, put, remove
// This defines methods such as getBlock, saveBlocksAndAttestations, etc.
type BeaconDB struct {
	// state objects and caches
	stateLock         sync.RWMutex
	serializedState   []byte
	stateHash         [32]byte
	validatorRegistry []*ethpb.Validator
	validatorBalances []uint64
	db                *bolt.DB
	DatabasePath      string

	// Beacon block info in memory.
	highestBlockSlot uint64
	// We keep a map of hashes of blocks which failed processing for blacklisting.
	badBlockHashes map[[32]byte]bool
	badBlocksLock  sync.RWMutex
	blocks         map[[32]byte]*ethpb.BeaconBlock
	blocksLock     sync.RWMutex

	// Beacon chain deposits in memory.
	pendingDeposits       []*DepositContainer
	deposits              []*DepositContainer
	depositsLock          sync.RWMutex
	chainstartPubkeys     map[string]bool
	chainstartPubkeysLock sync.RWMutex
}

// Close closes the underlying boltdb database.
func (db *BeaconDB) Close() error {
	return db.db.Close()
}

func (db *BeaconDB) update(fn func(*bolt.Tx) error) error {
	return db.db.Update(fn)
}
func (db *BeaconDB) batch(fn func(*bolt.Tx) error) error {
	return db.db.Batch(fn)
}
func (db *BeaconDB) view(fn func(*bolt.Tx) error) error {
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

// NewDB initializes a new DB. If the genesis block and states do not exist, this method creates it.
func NewDB(dirPath string) (*BeaconDB, error) {
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

	db := &BeaconDB{db: boltDB, DatabasePath: dirPath}
	db.blocks = make(map[[32]byte]*ethpb.BeaconBlock)

	if err := db.update(func(tx *bolt.Tx) error {
		return createBuckets(tx, blockBucket, attestationBucket, attestationTargetBucket, mainChainBucket,
			histStateBucket, chainInfoBucket, cleanupHistoryBucket, blockOperationsBucket, validatorBucket)
	}); err != nil {
		return nil, err
	}

	return db, err
}

// ClearDB removes the previously stored directory at the data directory.
func ClearDB(dirPath string) error {
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		return nil
	}
	return os.RemoveAll(dirPath)
}
