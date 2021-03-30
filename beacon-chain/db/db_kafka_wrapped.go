// +build kafka_enabled

package db

import (
	"github.com/prysmaticlabs/prysm/beacon-chain/db/kafka"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/kv"
)

// NewDB initializes a new DB with kafka wrapper.
func NewDB(dirPath string, stateSummaryCache *kv.stateSummaryCache) (Database, error) {
	db, err := kv.NewKVStore(dirPath, stateSummaryCache, &kv.Config{})
	if err != nil {
		return nil, err
	}

	return kafka.Wrap(db)
}
