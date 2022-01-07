package htr

/*
#include <hashtree.h>
*/
import "C"
import (
	"fmt"
	"unsafe"

	"github.com/prysmaticlabs/prysm/container/trie"
	mathutil "github.com/prysmaticlabs/prysm/math"
)

func VectorizedSha256(arrayList [][32]byte) [][32]byte {
	sPtr := unsafe.Pointer(&arrayList[0])
	dList := make([][32]byte, len(arrayList)/2)
	C.sha256_8_avx2((*C.uchar)(unsafe.Pointer(&dList[0])), (*C.uchar)(sPtr), C.ulong(len(arrayList)/2))
	return dList
}

func SinglePassRoot(arrayList [][32]byte, depth int) [32]byte {
	numOfElems := len(arrayList)
	elemSum := 0
	layerLen := numOfElems
	for i := 0; i < depth; i++ {
		oddNodeLength := layerLen%2 == 1
		if oddNodeLength {
			layerLen++
		}
		elemSum += layerLen
		layerLen /= 2
		layerLen = int(mathutil.Max(1, uint64(layerLen)))
		if i+1 == depth {
			elemSum++
		}
	}

	resList := make([][32]byte, elemSum-numOfElems)
	resList = append(arrayList, resList...)
	if len(resList) == numOfElems {
		panic(fmt.Sprintf("incorrect length: %d, %d,%d, %d", len(resList), numOfElems, elemSum, depth))
	}
	layerLen = numOfElems
	currIdx := -1
	for i := 0; i < depth; i++ {
		currIdx += layerLen
		oddNodeLength := layerLen%2 == 1
		if oddNodeLength {
			zerohash := trie.ZeroHashes[i]
			layerLen++
			currIdx++
			resList[currIdx] = zerohash
		}
		layerLen /= 2
		layerLen = int(mathutil.Max(1, uint64(layerLen)))
		if i+1 == depth {
			elemSum++
		}
	}
	sPtr := unsafe.Pointer(&resList[0])
	C.sha256_8_avx2((*C.uchar)(unsafe.Pointer(&resList[numOfElems])), (*C.uchar)(sPtr), C.ulong(len(resList)/2))
	return resList[len(resList)-1]
}

func Hash2Chunks(arrayList [][32]byte) [32]byte {
	sPtr := unsafe.Pointer(&arrayList[0])
	dList := [32]byte{}
	C.sha256_1_avx((*C.uchar)(unsafe.Pointer(&dList)), (*C.uchar)(sPtr), C.ulong(len(arrayList)/2))
	return dList
}
