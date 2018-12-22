package database

import (
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
)

// KVStore is an in-memory mapping of hashes to RLP encoded values.
type KVStore struct {
	kv   map[common.Hash][]byte
	lock sync.RWMutex
}

// NewKVStore creates an in-memory, key-value store.
func NewKVStore() *KVStore {
	return &KVStore{kv: make(map[common.Hash][]byte)}
}

// Get fetches a val from the mappping by key.
func (s *KVStore) Get(k []byte) ([]byte, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	v, ok := s.kv[common.BytesToHash(k)]
	if !ok {
		return []byte{}, fmt.Errorf("key not found: %v", k)
	}
	return v, nil
}

// Has checks if the key exists in the mapping.
func (s *KVStore) Has(k []byte) (bool, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	v := s.kv[common.BytesToHash(k)]
	return v != nil, nil
}

// Put updates a key's value in the mapping.
func (s *KVStore) Put(k []byte, v []byte) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	// there is no error in a simple setting of a value in a go map.
	s.kv[common.BytesToHash(k)] = v
	return nil
}

// Delete removes the key and value from the mapping.
func (s *KVStore) Delete(k []byte) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	// There is no return value for deleting a simple key in a go map.
	delete(s.kv, common.BytesToHash(k))
	return nil
}

// Close satisfies ethdb.Database.
func (s *KVStore) Close() {
	log.Debug("ShardKV Close() isnt implemented yet")
}

// NewBatch satisfies ethdb.Database.
func (s *KVStore) NewBatch() ethdb.Batch {
	log.Debug("ShardKV NewBatch() isnt implemented yet")
	return nil
}
