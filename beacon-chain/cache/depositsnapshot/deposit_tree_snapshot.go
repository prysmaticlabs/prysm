package depositsnapshot

import (
	"crypto/sha256"

	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
)

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

func fromTreeParts(finalised [][32]byte, depositCount uint64, executionBlock ExecutionBlock) DepositTreeSnapshot {
	return DepositTreeSnapshot{
		finalized:            finalised,
		depositRoot:          Zerohashes[0],
		depositCount:         depositCount,
		executionBlockHash:   executionBlock.Hash,
		executionBlockHeight: executionBlock.Depth,
	}
}
