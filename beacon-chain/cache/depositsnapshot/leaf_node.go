package depositsnapshot

import "github.com/pkg/errors"

// LeafNode represents a leaf node holding a deposit and satisfies the MerkleTreeNode interface.
type LeafNode struct {
	hash [32]byte
}

// GetRoot returns the root of the Merkle tree.
func (l *LeafNode) GetRoot() [32]byte {
	return l.hash
}

// IsFull returns whether there is space left for deposits.
func (l *LeafNode) IsFull() bool {
	return true
}

// Finalize marks deposits of the Merkle tree as finalized.
func (l *LeafNode) Finalize(deposits uint64, depth uint64) (MerkleTreeNode, error) {
	return &FinalizedNode{
		deposits: 1,
		hash:     l.hash,
	}, nil
}

// GetFinalized returns a list of hashes of all the finalized nodes and the number of deposits.
func (l *LeafNode) GetFinalized(result [][32]byte) ([][32]byte, uint64) {
	return result, 0
}

// PushLeaf adds a new leaf node at the next available zero node.
func (l *LeafNode) PushLeaf(leaf [32]byte, deposits uint64, depth uint64) (MerkleTreeNode, error) {
	return nil, errors.New("leaf should not be able to push another leaf")
}
