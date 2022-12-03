package depositsnapshot

import (
	"github.com/prysmaticlabs/prysm/v3/math"
)

const (
	DepositContractDepth = 32 // Maximum tree depth as defined by EIP-4881.
)

// MerkleTreeNode is the interface for a Merkle tree.
type MerkleTreeNode interface {
	// GetRoot returns the root of the Merkle tree.
	GetRoot() [32]byte
	// IsFull returns whether there is space left for deposits.
	IsFull() bool
	// Finalize marks deposits of the Merkle tree as finalized.
	Finalize(deposits uint64, depth uint64) MerkleTreeNode
	// GetFinalized returns a list of hashes of all the finalized nodes and the number of deposits.
	GetFinalized(result [][32]byte) ([][32]byte, uint64)
	// PushLeaf adds a new leaf node at the next available Zero node.
	PushLeaf(leaf [32]byte, depth uint64) MerkleTreeNode
}

func create(leaves [32]byte, depth uint64) MerkleTreeNode {
	if leaves == [32]byte{0} {
		return &ZeroNode{depth: depth}
	}
	if depth == 0 {
		return &LeafNode{hash: [32]byte{leaves[0]}}
	}
	split := math.Min(math.PowerOf2(depth-1), uint64(len(leaves)))
	left := create(leaves[0:split], depth-1)
	right := create(leaves[split:], depth-1)
	return &InnerNode{left: left, right: right}
}

func fromSnapshotParts(finalized [][32]byte, deposits uint64, level uint64) MerkleTreeNode {
	if len(finalized) < 1 || deposits == 0 {
		return &ZeroNode{
			depth: level,
		}
	}
	if deposits == math.PowerOf2(level) {
		return &FinalizedNode{
			depositCount: deposits,
			hash:         finalized[0],
		}
	}
	node := InnerNode{}
	if leftSubtree := math.PowerOf2(level - 1); deposits <= leftSubtree {
		node.left = fromSnapshotParts(finalized, deposits, level-1)
		node.right = &ZeroNode{depth: level - 1}

	} else {
		node.left = &FinalizedNode{
			depositCount: leftSubtree,
			hash:         finalized[0],
		}
		node.right = fromSnapshotParts(finalized[1:], deposits-leftSubtree, level-1)
	}
	return &node
}

func generateProof(tree InnerNode, index uint64, depth uint64) ([32]byte, [32]byte) {
	var proof [32]byte
	node := tree
	for depth > 0 {
		ithBit := (index >> (depth - 1)) & 0x1
		if ithBit == 1 {
			proof = append(proof[:], node.left.GetRoot())
			node = node.right
		} else {
			proof = append(proof[:], node.right.GetRoot())
			node = node.left
		}
		depth -= 1
	}
	//	TODO Add reverse
	return node.GetRoot(), proof
}

// FinalizedNode represents a finalized node and satisfies the MerkleTreeNode interface.
type FinalizedNode struct {
	depositCount uint64
	hash         [32]byte
}

func (f *FinalizedNode) GetRoot() [32]byte {
	return f.hash
}

func (f *FinalizedNode) IsFull() bool {
	return true
}

func (f *FinalizedNode) Finalize(deposits uint64, depth uint64) MerkleTreeNode {
	return f
}

func (f *FinalizedNode) GetFinalized(result [][32]byte) uint64 {
	result = append(result, f.hash)
	return f.depositCount
}

func (f *FinalizedNode) PushLeaf(leaf [32]byte, depth uint64) MerkleTreeNode {
	//TODO implement me
	panic("implement me")
}

// LeafNode represents a leaf node holding a deposit and satisfies the MerkleTreeNode interface.
type LeafNode struct {
	hash [32]byte
}

func (l *LeafNode) GetRoot() [32]byte {
	//TODO implement me
	panic("implement me")
}

func (l *LeafNode) IsFull() bool {
	//TODO implement me
	panic("implement me")
}

func (l *LeafNode) Finalize(deposits uint64, depth uint64) MerkleTreeNode {
	//TODO implement me
	panic("implement me")
}

func (l *LeafNode) GetFinalized(result [][32]byte) ([][32]byte, uint64) {
	//TODO implement me
	panic("implement me")
}

func (l *LeafNode) PushLeaf(leaf [32]byte, depth uint64) MerkleTreeNode {
	//TODO implement me
	panic("implement me")
}

// InnerNode represents an inner node with two children and satisfies the MerkleTreeNode interface.
type InnerNode struct {
	left, right MerkleTreeNode
}

func (i *InnerNode) GetRoot() [32]byte {
	//TODO implement me
	panic("implement me")
}

func (i *InnerNode) IsFull() bool {
	//TODO implement me
	panic("implement me")
}

func (i *InnerNode) Finalize(deposits uint64, depth uint64) MerkleTreeNode {
	//TODO implement me
	panic("implement me")
}

func (i *InnerNode) GetFinalized(result [][32]byte) ([][32]byte, uint64) {
	//TODO implement me
	panic("implement me")
}

func (i *InnerNode) PushLeaf(leaf [32]byte, depth uint64) MerkleTreeNode {
	//TODO implement me
	panic("implement me")
}

// ZeroNode represents an empty node without a deposit and satisfies the MerkleTreeNode interface.
type ZeroNode struct {
	depth uint64
}

func (z *ZeroNode) GetRoot() [32]byte {
	//TODO implement me
	panic("implement me")
}

func (z *ZeroNode) IsFull() bool {
	//TODO implement me
	panic("implement me")
}

func (z *ZeroNode) Finalize(deposits uint64, depth uint64) MerkleTreeNode {
	//TODO implement me
	panic("implement me")
}

func (z *ZeroNode) GetFinalized(result [][32]byte) ([][32]byte, uint64) {
	//TODO implement me
	panic("implement me")
}

func (z *ZeroNode) PushLeaf(leaf [32]byte, depth uint64) MerkleTreeNode {
	//TODO implement me
	panic("implement me")
}
