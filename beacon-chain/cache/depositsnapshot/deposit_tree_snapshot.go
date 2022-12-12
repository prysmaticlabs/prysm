package depositsnapshot

import (
	"crypto/sha256"

	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
)

// DepositTreeSnapshot represents the data used to create a
// deposit tree given a snapshot.
type DepositTreeSnapshot struct {
	finalized      [][32]byte
	depositRoot    [32]byte
	depositCount   uint64
	ExecutionBlock ExecutionBlock
}

// CalculateRoot returns the root of a deposit tree snapshot.
func (ds *DepositTreeSnapshot) CalculateRoot() [32]byte {
	size := ds.depositCount
	index := len(ds.finalized)
	root := Zerohashes[0]
	for i := 0; i < DepositContractDepth; i++ {
		if (size & 1) == 1 {
			index -= 1
			root = sha256.Sum256(append(ds.finalized[index][:], root[:]...))
		} else {
			root = sha256.Sum256(append(root[:], Zerohashes[i][:]...))
		}
		size >>= 1
	}
	return sha256.Sum256(append(root[:], bytesutil.Uint64ToBytesLittleEndian(ds.depositCount)...))
}

// fromTreeParts constructs the deposit tree from pre-existing data.
func fromTreeParts(finalised [][32]byte, depositCount uint64, executionBlock ExecutionBlock) DepositTreeSnapshot {
	return DepositTreeSnapshot{
		finalized:      finalised,
		depositRoot:    Zerohashes[0],
		depositCount:   depositCount,
		ExecutionBlock: executionBlock,
	}
}
