// Package database defines a beacon chain DB service that can be
// initialized with either a persistent db, or an in-memory kv-store.
package database

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/ethereum/go-ethereum/ethdb"
	sharedDB "github.com/prysmaticlabs/geth-sharding/shared/database"
	log "github.com/sirupsen/logrus"
)

// BeaconDB defines a service for the beacon chain system's persistent storage.
type BeaconDB struct {
	ctx      context.Context
	cancel   context.CancelFunc
	inmemory bool
	dataDir  string
	name     string
	cache    int
	handles  int
	db       ethdb.Database
}

// NewBeaconDB initializes a beaconDB instance.
func NewBeaconDB(ctx context.Context, dataDir string, name string, inmemory bool) (*BeaconDB, error) {
	// Uses default cache and handles values.
	// TODO: allow these arguments to be set based on cli context.
	ctx, cancel := context.WithCancel(ctx)
	beaconDB := &BeaconDB{
		ctx:     ctx,
		cancel:  cancel,
		name:    name,
		dataDir: dataDir,
	}
	if inmemory {
		beaconDB.inmemory = true
		beaconDB.db = sharedDB.NewKVStore()
	} else {
		beaconDB.inmemory = false
		beaconDB.cache = 16
		beaconDB.handles = 16
	}
	return beaconDB, nil
}

// Start the beacon DB service.
func (b *BeaconDB) Start() {
	log.Info("Starting beaconDB service")
	if !b.inmemory {
		db, err := ethdb.NewLDBDatabase(filepath.Join(b.dataDir, b.name), b.cache, b.handles)
		if err != nil {
			log.Error(fmt.Sprintf("Could not start beaconDB: %v", err))
			return
		}
		b.db = db
	}
}

// Stop the beaconDB service gracefully.
func (b *BeaconDB) Stop() error {
	log.Info("Stopping shardDB service")
	b.db.Close()
	b.cancel()
	return nil
}

// DB returns the attached ethdb instance.
func (b *BeaconDB) DB() ethdb.Database {
	return b.db
}
