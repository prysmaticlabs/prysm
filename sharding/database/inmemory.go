// Package database provides several constructs including a simple in-memory database.
// This should not be used for production, but would be a helpful interim
// solution for development.
package database

import (
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/common"
)

// ShardKV is an in-memory mapping of hashes to RLP encoded values.
type ShardKV struct {
	kv   map[common.Hash][]byte
	lock sync.RWMutex
}

// NewShardKV initializes a keyval store in memory.
func NewShardKV() *ShardKV {
	return &ShardKV{kv: make(map[common.Hash][]byte)}
}

// Get fetches a val from the mappping by key.
func (sb *ShardKV) Get(k []byte) ([]byte, error) {
	sb.lock.RLock()
	defer sb.lock.RUnlock()
	v, ok := sb.kv[common.BytesToHash(k)]
	if !ok {
		return []byte{}, fmt.Errorf("key not found: %v", k)
	}
	return v, nil
}

// Has checks if the key exists in the mapping.
func (sb *ShardKV) Has(k []byte) (bool, error) {
	sb.lock.RLock()
	defer sb.lock.RUnlock()
	v := sb.kv[common.BytesToHash(k)]
	return v != nil, nil
}

// Put updates a key's value in the mapping.
func (sb *ShardKV) Put(k []byte, v []byte) error {
	sb.lock.Lock()
	defer sb.lock.Unlock()
	// there is no error in a simple setting of a value in a go map.
	sb.kv[common.BytesToHash(k)] = v
	return nil
}

// Delete removes the key and value from the mapping.
func (sb *ShardKV) Delete(k []byte) error {
	sb.lock.Lock()
	defer sb.lock.Unlock()
	// There is no return value for deleting a simple key in a go map.
	delete(sb.kv, common.BytesToHash(k))
	return nil
}
