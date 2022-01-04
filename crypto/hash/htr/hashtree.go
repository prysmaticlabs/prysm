package htr

/*
#include <hashtree.h>
*/
import "C"
import "unsafe"

func VectorizedSha256(arrayList [][32]byte) [][32]byte {
	sPtr := unsafe.Pointer(&arrayList[0])
	dList := make([][32]byte, len(arrayList)/2)
	C.sha256_8_avx2((*C.uchar)(unsafe.Pointer(&dList[0])), (*C.uchar)(sPtr), C.ulong(len(arrayList)/2))
	return dList
}
