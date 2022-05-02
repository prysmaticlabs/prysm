package htr

import (
	"github.com/prysmaticlabs/gohashtree"
)

// VectorizedSha256 takes a list of roots and hashes them using CPU
// specific vector instructions. Depending on host machine's specific
// hardware configuration, using this routine can lead to a significant
// performance improvement compared to the default method of hashing
// lists.
func VectorizedSha256(inputList [][32]byte, outputList [][32]byte) {
	err := gohashtree.Hash(outputList, inputList)
	if err != nil {
		panic(err)
	}
}
