// Package memorypool includes useful tools for creating common
// data structures in eth2 with optimal memory allocation.
package memorypool

import (
	"sync"

	"github.com/prysmaticlabs/prysm/shared/featureconfig"
)

// DoubleByteSlicePool represents the memory pool
// for 2d byte slices.
var DoubleByteSlicePool = new(sync.Pool)

// RootsMemoryPool represents the memory pool
// for state roots/block roots trie.
var RootsMemoryPool = new(sync.Pool)

// RandaoMixesMemoryPool represents the memory pool
// for randao mixes trie.
var RandaoMixesMemoryPool = new(sync.Pool)

// ValidatorsMemoryPool represents the memory pool
// for 3d byte slices.
var ValidatorsMemoryPool = new(sync.Pool)

// GetDoubleByteSlice retrieves the 2d byte slice of
// the desired size from the memory pool.
func GetDoubleByteSlice(size int) [][]byte {
	if !featureconfig.Get().EnableByteMempool {
		return make([][]byte, size)
	}

	rawObj := DoubleByteSlicePool.Get()
	if rawObj == nil {
		return make([][]byte, size)
	}
	byteSlice, ok := rawObj.([][]byte)
	if !ok {
		return nil
	}
	if len(byteSlice) >= size {
		return byteSlice[:size]
	}
	return append(byteSlice, make([][]byte, size-len(byteSlice))...)
}

// PutDoubleByteSlice places the provided 2d byte slice
// in the memory pool.
func PutDoubleByteSlice(data [][]byte) {
	if featureconfig.Get().EnableByteMempool {
		DoubleByteSlicePool.Put(data)
	}
}

// GetRootsTrie retrieves the 3d byte trie of
// the desired size from the memory pool.
func GetRootsTrie(size int) [][]*[32]byte {
	if !featureconfig.Get().EnableByteMempool {
		return make([][]*[32]byte, size)
	}

	rawObj := RootsMemoryPool.Get()
	if rawObj == nil {
		return make([][]*[32]byte, size)
	}
	byteSlice, ok := rawObj.([][]*[32]byte)
	if !ok {
		return nil
	}
	if len(byteSlice) >= size {
		return byteSlice[:size]
	}
	return append(byteSlice, make([][]*[32]byte, size-len(byteSlice))...)
}

// PutRootsTrie places the provided 3d byte trie
// in the memory pool.
func PutRootsTrie(data [][]*[32]byte) {
	if featureconfig.Get().EnableByteMempool {
		RootsMemoryPool.Put(data)
	}
}

// GetRandaoMixesTrie retrieves the 3d byte slice of
// the desired size from the memory pool.
func GetRandaoMixesTrie(size int) [][]*[32]byte {
	if !featureconfig.Get().EnableByteMempool {
		return make([][]*[32]byte, size)
	}

	rawObj := RandaoMixesMemoryPool.Get()
	if rawObj == nil {
		return make([][]*[32]byte, size)
	}
	byteSlice, ok := rawObj.([][]*[32]byte)
	if !ok {
		return nil
	}
	if len(byteSlice) >= size {
		return byteSlice[:size]
	}
	return append(byteSlice, make([][]*[32]byte, size-len(byteSlice))...)
}

// PutRandaoMixesTrie places the provided 3d byte slice
// in the memory pool.
func PutRandaoMixesTrie(data [][]*[32]byte) {
	if featureconfig.Get().EnableByteMempool {
		RandaoMixesMemoryPool.Put(data)
	}
}

// GetValidatorsTrie retrieves the 3d byte slice of
// the desired size from the memory pool.
func GetValidatorsTrie(size int) [][]*[32]byte {
	if !featureconfig.Get().EnableByteMempool {
		return make([][]*[32]byte, size)
	}
	rawObj := ValidatorsMemoryPool.Get()
	if rawObj == nil {
		return make([][]*[32]byte, size)
	}
	byteSlice, ok := rawObj.([][]*[32]byte)
	if !ok {
		return nil
	}
	if len(byteSlice) >= size {
		return byteSlice[:size]
	}
	return append(byteSlice, make([][]*[32]byte, size-len(byteSlice))...)
}

// PutValidatorsTrie places the provided 3d byte slice
// in the memory pool.
func PutValidatorsTrie(data [][]*[32]byte) {
	if featureconfig.Get().EnableByteMempool {
		ValidatorsMemoryPool.Put(data)
	}
}
