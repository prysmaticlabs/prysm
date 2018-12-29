package trie

import (
	"testing"

	"github.com/prysmaticlabs/prysm/shared/hashutil"
)

func TestVerifyMerkleBranch(t *testing.T) {
	// We build up a Merkle proof and then run
	// the verify function for data integrity testing
	// along a Merkle branch in a Merkle trie structure.
	depth := 4
	leaf := [32]byte{1, 2, 3}
	root := leaf
	branch := [][]byte{
		{4, 5, 6},
		{7, 8, 9},
		{10, 11, 12},
		{13, 14, 15},
	}
	for i := 0; i < depth; i++ {
		if i%2 == 0 {
			root = hashutil.Hash(append(branch[i], root[:]...))
		} else {
			root = hashutil.Hash(append(root[:], branch[i]...))
		}
	}
	if ok := VerifyMerkleBranch(leaf, branch, uint64(depth), root); !ok {
		t.Error("Expected merkle branch to verify, received false")
	}
}
