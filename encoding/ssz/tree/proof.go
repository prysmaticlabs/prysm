package tree

import (
	"bytes"
	"errors"
	"math"

	"github.com/minio/sha256-simd"
)

// VerifyProof verifies a single merkle branch. It's more
// efficient than VerifyMultiproof for proving one leaf.
func VerifyProof(root []byte, proof *Proof) (bool, error) {
	if len(proof.Hashes) != getPathLength(proof.Index) {
		return false, errors.New("Invalid proof length")
	}

	node := proof.Leaf[:]
	tmp := make([]byte, 64)
	for i, h := range proof.Hashes {
		if getPosAtLevel(proof.Index, i) {
			copy(tmp[:32], h[:])
			copy(tmp[32:], node[:])
			node = hashFn(tmp)
		} else {
			copy(tmp[:32], node[:])
			copy(tmp[32:], h[:])
			node = hashFn(tmp)
		}
	}

	return bytes.Equal(root, node), nil
}

// Returns the position (i.e. false for left, true for right)
// of an index at a given level.
// Level 0 is the actual index's level, Level 1 is the position
// of the parent, etc.
func getPosAtLevel(index int, level int) bool {
	return (index & (1 << level)) > 0
}

// Returns the length of the path to a node represented by its generalized index.
func getPathLength(index int) int {
	return int(math.Log2(float64(index)))
}

func hashFn(data []byte) []byte {
	res := sha256.Sum256(data)
	return res[:]
}
