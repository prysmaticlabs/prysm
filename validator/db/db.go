package db

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/boltdb/bolt"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/go-bitfield"
	slashpb "github.com/prysmaticlabs/prysm/proto/slashing"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/validator/db/iface"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "db")

var _ = iface.ValidatorDB(&Store{})

var databaseFileName = "validator.db"

// Store defines an implementation of the Prysm Database interface
// using BoltDB as the underlying persistent kv-store for eth2.
type Store struct {
	db           *bolt.DB
	databasePath string
}

// Close closes the underlying boltdb database.
func (db *Store) Close() error {
	return db.db.Close()
}

func (db *Store) update(fn func(*bolt.Tx) error) error {
	return db.db.Update(fn)
}
func (db *Store) batch(fn func(*bolt.Tx) error) error {
	return db.db.Batch(fn)
}
func (db *Store) view(fn func(*bolt.Tx) error) error {
	return db.db.View(fn)
}

// ClearDB removes any previously stored data at the configured data directory.
func (db *Store) ClearDB() error {
	if _, err := os.Stat(db.databasePath); os.IsNotExist(err) {
		return nil
	}
	return os.Remove(filepath.Join(db.databasePath, databaseFileName))
}

// DatabasePath at which this database writes files.
func (db *Store) DatabasePath() string {
	return db.databasePath
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
func NewKVStore(dirPath string, pubkeys [][48]byte) (*Store, error) {
	if err := os.MkdirAll(dirPath, 0700); err != nil {
		return nil, err
	}
	datafile := filepath.Join(dirPath, databaseFileName)
	boltDB, err := bolt.Open(datafile, 0600, &bolt.Options{Timeout: 1 * time.Second})
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

	// Initialize the required pubkeys into the DB to ensure they're not empty.
	for _, pubkey := range pubkeys {
		proHistory, err := kv.ProposalHistory(context.Background(), pubkey[:])
		if err != nil {
			return nil, err
		}
		if proHistory == nil {
			cleanHistory := &slashpb.ProposalHistory{
				EpochBits: bitfield.NewBitlist(params.BeaconConfig().WeakSubjectivityPeriod),
			}
			if err := kv.SaveProposalHistory(context.Background(), pubkey[:], cleanHistory); err != nil {
				return nil, err
			}
		}

		attHistory, err := kv.AttestationHistory(context.Background(), pubkey[:])
		if err != nil {
			return nil, err
		}
		if attHistory == nil {
			newMap := make(map[uint64]uint64)
			newMap[0] = params.BeaconConfig().FarFutureEpoch
			cleanHistory := &slashpb.AttestationHistory{
				TargetToSource: newMap,
			}
			if err := kv.SaveAttestationHistory(context.Background(), pubkey[:], cleanHistory); err != nil {
				return nil, err
			}
		}

	}

	return kv, err
}

// Size returns the db size in bytes.
func (db *Store) Size() (int64, error) {
	var size int64
	err := db.db.View(func(tx *bolt.Tx) error {
		size = tx.Size()
		return nil
	})
	return size, err
}
