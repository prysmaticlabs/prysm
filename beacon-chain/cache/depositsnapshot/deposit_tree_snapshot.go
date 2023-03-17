package depositsnapshot

import (
	"crypto/sha256"

	"github.com/pkg/errors"
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
			root = sha256.Sum256(append(ds.finalized[index][:], root[:]...))
		} else {
			root = sha256.Sum256(append(root[:], Zerohashes[i][:]...))
		}
		size >>= 1
	}
	return sha256.Sum256(append(root[:], bytesutil.Uint64ToBytesLittleEndian32(ds.depositCount)...)), nil
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
