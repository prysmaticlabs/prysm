package database

import (
	"path/filepath"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/micro/cli"
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
func NewShardDB(ctx *cli.Context, name string) (*ethdb.LDBDatabase, error) {

	dataDir := ""
	path := filepath.Join(dataDir, name)

	// Uses default cache and handles values.
	// TODO: allow these to be set based on cli context.
	// TODO: fix interface - lots of methods do not match.
	return ethdb.NewLDBDatabase(path, 16, 16)
}
