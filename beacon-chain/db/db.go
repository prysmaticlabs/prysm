// Package db defines the ability to create a new database
// for an eth2 beacon node.
package db

import (
	"github.com/prysmaticlabs/prysm/beacon-chain/db/kv"
)

// NewDB initializes a new DB.
func NewDB(dirPath string) (Database, error) {
	return kv.NewKVStore(dirPath)
}
