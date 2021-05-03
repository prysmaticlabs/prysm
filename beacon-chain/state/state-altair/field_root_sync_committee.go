package state_altair

import (
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/htrutils"
)

// syncCommitteeRoot  computes the HashTreeRoot Merkleization of
// a SyncCommitteeRoot struct according to the eth2
// Simple Serialize specification.
func syncCommitteeRoot(committee *pb.SyncCommittee) ([32]byte, error) {
	hasher := hashutil.CustomSHA256Hasher()
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
	pubkeyRoot, err := htrutils.BitwiseMerkleizeArrays(hasher, pubKeyRoots, uint64(len(pubKeyRoots)), uint64(len(pubKeyRoots)))
	if err != nil {
		return [32]byte{}, err
	}

	// Field 2: Vector[BLSPubkey, SYNC_COMMITTEE_SIZE // SYNC_PUBKEYS_PER_AGGREGATE]
	aggregateKeyRoots := make([][32]byte, 0)
	for _, pubkey := range committee.PubkeyAggregates {
		r, err := merkleizePubkey(hasher, pubkey)
		if err != nil {
			return [32]byte{}, err
		}
		aggregateKeyRoots = append(aggregateKeyRoots, r)
	}
	aggregateKeyRoot, err := htrutils.BitwiseMerkleizeArrays(hasher, aggregateKeyRoots, uint64(len(aggregateKeyRoots)), uint64(len(aggregateKeyRoots)))
	if err != nil {
		return [32]byte{}, err
	}
	fieldRoots = [][32]byte{pubkeyRoot, aggregateKeyRoot}

	return htrutils.BitwiseMerkleizeArrays(hasher, fieldRoots, uint64(len(fieldRoots)), uint64(len(fieldRoots)))
}

func merkleizePubkey(hasher htrutils.HashFn, pubkey []byte) ([32]byte, error) {
	chunks, err := htrutils.Pack([][]byte{pubkey})
	if err != nil {
		return [32]byte{}, err
	}
	return htrutils.BitwiseMerkleize(hasher, chunks, uint64(len(chunks)), uint64(len(chunks)))
}
