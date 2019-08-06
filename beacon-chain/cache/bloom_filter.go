package cache

import (
	"sync"
	"encoding/binary"
	
	"github.com/willf/bloom"
)


type ActiveIndicesBloomFilter struct {
	filter *bloom.BloomFilter
	lock sync.RWMutex
}

const (
	m = 300000
	k = 5
)


func NewBloomFilter() *ActiveIndicesBloomFilter {
	return &ActiveIndicesBloomFilter{filter : bloom.New(m, k)}
}


func (bf *ActiveIndicesBloomFilter) AddActiveIndicesBloomFilter(byteIndices [][]byte)  {
	bf.lock.Lock()
	defer bf.lock.Unlock()
	for _, i := range byteIndices {
		bf.filter.Add(i)
	}
}


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

func (bf *ActiveIndicesBloomFilter) ClearBloomFilter()  {
	bf.filter = bf.filter.ClearAll()
}

func convertUint64ToByteSlice(activeIndices []uint64) [][]byte {
	byteIndices := make([][]byte, 0, len(activeIndices))
	for _, i := range activeIndices {
		n1 := make([]byte,8)
		binary.BigEndian.PutUint64(n1,i)
		byteIndices = append(byteIndices, n1)
	}
	return byteIndices
}


















