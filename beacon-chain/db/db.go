// Package db defines the ability to create a new database
// for an eth2 beacon node.
package db

import (
	"context"

	"github.com/prysmaticlabs/prysm/beacon-chain/db/kv"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/slasherkv"
)

// NewDB initializes a new DB.
func NewDB(ctx context.Context, dirPath string, config *kv.Config) (Database, error) {
	return kv.NewKVStore(ctx, dirPath, config)
}

// NewDBFilename uses the KVStoreDatafilePath so that if this layer of
// indirection between db.NewDB->kv.NewKVStore ever changes, it will be easy to remember
// to also change this filename indirection at the same time.
func NewDBFilename(dirPath string) string {
	return kv.KVStoreDatafilePath(dirPath)
}

// NewSlasherDB initializes a new DB for slasher.
func NewSlasherDB(ctx context.Context, dirPath string, config *slasherkv.Config) (SlasherDatabase, error) {
	return slasherkv.NewKVStore(ctx, dirPath, config)
}
