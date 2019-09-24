package trieutil

import (
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// NextPowerOf2 returns the next power of 2 >= the input
//
// Spec pseudocode definition:
//   def get_next_power_of_two(x: int) -> int:
//    """
//    Get next power of 2 >= the input.
//    """
//    if x <= 2:
//        return x
//    else:
//        return 2 * get_next_power_of_two((x + 1) // 2)
func NextPowerOf2(n int) int {
	if n <= 2 {
		return n
	}
	return 2 * NextPowerOf2((n+1)/2)
}

// PrevPowerOf2 returns the previous power of 2 >= the input
//
// Spec pseudocode definition:
//   def get_previous_power_of_two(x: int) -> int:
//    """
//    Get the previous power of 2 >= the input.
//    """
//    if x <= 2:
//        return x
//    else:
//        return 2 * get_previous_power_of_two(x // 2)
func PrevPowerOf2(n int) int {
	if n <= 2 {
		return n
	}
	return 2 * PrevPowerOf2(n/2)
}

// MerkleTree returns all the nodes in a merkle tree from inputting merkle leaves.
//
// Spec pseudocode definition:
//   def merkle_tree(leaves: Sequence[Hash]) -> Sequence[Hash]:
//    padded_length = get_next_power_of_two(len(leaves))
//    o = [Hash()] * padded_length + list(leaves) + [Hash()] * (padded_length - len(leaves))
//    for i in range(padded_length - 1, 0, -1):
//        o[i] = hash(o[i * 2] + o[i * 2 + 1])
//    return o
func MerkleTree(leaves [][]byte) [][]byte {
	paddedLength := NextPowerOf2(len(leaves))
	parents := make([][]byte, paddedLength)
	paddedLeaves := make([][]byte, paddedLength-len(leaves))

	for i := 0; i < len(parents); i++ {
		parents[i] = params.BeaconConfig().ZeroHash[:]
	}
	for i := 0; i < len(paddedLeaves); i++ {
		paddedLeaves[i] = params.BeaconConfig().ZeroHash[:]
	}

	merkleTree := make([][]byte, 0, len(parents)+len(leaves)+len(paddedLeaves))
	merkleTree = append(merkleTree, parents...)
	merkleTree = append(merkleTree, leaves...)
	merkleTree = append(merkleTree, paddedLeaves...)

	for i := len(paddedLeaves) - 1; i > 0; i-- {
		a := append(merkleTree[2*i], merkleTree[2*i+1]...)
		b := hashutil.Hash(a)
		merkleTree[i] = b[:]
	}

	return merkleTree
}
