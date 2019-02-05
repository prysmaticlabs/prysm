// Package validators contains libraries to shuffle validators
// and retrieve active validator indices from a given slot
// or an attestation. It also provides helper functions to locate
// validator based on pubic key.
package validators

import (
	"fmt"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bitutil"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
)

// CrosslinkCommittee defines the validator committee of slot and shard combinations.
type CrosslinkCommittee struct {
	Committee []uint64
	Shard     uint64
}

// AttestationParticipants returns the attesting participants indices.
//
// Spec pseudocode definition:
//   def get_attestation_participants(state: BeaconState,
//     attestation_data: AttestationData,
//     participation_bitfield: bytes) -> List[int]:
//     """
//     Returns the participant indices at for the ``attestation_data`` and ``participation_bitfield``.
//     """
//
//     # Find the committee in the list with the desired shard
//     crosslink_committees = get_crosslink_committees_at_slot(state, attestation_data.slot)
//	   assert attestation_data.shard in [shard for _, shard in crosslink_committees]
//     crosslink_committee = [committee for committee,
//     		shard in crosslink_committees if shard == attestation_data.shard][0]
//     assert len(participation_bitfield) == ceil_div8(len(shard_committee))
//
//     # Find the participating attesters in the committee
//     participants = []
//     for i, validator_index in enumerate(crosslink_committee):
//         aggregation_bit = (aggregation_bitfield[i // 8] >> (7 - (i % 8))) % 2
//         if aggregation_bit == 1:
//             participants.append(validator_index)
//     return participants
func AttestationParticipants(
	state *pb.BeaconState,
	attestationData *pb.AttestationData,
	AggregationBitfield []byte) ([]uint64, error) {

	// Find the relevant committee.
	crosslinkCommittees, err := helpers.CrosslinkCommitteesAtSlot(state, attestationData.Slot, false)
	if err != nil {
		return nil, err
	}

	var committee []uint64
	for _, crosslinkCommittee := range crosslinkCommittees {
		if crosslinkCommittee.Shard == attestationData.Shard {
			committee = crosslinkCommittee.Committee
			break
		}
	}
	if len(AggregationBitfield) != mathutil.CeilDiv8(len(committee)) {
		return nil, fmt.Errorf(
			"wanted participants bitfield length %d, got: %d",
			mathutil.CeilDiv8(len(committee)),
			len(AggregationBitfield))
	}

	// Find the participating validators in the committee.
	var participants []uint64
	for i, validatorIndex := range committee {
		bitSet, err := bitutil.CheckBit(AggregationBitfield, i)
		if err != nil {
			return nil, fmt.Errorf("could not get participant bitfield: %v", err)
		}
		if bitSet {
			participants = append(participants, validatorIndex)
		}
	}
	return participants, nil
}
