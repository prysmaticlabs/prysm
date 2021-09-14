package stateutil

import (
	"encoding/binary"

	"github.com/prysmaticlabs/prysm/encoding/bytes"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/htrutils"
)

// BlockHeaderRoot computes the HashTreeRoot Merkleization of
// a BeaconBlockHeader struct according to the Ethereum
// Simple Serialize specification.
func BlockHeaderRoot(header *ethpb.BeaconBlockHeader) ([32]byte, error) {
	fieldRoots := make([][]byte, 5)
	if header != nil {
		headerSlotBuf := make([]byte, 8)
		binary.LittleEndian.PutUint64(headerSlotBuf, uint64(header.Slot))
		headerSlotRoot := bytes.ToBytes32(headerSlotBuf)
		fieldRoots[0] = headerSlotRoot[:]
		proposerIdxBuf := make([]byte, 8)
		binary.LittleEndian.PutUint64(proposerIdxBuf, uint64(header.ProposerIndex))
		proposerIndexRoot := bytes.ToBytes32(proposerIdxBuf)
		fieldRoots[1] = proposerIndexRoot[:]
		parentRoot := bytes.ToBytes32(header.ParentRoot)
		fieldRoots[2] = parentRoot[:]
		stateRoot := bytes.ToBytes32(header.StateRoot)
		fieldRoots[3] = stateRoot[:]
		bodyRoot := bytes.ToBytes32(header.BodyRoot)
		fieldRoots[4] = bodyRoot[:]
	}
	return htrutils.BitwiseMerkleize(hashutil.CustomSHA256Hasher(), fieldRoots, uint64(len(fieldRoots)), uint64(len(fieldRoots)))
}
