package stateutil

import (
	"github.com/OffchainLabs/prysm/v7/encoding/ssz"
)

// HeadVoteParticipationRoot computes the merkle root of the per slot head vote committee bitvectors.
func HeadVoteParticipationRoot(participation [][]byte) ([32]byte, error) {
	roots := make([][32]byte, len(participation))
	for i, bits := range participation {
		chunks, err := ssz.PackByChunk([][]byte{bits})
		if err != nil {
			return [32]byte{}, err
		}
		roots[i], err = ssz.BitwiseMerkleize(chunks, uint64(len(chunks)), uint64(len(chunks)))
		if err != nil {
			return [32]byte{}, err
		}
	}
	return ssz.MerkleizeVector(roots, uint64(len(roots))), nil
}
