package depositsnapshot

import (
	"github.com/prysmaticlabs/prysm/v3/crypto/hash"
)

const (
	DepositContractDepth = 32 // Maximum tree depth as defined by EIP-4881
)

var zeroHash = [32]byte{}

// MerkleTreeNode is the interface for a Merkle tree.
type MerkleTreeNode interface {
	// GetRoot returns the root of the Merkle tree.
	GetRoot() [32]byte
	// IsFull returns whether there is space left for deposits.
	IsFull() bool
	// Finalize marks deposits of the Merkle tree as finalized.
	Finalize(deposits uint, depth uint) MerkleTreeNode
	// GetFinalized returns a list of hashes of all the finalized nodes and the number of deposits.
	GetFinalized(result [][32]byte) ([][32]byte, uint)
	// PushLeaf adds a new leaf node at the next available Zero node.
	PushLeaf(leaf [32]byte, deposits uint, depth uint) MerkleTreeNode
}

// Finalized represents a finalized node and satisfies the MerkleTreeNode interface.
type Finalized struct {
	deposits uint
	hash     [32]byte
}

// GetRoot satisfies the MerkleTreeNode interface.
func (f *Finalized) GetRoot() [32]byte {
	return f.hash
}

// IsFull satisfies the MerkleTreeNode interface.
func (f *Finalized) IsFull() bool {
	return true
}

// Finalize satisfies the MerkleTreeNode interface.
func (f *Finalized) Finalize(deposits uint, depth uint) MerkleTreeNode {
	return f
}

// GetFinalized satisfies the MerkleTreeNode interface.
func (f *Finalized) GetFinalized(result [][32]byte) ([][32]byte, uint) {
	return append(result, f.hash), f.deposits
}

// PushLeaf satisfies the MerkleTreeNode interface.
func (f *Finalized) PushLeaf(leaf [32]byte, deposits uint, depth uint) MerkleTreeNode {
	panic("Can't push a leaf to something finalized")
}

// Leaf represents a finalized leaf node holding a deposit and satisfies the MerkleTreeNode interface.
type Leaf struct {
	hash [32]byte
}

// GetRoot satisfies the MerkleTreeNode interface.
func (l *Leaf) GetRoot() [32]byte {
	return l.hash
}

// IsFull satisfies the MerkleTreeNode interface.
func (l *Leaf) IsFull() bool {
	return true
}

// Finalize satisfies the MerkleTreeNode interface.
func (l *Leaf) Finalize(deposits uint, depth uint) MerkleTreeNode {
	return &Finalized{
		deposits: 1,
		hash:     l.hash,
	}
}

// GetFinalized satisfies the MerkleTreeNode interface.
func (l *Leaf) GetFinalized(result [][32]byte) ([][32]byte, uint) {
	return result, 0
}

// PushLeaf satisfies the MerkleTreeNode interface.
func (l *Leaf) PushLeaf(leaf [32]byte, deposits uint, depth uint) MerkleTreeNode {
	panic("leaf should not be able to push another leaf")
}

// Node represents an inner node with two children and satisfies the MerkleTreeNode interface.
type Node struct {
	left, right MerkleTreeNode
}

// GetRoot satisfies the MerkleTreeNode interface.
func (n *Node) GetRoot() [32]byte {
	left := n.left.GetRoot()
	right := n.right.GetRoot()
	return hash.Hash(append(left[:], right[:]...))
}

// IsFull satisfies the MerkleTreeNode interface.
func (n *Node) IsFull() bool {
	return n.right.IsFull()
}

// Finalize satisfies the MerkleTreeNode interface.
func (n *Node) Finalize(deposits uint, depth uint) MerkleTreeNode {
	depositsNum := UintPow(2, depth)
	if depositsNum <= deposits {
		return &Finalized{
			deposits: depositsNum,
			hash:     n.GetRoot(),
		}
	}
	n.left = n.left.Finalize(deposits, depth-1)
	if deposits > depositsNum/2 {
		remaining := deposits - depositsNum/2
		n.right = n.right.Finalize(remaining, depth-1)
	}
	return n
}

// GetFinalized satisfies the MerkleTreeNode interface.
func (n *Node) GetFinalized(result [][32]byte) ([][32]byte, uint) {
	result, depositsLeft := n.left.GetFinalized(result)
	result, depositsRight := n.right.GetFinalized(result)

	return result, depositsLeft + depositsRight
}

// PushLeaf satisfies the MerkleTreeNode interface.
func (n *Node) PushLeaf(leaf [32]byte, deposits uint, depth uint) MerkleTreeNode {
	if !n.left.IsFull() {
		n.left = n.left.PushLeaf(leaf, deposits, depth-1)
	} else {
		n.right = n.right.PushLeaf(leaf, deposits, depth-1)
	}
	return n
}

// Zero represents an empty leaf without a deposit and satisfies the MerkleTreeNode interface.
type Zero struct {
	depth uint
}

// GetRoot satisfies the MerkleTreeNode interface.
func (z *Zero) GetRoot() [32]byte {
	if z.depth == DepositContractDepth {
		return hash.Hash(append(zeroHash[:], zeroHash[:]...))
	}
	return zeroHash
}

// IsFull satisfies the MerkleTreeNode interface.
func (z *Zero) IsFull() bool {
	return false
}

// Finalize satisfies the MerkleTreeNode interface.
func (z *Zero) Finalize(deposits uint, depth uint) MerkleTreeNode {
	panic("finalize should not be called")
}

// GetFinalized satisfies the MerkleTreeNode interface.
func (z *Zero) GetFinalized(result [][32]byte) ([][32]byte, uint) {
	return result, 0
}

// UintPow is a utility function to compute the power of 2 for uint.
func UintPow(n, m uint) uint {
	if m == 0 {
		return 1
	}
	result := n
	for i := uint(2); i <= m; i++ {
		result *= n
	}
	return result
}
