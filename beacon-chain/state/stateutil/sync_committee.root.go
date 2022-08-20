package stateutil

import (
	"github.com/prysmaticlabs/prysm/v3/config/features"
	"github.com/prysmaticlabs/prysm/v3/crypto/hash"
	"github.com/prysmaticlabs/prysm/v3/crypto/hash/htr"
	"github.com/prysmaticlabs/prysm/v3/encoding/ssz"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
)

// SyncCommitteeRoot computes the HashTreeRoot Merkleization of a committee root.
// a SyncCommitteeRoot struct according to the eth2
// Simple Serialize specification.
func SyncCommitteeRoot(committee *ethpb.SyncCommittee) ([32]byte, error) {
	hasher := hash.CustomSHA256Hasher()
	var fieldRoots [][32]byte
	if committee == nil {
		return [32]byte{}, nil
	}

	// Field 1:  Vector[BLSPubkey, SYNC_COMMITTEE_SIZE]
	pubKeyRoots := make([][32]byte, 0)
	for _, pubkey := range committee.Pubkeys {
		r, err := merkleizePubkey(hasher, pubkey)
		if err != nil {
			return [32]byte{}, err
		}
		pubKeyRoots = append(pubKeyRoots, r)
	}
	pubkeyRoot, err := ssz.BitwiseMerkleize(hasher, pubKeyRoots, uint64(len(pubKeyRoots)), uint64(len(pubKeyRoots)))
	if err != nil {
		return [32]byte{}, err
	}

	// Field 2: BLSPubkey
	aggregateKeyRoot, err := merkleizePubkey(hasher, committee.AggregatePubkey)
	if err != nil {
		return [32]byte{}, err
	}
	fieldRoots = [][32]byte{pubkeyRoot, aggregateKeyRoot}

	return ssz.BitwiseMerkleize(hasher, fieldRoots, uint64(len(fieldRoots)), uint64(len(fieldRoots)))
}

func merkleizePubkey(hasher ssz.HashFn, pubkey []byte) ([32]byte, error) {
	chunks, err := ssz.PackByChunk([][]byte{pubkey})
	if err != nil {
		return [32]byte{}, err
	}
	var pubKeyRoot [32]byte
	if features.Get().EnableVectorizedHTR {
		outputChunk := make([][32]byte, 1)
		htr.VectorizedSha256(chunks, outputChunk)
		pubKeyRoot = outputChunk[0]
	} else {
		pubKeyRoot, err = ssz.BitwiseMerkleize(hasher, chunks, uint64(len(chunks)), uint64(len(chunks)))
		if err != nil {
			return [32]byte{}, err
		}
	}
	return pubKeyRoot, nil
}
