package memorypool

import (
	"sync"
)

var ArrayPool = new(sync.Pool)

var SliceArrayPool = new(sync.Pool)

func GetByteArray() [32]byte {
	obj := ArrayPool.Get()
	if obj == nil {
		return [32]byte{}
	}
	return obj.([32]byte)
}

func GetSliceByteArray(size int) [][32]byte {
	obj := SliceArrayPool.Get()
	if obj == nil {
		return make([][32]byte, size)
	}
	sliceObj := obj.([][32]byte)

	if size > len(sliceObj) {
		return append(sliceObj, make([][32]byte, size-len(sliceObj))...)
	}
	return sliceObj[:size]
}
