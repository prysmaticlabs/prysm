package helper

import (
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
)

// IndexedAttestations converts attestation to index committee form.
//
// Spec pseudocode definition:
//   def get_indexed_attestation(beacon_state: BeaconState, attestation: Attestation) -> AttestationAndCommittee:
//    committee = get_beacon_committee(beacon_state, attestation.data.slot, attestation.data.index)
//    return AttestationAndCommittee(committee, attestation)
func IndexedAttestation(state *pb.BeaconState, attestation *ethpb.ShardAttestation) (*ethpb.AttestationAndCommittee, error) {
	committee, err := helpers.BeaconCommittee(state, attestation.Data.Slot, attestation.Data.Index)
	if err != nil {
		return nil, err
	}
	return &ethpb.AttestationAndCommittee{
		Attestation: attestation,
		Committee:   committee,
	}, nil
}
