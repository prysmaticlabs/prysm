package helperutils

import "github.com/prysmaticlabs/prysm/shared/hashutil"

// MerkleRoot derives the merkle root from a 2d byte array with each element
// in the outer array signifying the data that is to be represented in the
// merkle tree.
func MerkleRoot(values [][]byte) []byte {
	length := len(values)
	newSet := make([][]byte, length, length*2)
	newSet = append(newSet, values...)

	for i := length - 1; i >= 0; i-- {
		concatenatedNodes := append(newSet[i*2], newSet[i*2+1]...)
		hash := hashutil.Hash(concatenatedNodes)
		newSet[i] = hash[:]
	}
	return newSet[1]
}
