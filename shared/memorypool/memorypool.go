package memorypool

import (
	"sync"

	"github.com/prysmaticlabs/prysm/shared/featureconfig"
)

// DoubleByteSlicePool represents the memory pool
// for 2d byte slices.
var DoubleByteSlicePool = new(sync.Pool)

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
