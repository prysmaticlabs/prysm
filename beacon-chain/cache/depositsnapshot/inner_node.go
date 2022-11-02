package depositsnapshot

import (
	"github.com/prysmaticlabs/prysm/v3/crypto/hash"
	"github.com/prysmaticlabs/prysm/v3/math"
)

// Node represents an inner node with two children and satisfies the MerkleTreeNode interface.
type Node struct {
	left, right MerkleTreeNode
}

// GetRoot returns the root of the Merkle tree.
func (n *Node) GetRoot() [32]byte {
	left := n.left.GetRoot()
	right := n.right.GetRoot()
	return hash.Hash(append(left[:], right[:]...))
}

// IsFull returns whether there is space left for deposits.
func (n *Node) IsFull() bool {
	return n.right.IsFull()
}

// Finalize marks deposits of the Merkle tree as finalized.
func (n *Node) Finalize(deposits uint64, depth uint64) (MerkleTreeNode, error) {
	depositsNum := math.PowerOf2(depth)
	if depositsNum <= deposits {
		return &Finalized{
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
func (n *Node) GetFinalized(result [][32]byte) ([][32]byte, uint64) {
	result, depositsLeft := n.left.GetFinalized(result)
	result, depositsRight := n.right.GetFinalized(result)

	return result, depositsLeft + depositsRight
}

// PushLeaf adds a new leaf node at the next available Zero node.
func (n *Node) PushLeaf(leaf [32]byte, deposits uint64, depth uint64) (MerkleTreeNode, error) {
	if !n.left.IsFull() {
		n.left, _ = n.left.PushLeaf(leaf, deposits, depth-1)
	} else {
		n.right, _ = n.right.PushLeaf(leaf, deposits, depth-1)
	}
	return n, nil
}
