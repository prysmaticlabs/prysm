package helperutils

import "github.com/prysmaticlabs/prysm/shared/hashutil"

func MerkleRoot(values [][]byte) []byte {
	length := len(values)
	newSet := make([][]byte, length, length*2)
	newSet = append(newSet, values...)

	for i := length; i >= 0; i-- {
		concatenatedNodes := append(newSet[i*2], newSet[i*2+1]...)
		hash := hashutil.Hash(concatenatedNodes)
		newSet[i] = hash[:]
	}
	return newSet[1]
}
