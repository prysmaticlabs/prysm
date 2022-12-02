package depositsnapshot

import (
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/crypto/hash"
)

var (
	// ErrZeroNodeCannotBeFinalized may occur when attempting to finalize a zero node. A zero node is empty and can't be finalized.
	ErrZeroNodeCannotBeFinalized = errors.New("can't finalize a zero node")
)

// ZeroNode represents an empty node without a deposit and satisfies the MerkleTreeNode interface.
type ZeroNode struct {
	depth uint64
}

// GetRoot returns the root of the Merkle tree.
func (z *ZeroNode) GetRoot() [32]byte {
	if z.depth == DepositContractDepth {
		return hash.Hash(append(zeroHash[:], zeroHash[:]...))
	}
	return zeroHash
}

// IsFull returns whether there is space left for deposits.
// A ZeroNode will always return false as a ZeroNode is an empty node
// that gets replaced by a deposit.
func (z *ZeroNode) IsFull() bool {
	return false
}

// Finalize marks deposits of the Merkle tree as finalized.
func (z *ZeroNode) Finalize(deposits uint64, depth uint64) (MerkleTreeNode, error) {
	return nil, ErrZeroNodeCannotBeFinalized
}

// GetFinalized returns a list of hashes of all the finalized nodes and the number of deposits.
func (z *ZeroNode) GetFinalized(result [][32]byte) ([][32]byte, uint64) {
	return result, 0
}

// PushLeaf adds a new leaf node at the next available zero node.
func (z *ZeroNode) PushLeaf(leaf [32]byte, deposits uint64, depth uint64) (MerkleTreeNode, error) {
	return fromSnapshotPartsIter([][32]byte{leaf}, deposits, depth)
}
