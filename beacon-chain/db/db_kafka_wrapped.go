// +build kafka_enabled

package db

import (
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/kafka"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/kv"
)

// NewDB initializes a new DB with kafka wrapper.
func NewDB(dirPath string, stateSummaryCache *cache.StateSummaryCache) (Database, error) {
	db, err := kv.NewKVStore(dirPath, stateSummaryCache)
	if err != nil {
		return nil, err
	}

	return kafka.Wrap(db)
}
