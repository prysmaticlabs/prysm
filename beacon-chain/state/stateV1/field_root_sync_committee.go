package stateV1

import ethereum_beacon_p2p_v1 "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"

// syncCommitteeRoot computes the HashTreeRoot Merkleization of
// a SyncCommitteeRoot struct according to the eth2
// Simple Serialize specification.
func syncCommitteeRoot(committee *ethereum_beacon_p2p_v1.SyncCommittee) ([32]byte, error) {
	return committee.HashTreeRoot()
}
