package trieutil

import (
	"math"

	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// NextPowerOf2 returns the next power of 2 >= the input
//
// Spec pseudocode definition:
//   def get_power_of_two_ceil(x: int) -> int:
//    """
//    Get the power of 2 for given input, or the closest higher power of 2 if the input is not a power of 2.
//    Commonly used for "how many nodes do I need for a bottom tree layer fitting x elements?"
//    Example: 0->1, 1->1, 2->2, 3->4, 4->4, 5->8, 6->8, 7->8, 8->8, 9->16.
//    """
//    if x <= 1:
//        return 1
//    elif x == 2:
//        return 2
//    else:
//        return 2 * get_power_of_two_ceil((x + 1) // 2)
func NextPowerOf2(n int) int {
	if n <= 1 {
		return 1
	}
	if n == 2 {
		return n
	}
	return 2 * NextPowerOf2((n+1)/2)
}

// PrevPowerOf2 returns the previous power of 2 >= the input
//
// Spec pseudocode definition:
//   def get_power_of_two_floor(x: int) -> int:
//    """
//    Get the power of 2 for given input, or the closest lower power of 2 if the input is not a power of 2.
//    The zero case is a placeholder and not used for math with generalized indices.
//    Commonly used for "what power of two makes up the root bit of the generalized index?"
//    Example: 0->1, 1->1, 2->2, 3->2, 4->4, 5->4, 6->4, 7->4, 8->8, 9->8
//    """
//    if x <= 1:
//        return 1
//    if x == 2:
//        return x
//    else:
//        return 2 * get_power_of_two_floor(x // 2)
func PrevPowerOf2(n int) int {
	if n <= 1 {
		return 1
	}
	if n == 2 {
		return n
	}
	return 2 * PrevPowerOf2(n/2)
}

// MerkleTree returns all the nodes in a merkle tree from inputting merkle leaves.
//
// Spec pseudocode definition:
//   def merkle_tree(leaves: Sequence[Bytes32]) -> Sequence[Bytes32]:
//    """
//    Return an array representing the tree nodes by generalized index:
//    [0, 1, 2, 3, 4, 5, 6, 7], where each layer is a power of 2. The 0 index is ignored. The 1 index is the root.
//    The result will be twice the size as the padded bottom layer for the input leaves.
//    """
//    bottom_length = get_power_of_two_ceil(len(leaves))
//    o = [Bytes32()] * bottom_length + list(leaves) + [Bytes32()] * (bottom_length - len(leaves))
//    for i in range(bottom_length - 1, 0, -1):
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

	merkleTree := make([][]byte, len(parents)+len(leaves)+len(paddedLeaves))
	copy(merkleTree, parents)
	l := len(parents)
	copy(merkleTree[l:], leaves)
	l += len(paddedLeaves)
	copy(merkleTree[l:], paddedLeaves)

	for i := len(paddedLeaves) - 1; i > 0; i-- {
		a := append(merkleTree[2*i], merkleTree[2*i+1]...)
		b := hashutil.Hash(a)
		merkleTree[i] = b[:]
	}

	return merkleTree
}

// ConcatGeneralizedIndices concats the generalized indices together.
//
// Spec pseudocode definition:
//   def concat_generalized_indices(*indices: GeneralizedIndex) -> GeneralizedIndex:
//    """
//    Given generalized indices i1 for A -> B, i2 for B -> C .... i_n for Y -> Z, returns
//    the generalized index for A -> Z.
//    """
//    o = GeneralizedIndex(1)
//    for i in indices:
//        o = GeneralizedIndex(o * get_power_of_two_floor(i) + (i - get_power_of_two_floor(i)))
//    return o
func ConcatGeneralizedIndices(indices []int) int {
	index := 1
	for _, i := range indices {
		index = index*PrevPowerOf2(i) + (i - PrevPowerOf2(i))
	}
	return index
}

// GeneralizedIndexLength returns the generalized index length from a given index.
//
// Spec pseudocode definition:
//   def get_generalized_index_length(index: GeneralizedIndex) -> int:
//    """
//    Return the length of a path represented by a generalized index.
//    """
//    return int(log2(index))
func GeneralizedIndexLength(index int) int {
	return int(math.Log2(float64(index)))
}

// GeneralizedIndexBit returns the given bit of a generalized index.
//
// Spec pseudocode definition:
//   def get_generalized_index_bit(index: GeneralizedIndex, position: int) -> bool:
//    """
//    Return the given bit of a generalized index.
//    """
//    return (index & (1 << position)) > 0
func GeneralizedIndexBit(index, pos uint64) bool {
	return (index & (1 << pos)) > 0
}

// GeneralizedIndexSibling returns the sibling of a generalized index.
//
// Spec pseudocode definition:
//   def generalized_index_sibling(index: GeneralizedIndex) -> GeneralizedIndex:
//    return GeneralizedIndex(index ^ 1)
func GeneralizedIndexSibling(index int) int {
	return index ^ 1
}

// GeneralizedIndexChild returns the child of a generalized index.
//
// Spec pseudocode definition:
//   def generalized_index_child(index: GeneralizedIndex, right_side: bool) -> GeneralizedIndex:
//    return GeneralizedIndex(index * 2 + right_side)
func GeneralizedIndexChild(index int, rightSide bool) int {
	if rightSide {
		return index*2 + 1
	}
	return index * 2
}

// GeneralizedIndexParent returns the parent of a generalized index.
//
// Spec pseudocode definition:
//   def generalized_index_parent(index: GeneralizedIndex) -> GeneralizedIndex:
//    return GeneralizedIndex(index // 2)
func GeneralizedIndexParent(index int) int {
	return index / 2
}
