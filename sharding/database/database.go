// Package database provides several constructs including a simple in-memory database.
// This should not be used for production, but would be a helpful interim
// solution for development.
package database

import (
	"fmt"
	"path/filepath"

	"github.com/ethereum/go-ethereum/ethdb"
	log "github.com/sirupsen/logrus"
)

type ShardDB struct {
	inmemory bool
	dataDir  string
	name     string
	cache    int
	handles  int
	db       ethdb.Database
}

// NewShardDB initializes a shardDB.
func NewShardDB(dataDir string, name string, inmemory bool) (*ShardDB, error) {
	// Uses default cache and handles values.
	// TODO: allow these arguments to be set based on cli context.
	if inmemory {
		return &ShardDB{
			inmemory: inmemory,
			dataDir:  dataDir,
			name:     name,
			cache:    16,
			handles:  16,
			db:       NewShardKV(),
		}, nil
	}
	return &ShardDB{
		dataDir: dataDir,
		name:    name,
		cache:   16,
		handles: 16,
		db:      nil,
	}, nil
}

// Start the shard DB service.
func (s *ShardDB) Start() {
	log.Info("Starting shardDB service")
	if !s.inmemory {
		db, err := ethdb.NewLDBDatabase(filepath.Join(s.dataDir, s.name), s.cache, s.handles)
		if err != nil {
			log.Error(fmt.Sprintf("Could not start shard DB: %v", err))
			return
		}
		s.db = db
	}
}

// Stop the shard DB service gracefully.
func (s *ShardDB) Stop() error {
	log.Info("Stopping shardDB service")
	s.db.Close()
	return nil
}

// DB returns the attached ethdb instance.
func (s *ShardDB) DB() ethdb.Database {
	return s.db
}
