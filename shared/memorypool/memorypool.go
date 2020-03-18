package memorypool

import (
	"sync"

	"github.com/prysmaticlabs/prysm/shared/featureconfig"
)

// DoubleByteSlicePool represents the memory pool
// for 2d byte slices.
var DoubleByteSlicePool = new(sync.Pool)

// BlockRootsMemoryPool represents the memory pool
// for block roots trie.
var BlockRootsMemoryPool = new(sync.Pool)

// StateRootsMemoryPool represents the memory pool
// for state roots trie.
var StateRootsMemoryPool = new(sync.Pool)

// RandaoMixesMemoryPool represents the memory pool
// for randao mixes trie.
var RandaoMixesMemoryPool = new(sync.Pool)

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
	byteSlice := rawObj.([][]byte)
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

// GetBlockRootsTrie retrieves the 3d byte trie of
// the desired size from the memory pool.
func GetBlockRootsTrie(size int) [][]*[32]byte {
	if !featureconfig.Get().EnableByteMempool {
		return make([][]*[32]byte, size)
	}

	rawObj := BlockRootsMemoryPool.Get()
	if rawObj == nil {
		return make([][]*[32]byte, size)
	}
	byteSlice := rawObj.([][]*[32]byte)
	if len(byteSlice) >= size {
		return byteSlice[:size]
	}
	return append(byteSlice, make([][]*[32]byte, size-len(byteSlice))...)
}

// PutBlockRootsTrie places the provided 3d byte trie
// in the memory pool.
func PutBlockRootsTrie(data [][]*[32]byte) {
	if featureconfig.Get().EnableByteMempool {
		BlockRootsMemoryPool.Put(data)
	}
}

// GetStateRootsTrie retrieves the 3d byte slice of
// the desired size from the memory pool.
func GetStateRootsTrie(size int) [][]*[32]byte {
	if !featureconfig.Get().EnableByteMempool {
		return make([][]*[32]byte, size)
	}

	rawObj := BlockRootsMemoryPool.Get()
	if rawObj == nil {
		return make([][]*[32]byte, size)
	}
	byteSlice := rawObj.([][]*[32]byte)
	if len(byteSlice) >= size {
		return byteSlice[:size]
	}
	return append(byteSlice, make([][]*[32]byte, size-len(byteSlice))...)
}

// PutStateRootsTrie places the provided trie
// in the memory pool.
func PutStateRootsTrie(data [][]*[32]byte) {
	if featureconfig.Get().EnableByteMempool {
		StateRootsMemoryPool.Put(data)
	}
}

// GetRandaoMixesTrie retrieves the 3d byte slice of
// the desired size from the memory pool.
func GetRandaoMixesTrie(size int) [][]*[32]byte {
	if !featureconfig.Get().EnableByteMempool {
		return make([][]*[32]byte, size)
	}

	rawObj := StateRootsMemoryPool.Get()
	if rawObj == nil {
		return make([][]*[32]byte, size)
	}
	byteSlice := rawObj.([][]*[32]byte)
	if len(byteSlice) >= size {
		return byteSlice[:size]
	}
	return append(byteSlice, make([][]*[32]byte, size-len(byteSlice))...)
}

// PutRandaoMixesTrie places the provided 3d byte slice
// in the memory pool.
func PutRandaoMixesTrie(data [][]*[32]byte) {
	if featureconfig.Get().EnableByteMempool {
		StateRootsMemoryPool.Put(data)
	}
}
