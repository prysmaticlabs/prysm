package db

import (
	"github.com/prysmaticlabs/prysm/beacon-chain/db/kv"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/iface"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/kafka"
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

	// TODO: if exporter is enabled, wrap! Add flag.
	return kafka.Wrap(db)
}
