package depositsnapshot

import (
	"github.com/prysmaticlabs/prysm/v3/crypto/hash"
	"github.com/prysmaticlabs/prysm/v3/math"
)

// InnerNode represents an inner node with two children and satisfies the MerkleTreeNode interface.
type InnerNode struct {
	left, right MerkleTreeNode
}

// GetRoot returns the root of the Merkle tree.
func (n *InnerNode) GetRoot() [32]byte {
	left := n.left.GetRoot()
	right := n.right.GetRoot()
	return hash.Hash(append(left[:], right[:]...))
}

// IsFull returns whether there is space left for deposits.
func (n *InnerNode) IsFull() bool {
	return n.right.IsFull()
}

// Finalize marks deposits of the Merkle tree as finalized.
func (n *InnerNode) Finalize(deposits uint64, depth uint64) (MerkleTreeNode, error) {
	depositsNum := math.PowerOf2(depth)
	if depositsNum <= deposits {
		return &FinalizedNode{
			deposits: depositsNum,
			hash:     n.GetRoot(),
		}, nil
	}
	n.left, _ = n.left.Finalize(deposits, depth-1)
	if deposits > depositsNum/2 {
		remaining := deposits - depositsNum/2
		n.right, _ = n.right.Finalize(remaining, depth-1)
	}
	return n, nil
}

// GetFinalized returns a list of hashes of all the finalized nodes and the number of deposits.
func (n *InnerNode) GetFinalized(result [][32]byte) ([][32]byte, uint64) {
	result, depositsLeft := n.left.GetFinalized(result)
	result, depositsRight := n.right.GetFinalized(result)

	return result, depositsLeft + depositsRight
}

// PushLeaf adds a new leaf node at the next available zero node.
func (n *InnerNode) PushLeaf(leaf [32]byte, deposits uint64, depth uint64) (MerkleTreeNode, error) {
	if !n.left.IsFull() {
		left, err := n.left.PushLeaf(leaf, deposits, depth-1)
		if err == nil {
			n.left = left
		} else {
			return err
		}
	} else {
		right, err := n.right.PushLeaf(leaf, deposits, depth-1)
		if err == nil {
			n.right = right
		} else {
			return err
		}
	}
	return n, nil
}
