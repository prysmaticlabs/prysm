// Package depositsnapshot implements the EIP-4881 standard for minimal sparse Merkle tree.
// The format proposed by the EIP allows for the pruning of deposits that are no longer needed to participate fully in consensus.
// Full EIP-4881 specification can be found here: https://eips.ethereum.org/EIPS/eip-4881
package depositsnapshot

import (
	"encoding/binary"
	"fmt"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/crypto/hash"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v4/math"
	eth "github.com/prysmaticlabs/prysm/v4/proto/eth/v1"
)

var (
	// ErrEmptyExecutionBlock occurs when the execution block is nil.
	ErrEmptyExecutionBlock = errors.New("empty execution block")
	// ErrInvalidSnapshotRoot occurs when the snapshot root does not match the calculated root.
	ErrInvalidSnapshotRoot = errors.New("snapshot root is invalid")
	// ErrInvalidMixInLength occurs when the value for mix in length is 0.
	ErrInvalidMixInLength = errors.New("depositCount should be greater than 0")
	// ErrInvalidIndex occurs when the index is less than the number of finalized deposits.
	ErrInvalidIndex = errors.New("index should be greater than finalizedDeposits - 1")
	// ErrNoDeposits occurs when the number of deposits is 0.
	ErrNoDeposits = errors.New("number of deposits should be greater than 0")
	// ErrTooManyDeposits occurs when the number of deposits exceeds the capacity of the tree.
	ErrTooManyDeposits = errors.New("number of deposits should not be greater than the capacity of the tree")
)

// DepositTree is the Merkle tree representation of deposits.
type DepositTree struct {
	tree                    MerkleTreeNode
	depositCount            uint64 // number of deposits in the tree, reference implementation calls this mix_in_length.
	finalizedExecutionBlock executionBlock
}

type executionBlock struct {
	Hash  [32]byte
	Depth uint64
}

// NewDepositTree creates an empty deposit tree.
func NewDepositTree() *DepositTree {
	var leaves [][32]byte
	merkle := create(leaves, DepositContractDepth)
	return &DepositTree{
		tree:                    merkle,
		depositCount:            0,
		finalizedExecutionBlock: executionBlock{},
	}
}

// GetSnapshot returns a deposit tree snapshot.
func (d *DepositTree) GetSnapshot() (DepositTreeSnapshot, error) {
	if d.finalizedExecutionBlock == (executionBlock{}) {
		return DepositTreeSnapshot{}, ErrEmptyExecutionBlock
	}
	var finalized [][32]byte
	depositCount, finalized := d.tree.GetFinalized(finalized)
	return fromTreeParts(finalized, depositCount, d.finalizedExecutionBlock)
}

// fromSnapshot returns a deposit tree from a deposit tree snapshot.
//
//nolint:unused
func fromSnapshot(snapshot DepositTreeSnapshot) (*DepositTree, error) {
	root, err := snapshot.CalculateRoot()
	if err != nil {
		return nil, err
	}
	if snapshot.depositRoot != root {
		return nil, ErrInvalidSnapshotRoot
	}
	if snapshot.depositCount >= math.PowerOf2(uint64(DepositContractDepth)) {
		return nil, ErrTooManyDeposits
	}
	tree, err := fromSnapshotParts(snapshot.finalized, snapshot.depositCount, DepositContractDepth)
	if err != nil {
		return nil, err
	}
	if snapshot.depositCount == 0 {
		return nil, ErrNoDeposits
	}
	return &DepositTree{
		tree:                    tree,
		depositCount:            snapshot.depositCount,
		finalizedExecutionBlock: snapshot.executionBlock,
	}, nil
}

// finalize marks a deposit as finalized.
func (d *DepositTree) finalize(eth1data *eth.Eth1Data, executionBlockHeight uint64) error {
	var blockHash [32]byte
	copy(blockHash[:], eth1data.BlockHash)
	d.finalizedExecutionBlock = executionBlock{
		Hash:  blockHash,
		Depth: executionBlockHeight,
	}
	_, err := d.tree.Finalize(eth1data.DepositCount, DepositContractDepth)
	if err != nil {
		return err
	}
	return nil
}

// getProof returns the deposit tree proof.
func (d *DepositTree) getProof(index uint64) ([32]byte, [][32]byte, error) {
	if d.depositCount <= 0 {
		return [32]byte{}, nil, ErrInvalidMixInLength
	}
	finalizedDeposits, _ := d.tree.GetFinalized([][32]byte{})
	if finalizedDeposits != 0 {
		finalizedDeposits = finalizedDeposits - 1
	}
	if index <= finalizedDeposits {
		return [32]byte{}, nil, ErrInvalidIndex
	}
	leaf, proof := generateProof(d.tree, index, DepositContractDepth)
	var mixInLength [32]byte
	copy(mixInLength[:], bytesutil.Uint64ToBytesLittleEndian32(d.depositCount))
	proof = append(proof, mixInLength)
	return leaf, proof, nil
}

// getRoot returns the root of the deposit tree.
func (d *DepositTree) getRoot() [32]byte {
	var enc [32]byte
	binary.LittleEndian.PutUint64(enc[:], d.depositCount)

	root := d.tree.GetRoot()
	fmt.Printf("Our deposit snapshot tree has deposit count %#x, and premixin %#x\n", enc, root)
	return hash.Hash(append(root[:], enc[:]...))
}

// PushLeaf adds a new leaf to the tree.
func (d *DepositTree) PushLeaf(leaf [32]byte) error {
	var err error
	d.tree, err = d.tree.PushLeaf(leaf, DepositContractDepth)
	if err != nil {
		return err
	}
	d.depositCount++
	return nil
}

func (d *DepositTree) Insert(item []byte, index int) error {
	var err error
	var leaf [32]byte
	copy(leaf[:], item[:32])
	err = d.PushLeaf(leaf)
	if err != nil {
		return err
	}
	return nil
}

func (d *DepositTree) HashTreeRoot() ([32]byte, error) {
	root := d.getRoot()
	if root == [32]byte{} {
		return [32]byte{}, errors.New("could not retrieve hash tree root")
	}
	return root, nil
}

func (d *DepositTree) NumOfItems() int {
	// TODO: discuss usefulness?
	return int(d.depositCount)
}

func (d *DepositTree) MerkleProof(index int) ([][]byte, error) {
	_, proof, err := d.getProof(uint64(index))
	if err != nil {
		return nil, err
	}
	byteSlices := make([][]byte, len(proof))
	for i, p := range proof {
		byteSlices[i] = p[:]
	}
	return byteSlices, nil
}
