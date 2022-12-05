package depositsnapshot

import (
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/crypto/hash"
	"github.com/prysmaticlabs/prysm/v3/math"
)

const (
	DepositContractDepth = 32 // Maximum tree depth as defined by EIP-4881.
)

var (
	// ErrFinalizedNodeCannotPushLeaf may occur when attempting to push a leaf to a finalized node. When a node is finalized, it cannot be modified or changed.
	ErrFinalizedNodeCannotPushLeaf = errors.New("can't push a leaf to a finalized node")
	ErrLeafNodeCannotPushLeaf      = errors.New("can't push a leaf to a leaf node")
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
	GetFinalized(result [][32]byte) uint64
	// PushLeaf adds a new leaf node at the next available Zero node.
	PushLeaf(leaf [32]byte, depth uint64) (MerkleTreeNode, error)
}

func create(leaves [][32]byte, depth uint64) MerkleTreeNode {
	length := uint64(len(leaves))
	if length == 0 {
		return &ZeroNode{depth: depth}
	}
	if depth == 0 {
		return &LeafNode{hash: leaves[0]}
	}
	split := math.Min(math.PowerOf2(depth-1), length)
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

func generateProof(tree InnerNode, index uint64, depth uint64) ([32]byte, [][32]byte) {
	var proof [][32]byte
	node := tree
	for depth > 0 {
		ithBit := (index >> (depth - 1)) & 0x1
		if ithBit == 1 {
			proof = append(proof, node.left.GetRoot())
			//node = node.right
		} else {
			proof = append(proof, node.right.GetRoot())
			//node = node.left
		}
		depth -= 1
	}
	proof = Reverse(proof)
	return node.GetRoot(), proof
}

func Reverse[E any](s []E) []E {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
	return s
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

func (f *FinalizedNode) PushLeaf(leaf [32]byte, depth uint64) (MerkleTreeNode, error) {
	return nil, ErrFinalizedNodeCannotPushLeaf
}

// LeafNode represents a leaf node holding a deposit and satisfies the MerkleTreeNode interface.
type LeafNode struct {
	hash [32]byte
}

func (l *LeafNode) GetRoot() [32]byte {
	return l.hash
}

func (l *LeafNode) IsFull() bool {
	return true
}

func (l *LeafNode) Finalize(deposits uint64, depth uint64) MerkleTreeNode {
	return &FinalizedNode{1, l.hash}
}

func (l *LeafNode) GetFinalized(result [][32]byte) uint64 {
	return 0
}

func (l *LeafNode) PushLeaf(leaf [32]byte, depth uint64) (MerkleTreeNode, error) {
	return nil, ErrLeafNodeCannotPushLeaf
}

// InnerNode represents an inner node with two children and satisfies the MerkleTreeNode interface.
type InnerNode struct {
	left, right MerkleTreeNode
}

func (n *InnerNode) GetRoot() [32]byte {
	left := n.left.GetRoot()
	right := n.right.GetRoot()
	return hash.Hash(append(left[:], right[:]...))
}

func (n *InnerNode) IsFull() bool {
	return n.right.IsFull()
}

func (n *InnerNode) Finalize(depositsToFinalize uint64, depth uint64) MerkleTreeNode {
	deposits := math.PowerOf2(depth)
	if deposits <= depositsToFinalize {
		return &FinalizedNode{deposits, n.GetRoot()}
	}
	n.left = n.left.Finalize(depositsToFinalize, depth-1)
	if depositsToFinalize > deposits/2 {
		remaining := depositsToFinalize - deposits/2
		n.right = n.right.Finalize(remaining, depth-1)
	}
	return n
}

func (n *InnerNode) GetFinalized(result [][32]byte) uint64 {
	return n.left.GetFinalized(result) + n.right.GetFinalized(result)
}

func (n *InnerNode) PushLeaf(leaf [32]byte, depth uint64) (MerkleTreeNode, error) {
	if !n.left.IsFull() {
		left, err := n.left.PushLeaf(leaf, depth-1)
		if err == nil {
			n.left = left
		} else {
			return n, err
		}
	} else {
		right, err := n.right.PushLeaf(leaf, depth-1)
		if err == nil {
			n.right = right
		} else {
			return n, err
		}
	}
	return n, nil
}

// ZeroNode represents an empty node without a deposit and satisfies the MerkleTreeNode interface.
type ZeroNode struct {
	depth uint64
}

func (z *ZeroNode) GetRoot() [32]byte {
	if z.depth == DepositContractDepth {
		return hash.Hash(append(Zerohashes[z.depth-1][:], Zerohashes[z.depth-1][:]...))
	}
	return Zerohashes[z.depth]
}

func (z *ZeroNode) IsFull() bool {
	return false
}

func (z *ZeroNode) Finalize(deposits uint64, depth uint64) MerkleTreeNode {
	return nil
}

func (z *ZeroNode) GetFinalized(result [][32]byte) uint64 {
	return 0
}

func (z *ZeroNode) PushLeaf(leaf [32]byte, depth uint64) (MerkleTreeNode, error) {
	return create([][32]byte{leaf}, depth), nil
}
