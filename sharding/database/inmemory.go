// Package database provides several constructs including a simple in-memory database.
// This should not be used for production, but would be a helpful interim
// solution for development.
package database

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

// ShardKV is an in-memory mapping of hashes to RLP encoded values.
type ShardKV struct {
	kv map[common.Hash][]byte
}

// NewShardKV initializes a keyval store in memory.
func NewShardKV() *ShardKV {
	return &ShardKV{kv: make(map[common.Hash][]byte)}
}

// Get fetches a val from the mappping by key.
func (sb *ShardKV) Get(k common.Hash) ([]byte, error) {
	v, ok := sb.kv[k]
	if !ok {
		return nil, fmt.Errorf("key not found: %v", k)
	}
	return v, nil
}

// Has checks if the key exists in the mapping.
func (sb *ShardKV) Has(k common.Hash) bool {
	v := sb.kv[k]
	return v != nil
}

// Put updates a key's value in the mapping.
func (sb *ShardKV) Put(k common.Hash, v []byte) error {
	// there is no error in a simple setting of a value in a go map.
	sb.kv[k] = v
	return nil
}

// Delete removes the key and value from the mapping.
func (sb *ShardKV) Delete(k common.Hash) error {
	// There is no return value for deleting a simple key in a go map.
	delete(sb.kv, k)
	return nil
}
