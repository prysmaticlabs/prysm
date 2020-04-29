// Package db defines a persistent backend for the slasher service.
package db

import (
	"github.com/prysmaticlabs/prysm/slasher/db/kv"
)

// NewDB initializes a new DB.
func NewDB(dirPath string, cfg *kv.Config) (*kv.Store, error) {
	return kv.NewKVStore(dirPath, cfg)
}
