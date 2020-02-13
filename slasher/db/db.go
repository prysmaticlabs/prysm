package db

import (
	"github.com/prysmaticlabs/prysm/slasher/db/kv"
)

// NewSlasherDB initializes a new DB.
func NewSlasherDB(dirPath string, cfg *kv.Config) (*kv.Store, error) {
	return kv.NewKVStore(dirPath, cfg)
}
