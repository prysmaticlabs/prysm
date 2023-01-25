package depositsnapshot

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
