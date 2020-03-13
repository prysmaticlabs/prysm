package memorypool

import (
	"sync"

	"github.com/prysmaticlabs/prysm/shared/featureconfig"
)

// DoubleByteSlicePool represents the memory pool
// for 2d byte slices.
var DoubleByteSlicePool = new(sync.Pool)

var TripleByteSliceBlockRootsPool = new(sync.Pool)

var TripleByteSliceStateRootsPool = new(sync.Pool)

var TripleByteSliceRandaoMixesPool = new(sync.Pool)

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

// GetTripleByteSliceRoots retrieves the 2d byte slice of
// the desired size from the memory pool.
func GetTripleByteSliceBlockRoots(size int) [][]*[32]byte {
	if !featureconfig.Get().EnableByteMempool {
		return make([][]*[32]byte, size)
	}

	rawObj := TripleByteSliceBlockRootsPool.Get()
	if rawObj == nil {
		return make([][]*[32]byte, size)
	}
	byteSlice := rawObj.([][]*[32]byte)
	if len(byteSlice) >= size {
		return byteSlice[:size]
	}
	return append(byteSlice, make([][]*[32]byte, size-len(byteSlice))...)
}

// PutTripleByteSliceRoots places the provided 2d byte slice
// in the memory pool.
func PutTripleByteSliceBlockRoots(data [][]*[32]byte) {
	if featureconfig.Get().EnableByteMempool {
		TripleByteSliceBlockRootsPool.Put(data)
	}
}

// GetTripleByteSliceRoots retrieves the 2d byte slice of
// the desired size from the memory pool.
func GetTripleByteSliceStateRoots(size int) [][]*[32]byte {
	if !featureconfig.Get().EnableByteMempool {
		return make([][]*[32]byte, size)
	}

	rawObj := TripleByteSliceStateRootsPool.Get()
	if rawObj == nil {
		return make([][]*[32]byte, size)
	}
	byteSlice := rawObj.([][]*[32]byte)
	if len(byteSlice) >= size {
		return byteSlice[:size]
	}
	return append(byteSlice, make([][]*[32]byte, size-len(byteSlice))...)
}

// PutTripleByteSliceRoots places the provided 2d byte slice
// in the memory pool.
func PutTripleByteSliceStateRoots(data [][]*[32]byte) {
	if featureconfig.Get().EnableByteMempool {
		TripleByteSliceStateRootsPool.Put(data)
	}
}

// GetTripleByteSliceRoots retrieves the 2d byte slice of
// the desired size from the memory pool.
func GetTripleByteSliceRandaoMixes(size int) [][]*[32]byte {
	if !featureconfig.Get().EnableByteMempool {
		return make([][]*[32]byte, size)
	}

	rawObj := TripleByteSliceRandaoMixesPool.Get()
	if rawObj == nil {
		return make([][]*[32]byte, size)
	}
	byteSlice := rawObj.([][]*[32]byte)
	if len(byteSlice) >= size {
		return byteSlice[:size]
	}
	return append(byteSlice, make([][]*[32]byte, size-len(byteSlice))...)
}

// PutTripleByteSliceRoots places the provided 2d byte slice
// in the memory pool.
func PutTripleByteSliceRandaoMixes(data [][]*[32]byte) {
	if featureconfig.Get().EnableByteMempool {
		TripleByteSliceRandaoMixesPool.Put(data)
	}
}
