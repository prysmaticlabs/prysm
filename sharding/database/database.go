package database

import (
	"path/filepath"

	"github.com/ethereum/go-ethereum/ethdb"
)

// ShardBackend defines an interface for a shardDB's necessary method
// signatures.
type ShardBackend interface {
	Get(k []byte) ([]byte, error)
	Has(k []byte) (bool, error)
	Put(k []byte, val []byte) error
	Delete(k []byte) error
}

// NewShardDB initializes a shardDB that writes to local disk.
func NewShardDB(dataDir string, name string) (ShardBackend, error) {
	// Uses default cache and handles values.
	// TODO: allow these arguments to be set based on cli context.
	return ethdb.NewLDBDatabase(filepath.Join(dataDir, name), 16, 16)
}
