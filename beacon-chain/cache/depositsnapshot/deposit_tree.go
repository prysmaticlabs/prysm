package depositsnapshot

// DepositTree is the Merkle tree representation of deposits.
type DepositTree struct {
	tree                    MerkleTreeNode
	depositCount            uint64 // number of deposits in the tree, reference implementation calls this mix_in_length.
	finalizedExecutionblock [32]byte
}

// NewDepositTree creates an empty deposit tree.
func NewDepositTree() *DepositTree {
	return &DepositTree{
		tree:                    &ZeroNode{depth: DepositContractDepth},
		depositCount:            0,
		finalizedExecutionblock: [32]byte{},
	}
}

// NewDepositTreeFromSnapshot creates a deposit tree from an existing deposit tree snapshot recursively.
func NewDepositTreeFromSnapshot(finalized [][32]byte, deposits uint64, finalizedExecutionblock [32]byte) *DepositTree {
	return &DepositTree{
		tree:                    fromSnapshotParts(finalized, deposits, DepositContractDepth),
		depositCount:            deposits,
		finalizedExecutionblock: finalizedExecutionblock,
	}
}

// NewDepositTreeFromSnapshotIter creates a deposit tree from an existing deposit tree snapshot iteratively.
func NewDepositTreeFromSnapshotIter(finalized [][32]byte, deposits uint64, finalizedExecutionblock [32]byte) *DepositTree {
	return &DepositTree{
		tree:                    fromSnapshotPartsIter(finalized, deposits, DepositContractDepth),
		depositCount:            deposits,
		finalizedExecutionblock: finalizedExecutionblock,
	}
}

// AddDeposit adds a new deposit to the tree.
func (d *DepositTree) AddDeposit(leaf [32]byte, deposits uint64) error {
	var err error
	d.depositCount += 1
	d.tree, err = d.tree.PushLeaf(leaf, deposits, DepositContractDepth)
	if err != nil {
		return err
	}
	return nil
}
