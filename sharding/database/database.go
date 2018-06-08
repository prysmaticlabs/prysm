// Package database provides several constructs including a simple in-memory database.
// This should not be used for production, but would be a helpful interim
// solution for development.
package database

import (
	"path/filepath"

	"github.com/ethereum/go-ethereum/ethdb"
)

// NewShardDB initializes a shardDB that writes to local disk.
func NewShardDB(dataDir string, name string) (ethdb.Database, error) {
	// Uses default cache and handles values.
	// TODO: allow these arguments to be set based on cli context.
	return ethdb.NewLDBDatabase(filepath.Join(dataDir, name), 16, 16)
}
