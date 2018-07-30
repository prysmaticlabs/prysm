// Package database defines a beacon chain DB service that can be
// initialized with either a persistent db, or an in-memory kv-store.
package database

import (
	"fmt"
	"path/filepath"

	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "db")

// BeaconDB defines a service for the beacon chain system's persistent storage.
type BeaconDB struct {
	db ethdb.Database
}

// BeaconDBConfig specifies configuration options for the db service.
type BeaconDBConfig struct {
	DataDir  string
	Name     string
	InMemory bool
}

// NewBeaconDB initializes a beaconDB instance.
func NewBeaconDB(config *BeaconDBConfig) (*BeaconDB, error) {
	// Uses default cache and handles values.
	// TODO: allow these arguments to be set based on cli context.
	beaconDB := &BeaconDB{}
	if config.InMemory {
		beaconDB.db = NewKVStore()
		return beaconDB, nil
	}

	db, err := ethdb.NewLDBDatabase(filepath.Join(config.DataDir, config.Name), 16, 16)
	if err != nil {
		log.Error(fmt.Sprintf("Could not start beaconDB: %v", err))
		return nil, err
	}
	beaconDB.db = db

	return beaconDB, nil
}

// Close closes the database
func (b *BeaconDB) Close() {
	b.db.Close()
}

// DB returns the attached ethdb instance.
func (b *BeaconDB) DB() ethdb.Database {
	return b.db
}
