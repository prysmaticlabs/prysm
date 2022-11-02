package depositsnapshot

import (
	"github.com/prysmaticlabs/prysm/v3/math"
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
	Finalize(deposits uint64, depth uint64) (MerkleTreeNode, error)
	// GetFinalized returns a list of hashes of all the finalized nodes and the number of deposits.
	GetFinalized(result [][32]byte) ([][32]byte, uint64)
	// PushLeaf adds a new leaf node at the next available Zero node.
	PushLeaf(leaf [32]byte, deposits uint64, depth uint64) (MerkleTreeNode, error)
}

// fromSnapshotParts creates a new Merkle tree from a list of finalized leaves, number of deposits and specified depth.
// The tree creation is done recursively and not iteratively.
func fromSnapshotParts(finalized [][32]byte, deposits uint64, depth uint64) MerkleTreeNode {
	if len(finalized) < 1 || deposits == 0 {
		return &Zero{
			depth: depth,
		}
	}
	if deposits == math.PowerOf2(depth) {
		return &Finalized{
			deposits: deposits,
			hash:     finalized[0],
		}
	}

	node := Node{}
	if leftSubtree := math.PowerOf2(depth - 1); deposits <= leftSubtree {
		node.left = fromSnapshotParts(finalized, deposits, depth-1)
		node.right = &Zero{depth: depth - 1}

	} else {
		node.left = &Finalized{
			deposits: leftSubtree,
			hash:     finalized[0],
		}
		node.right = fromSnapshotParts(finalized[1:], deposits-leftSubtree, depth-1)
	}
	return &node
}

// fromSnapshotPartsIter creates a new Merkle tree from a list of finalized leaves, number of deposits and specified depth.
// The tree creation is done iteratively and not recursively.
func fromSnapshotPartsIter(finalized [][32]byte, deposits uint64, depth uint64) MerkleTreeNode {
	switch {
	case deposits == 0, len(finalized) == 0:
		return &Zero{depth: depth}
	case depth == 0:
		return &Leaf{
			hash: finalized[0],
		}
	default:
		node := &Node{
			left:  nil,
			right: nil,
		}
		var split uint64
		for depth > 0 {
			split = math.PowerOf2(depth - 1)
			if deposits < split {
				next := &Node{}
				node.left = &Node{left: next, right: &Node{}}
				node.right = &Zero{depth: depth - 1}
				node = next // = node.left
				depth -= 1
			} else if deposits > split {
				node.left = &Finalized{
					deposits: deposits,
					hash:     finalized[0],
				}
				next := &Node{}
				node.right = &Node{left: &Node{}, right: next}
				finalized = finalized[1:]
				deposits -= split
				node = next // = node.right
				depth -= 1
			} else {
				node.left = &Finalized{split, finalized[0]}
				node.right = &Zero{depth: depth - 1}
				finalized = finalized[1:]
			}
		}
		return node
	}
}
