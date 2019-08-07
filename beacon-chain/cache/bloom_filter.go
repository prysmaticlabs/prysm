package cache

import (
	"sync"
	"encoding/binary"
	
	"github.com/willf/bloom"
)

// ActiveIndicesBloomFilter defines BloomFilter for storing and twsting of active indices
// Bloom Filter is a space-efficient probabilistic data structure 
// that is used to test whether an element is a member of a set
type ActiveIndicesBloomFilter struct {
	filter *bloom.BloomFilter
	lock sync.RWMutex
}

// A Bloom filter has two parameters: m - maximum size, k -the number of hashing functions on elements of the set
const (
	m = 300000
	k = 5
)

// NewBloomFiltercreates a new Bloom Filter for storing and testing active indices
func NewBloomFilter() *ActiveIndicesBloomFilter {
	return &ActiveIndicesBloomFilter{filter : bloom.New(m, k)}
}

// AddActiveIndicesBloomFilter adds a byte representation of active indices to a bloom filter
func (bf *ActiveIndicesBloomFilter) AddActiveIndicesBloomFilter(byteIndices [][]byte)  {
	bf.lock.Lock()
	defer bf.lock.Unlock()
	for _, i := range byteIndices {
		bf.filter.Add(i)
	}
}

// TestActiveIndicesBloomFilter tests makes a membership check if active indices are in bloom filter
func (bf *ActiveIndicesBloomFilter) TestActiveIndicesBloomFilter(byteIndices [][]byte) bool {
	bf.lock.Lock()
	defer bf.lock.Unlock()
	for _, i := range byteIndices {
		if !bf.filter.Test(i) {
			return false
		}
	}
	return true
}

// ClearBloomFilter clears keys of bloom filter
func (bf *ActiveIndicesBloomFilter) ClearBloomFilter()  {
	bf.filter = bf.filter.ClearAll()
}

// convertUint64ToByteSlice converts active indices to a byte representation
func convertUint64ToByteSlice(activeIndices []uint64) [][]byte {
	byteIndices := make([][]byte, 0, len(activeIndices))
	for _, i := range activeIndices {
		n1 := make([]byte,8)
		binary.BigEndian.PutUint64(n1,i)
		byteIndices = append(byteIndices, n1)
	}
	return byteIndices
}


















