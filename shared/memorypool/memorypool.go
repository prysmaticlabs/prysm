package memorypool

import "sync"

var DoubleByteSlicePool = new(sync.Pool)

func GetDoubleByteSlice(size int) [][]byte {
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

func PutDoubleByteSlice(data [][]byte) {
	DoubleByteSlicePool.Put(data)
}
