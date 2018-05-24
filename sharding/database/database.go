package database

import (
	"path/filepath"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
)

// ShardBackend defines an interface for a shardDB's necessary method
// signatures.
type ShardBackend interface {
	Get(k common.Hash) (*[]byte, error)
	Has(k common.Hash) bool
	Put(k common.Hash, val []byte) error
	Delete(k common.Hash) error
}

// NewShardDB initializes a shardDB that writes to local disk.
// TODO: make it return ShardBackend but modify interface methods.
func NewShardDB(dataDir string, name string) (*ethdb.LDBDatabase, error) {
	// Uses default cache and handles values.
	// TODO: allow these to be set based on cli context.
	// TODO: fix interface - lots of methods do not match.
	return ethdb.NewLDBDatabase(filepath.Join(dataDir, name), 16, 16)
}
