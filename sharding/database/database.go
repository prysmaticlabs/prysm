// Package database provides several constructs including a simple in-memory database.
// This should not be used for production, but would be a helpful interim
// solution for development.
package database

import (
	"path/filepath"

	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
)

type ShardDB struct {
	db *ethdb.LDBDatabase
}

// NewShardDB initializes a shardDB that writes to local disk.
func NewShardDB(dataDir string, name string) (*ShardDB, error) {
	// Uses default cache and handles values.
	// TODO: allow these arguments to be set based on cli context.

	db, err := ethdb.NewLDBDatabase(filepath.Join(dataDir, name), 16, 16)
	if err != nil {
		return nil, err
	}

	return &ShardDB{
		db: db,
	}, nil
}

// Start the shard DB service.
func (s *ShardDB) Start() {
	log.Info("Starting shardDB service")
}

// Stop the shard DB service gracefully.
func (s *ShardDB) Stop() error {
	log.Info("Stopping shardDB service")
	s.db.Close()
	return nil
}
