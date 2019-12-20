package db

import "github.com/prysmaticlabs/prysm/beacon-chain/db/kv"

// NewDB initializes a new DB.
func NewDB(dirPath string) (Database, error) {
	return kv.NewKVStore(dirPath)
}
