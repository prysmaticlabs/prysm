package depositsnapshot

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/container/trie"
	"github.com/prysmaticlabs/prysm/v4/crypto/hash"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
)

var (
	// ErrZeroIndex occurs when the value of index is 0.
	ErrZeroIndex = errors.New("index should be greater than 0")
)

// DepositTreeSnapshot represents the data used to create a
// deposit tree given a snapshot.
//
//nolint:unused
type DepositTreeSnapshot struct {
	finalized      [][32]byte
	depositRoot    [32]byte
	depositCount   uint64
	executionBlock executionBlock
}

// CalculateRoot returns the root of a deposit tree snapshot.
func (ds *DepositTreeSnapshot) CalculateRoot() ([32]byte, error) {
	size := ds.depositCount
	index := len(ds.finalized)
	root := Zerohashes[0]
	for i := 0; i < DepositContractDepth; i++ {
		if (size & 1) == 1 {
			if index == 0 {
				return [32]byte{}, ErrZeroIndex
			}
			index--
			root = hash.Hash(append(ds.finalized[index][:], root[:]...))
		} else {
			root = hash.Hash(append(root[:], Zerohashes[i][:]...))
		}
		size >>= 1
	}
	fmt.Println("in snapshot calc root, deposit count is", ds.depositCount)
	fmt.Printf("in snapshot calc, root before mixin is %#x\n", root)
	return hash.Hash(append(root[:], bytesutil.Uint64ToBytesLittleEndian32(ds.depositCount)...)), nil
}

func CalculateRootEquivalent(leaves [][32]byte) ([32]byte, error) {
	depth := uint64(DepositContractDepth)
	layers := make([][][]byte, depth+1)
	transformedLeaves := make([][]byte, len(leaves))
	for i := range leaves {
		transformedLeaves[i] = leaves[i][:]
	}
	layers[0] = transformedLeaves
	for i := uint64(0); i < depth; i++ {
		if len(layers[i])%2 == 1 {
			layers[i] = append(layers[i], trie.ZeroHashes[i][:])
		}
		updatedValues := make([][]byte, 0)
		for j := 0; j < len(layers[i]); j += 2 {
			concat := hash.Hash(append(layers[i][j], layers[i][j+1]...))
			updatedValues = append(updatedValues, concat[:])
		}
		layers[i+1] = updatedValues
	}
	preMixInRoot := layers[len(layers)-1][0]
	fmt.Printf("in equivalent, root before mixin is %#x\n", preMixInRoot)
	return bytesutil.ToBytes32(preMixInRoot), nil
}

// fromTreeParts constructs the deposit tree from pre-existing data.
//
//nolint:unused
func fromTreeParts(finalised [][32]byte, depositCount uint64, executionBlock executionBlock) (DepositTreeSnapshot, error) {
	snapshot := DepositTreeSnapshot{
		finalized:      finalised,
		depositRoot:    Zerohashes[0],
		depositCount:   depositCount,
		executionBlock: executionBlock,
	}
	root, err := snapshot.CalculateRoot()
	if err != nil {
		return snapshot, ErrInvalidSnapshotRoot
	}
	snapshot.depositRoot = root
	return snapshot, nil
}
