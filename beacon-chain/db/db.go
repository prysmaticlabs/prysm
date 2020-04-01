package db

import (
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/kv"
)

// NewDB initializes a new DB.
func NewDB(dirPath string, stateSummaryCache *cache.StateSummaryCache) (Database, error) {
	return kv.NewKVStore(dirPath, stateSummaryCache)
}
