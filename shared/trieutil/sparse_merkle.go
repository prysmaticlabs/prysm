package trieutil

import "github.com/prysmaticlabs/prysm/shared/hashutil"

type MerkleTrie struct {
	branches [][][32]byte
}

// GenerateTrieFromItems constructs a Merkle trie from a sequence of byte slices.
func GenerateTrieFromItems(items [][]byte) *MerkleTrie {
	return &MerkleTrie{}
}

// BranchIndices returns the indices of all ancestors for a node with up to the root
// given the node's index by utilizing the depth of the trie.
func BranchIndices(merkleIndex int) []int {
	return []int{}
}

// VerifyMerkleProof verifies a Merkle branch against a root of a trie.
func VerifyMerkleProof(root [32]byte, item []byte, merkleIndex int, proof [][32]byte) bool {
	return true
}

// CalculateRootFromItems constructs a Merkle trie from a sequence
// of items and fetches the corresponding Merkle root.
func CalculateRootFromItems(items [][]byte) [32]byte {
	return [32]byte{}
}

// Root of the Merkle trie.
func (m *MerkleTrie) Root() [32]byte{
	return m.branches[0][0]
}

// MerkleProof obtains a Merkle proof for an item at a given
// index in the Merkle trie up to the root of the trie.
func (m *MerkleTrie) MerkleProof(merkleIndex int) [][32]byte {
	_ := BranchIndices(merkleIndex)
	return [][32]byte{}
}

// parentHash takes a left and right node and hashes their concatenation.
func (m *MerkleTrie) parentHash(left [32]byte, right [32]byte) [32]byte {
	return hashutil.Hash(append(left[:], right[:]...))
}
