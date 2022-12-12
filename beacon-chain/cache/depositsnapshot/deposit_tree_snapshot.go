package depositsnapshot

import (
	"crypto/sha256"

	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
)

type DepositTreeSnapshot struct {
	Finalized            [][32]byte
	DepositRoot          [32]byte
	DepositCount         uint64
	ExecutionBlockHash   [32]byte
	ExecutionBlockHeight uint64
}

func (ds *DepositTreeSnapshot) CalculateRoot() [32]byte {
	size := ds.DepositCount
	index := len(ds.Finalized)
	root := Zerohashes[0]
	for i := 0; i < DepositContractDepth; i++ {
		if (size & 1) == 1 {
			index -= 1
			root = sha256.Sum256(append(ds.Finalized[index][:], root[:]...))
		} else {
			root = sha256.Sum256(append(root[:], Zerohashes[i][:]...))
		}
		size >>= 1
	}
	return sha256.Sum256(append(root[:], bytesutil.Uint64ToBytesLittleEndian32(ds.DepositCount)...))
}

func fromTreeParts(finalised [][32]byte, depositCount uint64, executionBlock ExecutionBlock) DepositTreeSnapshot {
	snapshot := DepositTreeSnapshot{
		Finalized:            finalised,
		DepositRoot:          Zerohashes[0],
		DepositCount:         depositCount,
		ExecutionBlockHash:   executionBlock.Hash,
		ExecutionBlockHeight: executionBlock.Depth,
	}
	snapshot.DepositRoot = snapshot.CalculateRoot()
	return snapshot
}
