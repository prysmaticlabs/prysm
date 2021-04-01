package state_altair

import pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"

// syncCommitteeRoot computes the HashTreeRoot Merkleization of
// a SyncCommitteeRoot struct according to the eth2
// Simple Serialize specification.
func syncCommitteeRoot(committee *pb.SyncCommittee) ([32]byte, error) {
	return committee.HashTreeRoot()
}
