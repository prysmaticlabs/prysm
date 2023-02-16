package stateutil

import (
	"github.com/prysmaticlabs/prysm/v3/crypto/hash/htr"
	"github.com/prysmaticlabs/prysm/v3/encoding/ssz"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
)

// SyncCommitteeRoot computes the HashTreeRoot Merkleization of a committee root.
// a SyncCommitteeRoot struct according to the eth2
// Simple Serialize specification.
func SyncCommitteeRoot(committee *ethpb.SyncCommittee) ([32]byte, error) {
	var fieldRoots [][32]byte
	if committee == nil {
		return [32]byte{}, nil
	}

	// Field 1:  Vector[BLSPubkey, SYNC_COMMITTEE_SIZE]
	pubKeyRoots := make([][32]byte, 0)
	for _, pubkey := range committee.Pubkeys {
		r, err := merkleizePubkey(pubkey)
		if err != nil {
			return [32]byte{}, err
		}
		pubKeyRoots = append(pubKeyRoots, r)
	}
	pubkeyRoot, err := ssz.BitwiseMerkleize(pubKeyRoots, uint64(len(pubKeyRoots)), uint64(len(pubKeyRoots)))
	if err != nil {
		return [32]byte{}, err
	}

	// Field 2: BLSPubkey
	aggregateKeyRoot, err := merkleizePubkey(committee.AggregatePubkey)
	if err != nil {
		return [32]byte{}, err
	}
	fieldRoots = [][32]byte{pubkeyRoot, aggregateKeyRoot}

	return ssz.BitwiseMerkleize(fieldRoots, uint64(len(fieldRoots)), uint64(len(fieldRoots)))
}

func merkleizePubkey(pubkey []byte) ([32]byte, error) {
	chunks, err := ssz.PackByChunk([][]byte{pubkey})
	if err != nil {
		return [32]byte{}, err
	}
	var pubKeyRoot [32]byte
	outputChunk := make([][32]byte, 1)
	htr.VectorizedSha256(chunks, outputChunk)
	pubKeyRoot = outputChunk[0]
	return pubKeyRoot, nil
}
