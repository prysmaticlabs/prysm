package depositsnapshot

import "github.com/pkg/errors"

// Finalized represents a finalized node and satisfies the MerkleTreeNode interface.
type Finalized struct {
	deposits uint64
	hash     [32]byte
}

// GetRoot returns the root of the Merkle tree.
func (f *Finalized) GetRoot() [32]byte {
	return f.hash
}

// IsFull returns whether there is space left for deposits.
func (f *Finalized) IsFull() bool {
	return true
}

// Finalize marks deposits of the Merkle tree as finalized.
func (f *Finalized) Finalize(deposits uint64, depth uint64) (MerkleTreeNode, error) {
	return f, nil
}

// GetFinalized returns a list of hashes of all the finalized nodes and the number of deposits.
func (f *Finalized) GetFinalized(result [][32]byte) ([][32]byte, uint64) {
	return append(result, f.hash), f.deposits
}

// PushLeaf adds a new leaf node at the next available Zero node.
func (f *Finalized) PushLeaf(leaf [32]byte, deposits uint64, depth uint64) (MerkleTreeNode, error) {
	return nil, errors.New("can't push a leaf to a finalized node")
}
