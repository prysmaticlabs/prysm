package depositsnapshot

import (
	"crypto/sha256"

	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
)

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
func NewDepositTreeFromSnapshotIter(finalized [][32]byte, deposits uint64, finalizedExecutionblock [32]byte) (*DepositTree, error) {
	tree, err := fromSnapshotPartsIter(finalized, deposits, DepositContractDepth)
	if err != nil {
		return nil, err
	}
	return &DepositTree{
		tree:                    tree,
		depositCount:            deposits,
		finalizedExecutionblock: finalizedExecutionblock,
	}, nil
}

// AddDeposit adds a new deposit to the tree.
// Renamed from push_leaf for clarity as it is unrelated to MerkleTreeNode's push_leaf.
func (d *DepositTree) AddDeposit(leaf [32]byte, deposits uint64) error {
	var err error
	d.depositCount += 1
	tree, err := d.tree.PushLeaf(leaf, deposits, DepositContractDepth)
	if err != nil {
		return err
	}
	d.tree = tree
	return nil
}

func (d *DepositTree) GetRoot() [32]byte {
	root := d.tree.GetRoot()
	return sha256.Sum256(append(root[:], bytesutil.Uint64ToBytesLittleEndian32(d.depositCount)...))
}

type DepositTreeSnapshot struct {
	finalized            [][32]byte
	depositRoot          [32]byte
	depositCount         uint64
	executionBlockHash   [32]byte
	executionBlockHeight uint64
}

func (ds *DepositTreeSnapshot) CalculateRoot() [32]byte {
	size := ds.depositCount
	index := len(ds.finalized)
	root := zeroHash
	for i := 0; i <= DepositContractDepth; i++ {
		if size == 1 {
			index -= 1
			root = sha256.Sum256(append(ds.finalized[index][:], root[:]...))
		} else {
			root = sha256.Sum256(append(root[:], zeroHash[:]...))
		}
	}
	return sha256.Sum256(append(root[:], bytesutil.Uint64ToBytesLittleEndian(ds.depositCount)...))
}

type ExecutionBlock struct {
	Hash  [32]byte
	Depth uint64
}

func fromTreeParts(finalised [][32]byte, depositCount uint64, executionBlockHash [32]byte, executionBlockDepth uint64) DepositTreeSnapshot {
	return DepositTreeSnapshot{
		finalized:            finalised,
		depositRoot:          zeroHash,
		depositCount:         depositCount,
		executionBlockHash:   executionBlockHash,
		executionBlockHeight: executionBlockDepth,
	}
}
