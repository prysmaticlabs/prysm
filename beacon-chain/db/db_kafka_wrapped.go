package db

import (
	"github.com/prysmaticlabs/prysm/beacon-chain/db/kafka"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/kv"
)

// NewDB initializes a new DB with kafka wrapper.
func NewDB(dirPath string) (Database, error) {
	db, err := kv.NewKVStore(dirPath)
	if err != nil {
		return nil, err
	}

	return kafka.Wrap(db)
}
