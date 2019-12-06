package db

import (
	"github.com/prysmaticlabs/prysm/beacon-chain/db/iface"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/kafka"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/kv"
)

// Database defines the necessary methods for Prysm's eth2 backend which may
// be implemented by any key-value or relational database in practice.
type Database = iface.Database

// NewDB initializes a new DB.
func NewDB(dirPath string) (Database, error) {
	db, err := kv.NewKVStore(dirPath)
	if err != nil {
		return nil, err
	}

	return kafka.Wrap(db)
}
