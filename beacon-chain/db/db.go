package db

import (
	"context"
	"os"
	"path"
	"sync"
	"time"

	"github.com/boltdb/bolt"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache/depositcache"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/kv"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "beacondb")

// Database defines the necessary methods for Prysm's eth2 backend which may
// be implemented by any key-value or relational database in practice.
type Database interface {
	ClearDB() error
	Attestation(ctx context.Context, attRoot [32]byte) (*ethpb.Attestation, error)
	Attestations(ctx context.Context, f *filters.QueryFilter) ([]*ethpb.Attestation, error)
	HasAttestation(ctx context.Context, attRoot [32]byte) bool
	DeleteAttestation(ctx context.Context, attRoot [32]byte) error
	SaveAttestation(ctx context.Context, att *ethpb.Attestation) error
	SaveAttestations(ctx context.Context, atts []*ethpb.Attestation) error
	Block(ctx context.Context, blockRoot [32]byte) (*ethpb.BeaconBlock, error)
	HeadBlock(ctx context.Context) (*ethpb.BeaconBlock, error)
	Blocks(ctx context.Context, f *filters.QueryFilter) ([]*ethpb.BeaconBlock, error)
	BlockRoots(ctx context.Context, f *filters.QueryFilter) ([][]byte, error)
	HasBlock(ctx context.Context, blockRoot [32]byte) bool
	DeleteBlock(ctx context.Context, blockRoot [32]byte) error
	SaveBlock(ctx context.Context, block *ethpb.BeaconBlock) error
	SaveBlocks(ctx context.Context, blocks []*ethpb.BeaconBlock) error
	SaveHeadBlockRoot(ctx context.Context, blockRoot [32]byte) error
	ValidatorLatestVote(ctx context.Context, validatorIdx uint64) (*pb.ValidatorLatestVote, error)
	HasValidatorLatestVote(ctx context.Context, validatorIdx uint64) bool
	SaveValidatorLatestVote(ctx context.Context, validatorIdx uint64, vote *pb.ValidatorLatestVote) error
	State(ctx context.Context, blockRoot [32]byte) (*pb.BeaconState, error)
	HeadState(ctx context.Context) (*pb.BeaconState, error)
	SaveState(ctx context.Context, state *pb.BeaconState, blockRoot [32]byte) error
	ValidatorIndex(ctx context.Context, publicKey [48]byte) (uint64, bool, error)
	HasValidatorIndex(ctx context.Context, publicKey [48]byte) bool
	DeleteValidatorIndex(ctx context.Context, publicKey [48]byte) error
	SaveValidatorIndex(ctx context.Context, publicKey [48]byte, validatorIdx uint64) error
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
	DepositCache *depositcache.DepositCache
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

// NewDBDeprecated initializes a new DB. If the genesis block and states do not exist, this method creates it.
func NewDBDeprecated(dirPath string) (*BeaconDB, error) {
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

	depCache := depositcache.NewDepositCache()

	db := &BeaconDB{db: boltDB, DatabasePath: dirPath, DepositCache: depCache}
	db.blocks = make(map[[32]byte]*ethpb.BeaconBlock)

	if err := db.update(func(tx *bolt.Tx) error {
		return createBuckets(tx, blockBucket, attestationBucket, attestationTargetBucket, mainChainBucket,
			histStateBucket, chainInfoBucket, cleanupHistoryBucket, blockOperationsBucket, validatorBucket)
	}); err != nil {
		return nil, err
	}

	return db, err
}

// NewDB initializes a new DB.
func NewDB(dirPath string) (Database, error) {
	return kv.NewKVStore(dirPath)
}

// ClearDB removes the previously stored directory at the data directory.
func ClearDB(dirPath string) error {
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		return nil
	}
	return os.RemoveAll(dirPath)
}
