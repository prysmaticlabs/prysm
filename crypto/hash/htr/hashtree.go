package htr

import (
	"github.com/prysmaticlabs/gohashtree"
)

// VectorizedSha256 takes a list of roots and hashes them using CPU
// specific vector instructions. Depending on host machine's specific
// hardware configuration, using this routine can lead to a significant
// performance improvement compared to the default method of hashing
// lists.
func VectorizedSha256(arrayList [][32]byte) [][32]byte {
	dList := make([][32]byte, len(arrayList)/2)
	err := gohashtree.Hash(dList, arrayList)
	if err != nil {
		panic(err)
	}
	return dList
}
