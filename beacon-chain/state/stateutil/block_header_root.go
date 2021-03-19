package stateutil

import (
	"encoding/binary"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/htrutils"
)

// BlockHeaderRoot computes the HashTreeRoot Merkleization of
// a BeaconBlockHeader struct according to the eth2
// Simple Serialize specification.
func BlockHeaderRoot(header *ethpb.BeaconBlockHeader) ([32]byte, error) {
	fieldRoots := make([][]byte, 5)
	if header != nil {
		headerSlotBuf := make([]byte, 8)
		binary.LittleEndian.PutUint64(headerSlotBuf, uint64(header.Slot))
		headerSlotRoot := bytesutil.ToBytes32(headerSlotBuf)
		fieldRoots[0] = headerSlotRoot[:]
		proposerIdxBuf := make([]byte, 8)
		binary.LittleEndian.PutUint64(proposerIdxBuf, uint64(header.ProposerIndex))
		proposerIndexRoot := bytesutil.ToBytes32(proposerIdxBuf)
		fieldRoots[1] = proposerIndexRoot[:]
		parentRoot := bytesutil.ToBytes32(header.ParentRoot)
		fieldRoots[2] = parentRoot[:]
		stateRoot := bytesutil.ToBytes32(header.StateRoot)
		fieldRoots[3] = stateRoot[:]
		bodyRoot := bytesutil.ToBytes32(header.BodyRoot)
		fieldRoots[4] = bodyRoot[:]
	}
	return htrutils.BitwiseMerkleize(hashutil.CustomSHA256Hasher(), fieldRoots, uint64(len(fieldRoots)), uint64(len(fieldRoots)))
}
