package trie

import (
	"github.com/prysmaticlabs/prysm/shared/hashutil"
)

// VerifyMerkleBranch verifies a merkle path in a trie
// by checking the aggregated hash of contiguous leaves along a path
// eventually equals the root hash of the merkle trie.
func VerifyMerkleBranch(leaf [32]byte, branch [][]byte, depth uint64, root [32]byte) bool {
	value := leaf
	for i := uint64(0); i < depth; i++ {
		if i%2 == 0 {
			value = hashutil.Hash(append(branch[i], value[:]...))
		} else {
			value = hashutil.Hash(append(value[:], branch[i]...))
		}
	}
	return value == root
}
