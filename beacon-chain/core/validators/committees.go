package validators

import (
	"fmt"

	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bitutil"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// ShuffleValidatorRegistryToCommittees shuffles validator indices and splits them by slot and shard.
func ShuffleValidatorRegistryToCommittees(
	seed [32]byte,
	validators []*pb.ValidatorRecord,
	crosslinkStartShard uint64,
	slot uint64,
) ([]*pb.ShardAndCommitteeArray, error) {
	indices := ActiveValidatorIndices(validators, slot)
	// split the shuffled list for slot.
	shuffledValidatorRegistry, err := utils.ShuffleIndices(seed, indices)
	if err != nil {
		return nil, err
	}
	return splitBySlotShard(shuffledValidatorRegistry, crosslinkStartShard), nil
}

// splitBySlotShard splits the validator list into evenly sized committees and assigns each
// committee to a slot and a shard. If the validator set is large, multiple committees are assigned
// to a single slot and shard. See getCommitteesPerSlot for more details.
func splitBySlotShard(shuffledValidatorRegistry []uint32, crosslinkStartShard uint64) []*pb.ShardAndCommitteeArray {
	committeesPerSlot := getCommitteesPerSlot(uint64(len(shuffledValidatorRegistry)))
	committeBySlotAndShard := []*pb.ShardAndCommitteeArray{}

	// split the validator indices by slot.
	validatorsBySlot := utils.SplitIndices(shuffledValidatorRegistry, params.BeaconConfig().EpochLength)
	for i, validatorsForSlot := range validatorsBySlot {
		shardCommittees := []*pb.ShardAndCommittee{}
		validatorsByShard := utils.SplitIndices(validatorsForSlot, committeesPerSlot)
		shardStart := crosslinkStartShard + uint64(i)*committeesPerSlot

		for j, validatorsForShard := range validatorsByShard {
			shardID := (shardStart + uint64(j)) % params.BeaconConfig().ShardCount
			shardCommittees = append(shardCommittees, &pb.ShardAndCommittee{
				Shard:     shardID,
				Committee: validatorsForShard,
			})
		}

		committeBySlotAndShard = append(committeBySlotAndShard, &pb.ShardAndCommitteeArray{
			ArrayShardAndCommittee: shardCommittees,
		})
	}
	return committeBySlotAndShard
}

// getCommitteesPerSlot calculates the parameters for ShuffleValidatorRegistryToCommittees.
// The minimum value for committeesPerSlot is 1.
// Otherwise, the value for committeesPerSlot is the smaller of
// numActiveValidatorRegistry / CycleLength /  (MinCommitteeSize*2) + 1 or
// ShardCount / CycleLength.
func getCommitteesPerSlot(numActiveValidatorRegistry uint64) uint64 {
	cycleLength := params.BeaconConfig().EpochLength
	boundOnValidatorRegistry := numActiveValidatorRegistry/cycleLength/(params.BeaconConfig().TargetCommitteeSize*2) + 1
	boundOnShardCount := params.BeaconConfig().ShardCount / cycleLength
	// Ensure that comitteesPerSlot is at least 1.
	if boundOnShardCount == 0 {
		return 1
	} else if boundOnValidatorRegistry > boundOnShardCount {
		return boundOnShardCount
	}
	return boundOnValidatorRegistry
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
//     # Find the relevant committee
//     shard_committees = get_shard_committees_at_slot(state, attestation_data.slot)
//     shard_committee = [x for x in shard_committees if x.shard == attestation_data.shard][0]
//     assert len(participation_bitfield) == ceil_div8(len(shard_committee.committee))
//
//     # Find the participating attesters in the committee
//     participants = []
//     for i, validator_index in enumerate(shard_committee.committee):
//         participation_bit = (participation_bitfield[i//8] >> (7 - (i % 8))) % 2
//         if participation_bit == 1:
//             participants.append(validator_index)
//     return participants
func AttestationParticipants(
	state *pb.BeaconState,
	attestationData *pb.AttestationData,
	participationBitfield []byte) ([]uint32, error) {

	// Find the relevant committee.
	shardCommittees, err := ShardAndCommitteesAtSlot(state, attestationData.Slot)
	if err != nil {
		return nil, err
	}

	var participants *pb.ShardAndCommittee
	for _, committee := range shardCommittees.ArrayShardAndCommittee {
		if committee.Shard == attestationData.Shard {
			participants = committee
			break
		}
	}
	if len(participationBitfield) != mathutil.CeilDiv8(len(participants.Committee)) {
		return nil, fmt.Errorf(
			"wanted participants bitfield length %d, got: %d",
			len(participants.Committee), len(participationBitfield))
	}

	// Find the participating attesters in the committee.
	var participantIndices []uint32
	for i, validatorIndex := range participants.Committee {
		bitSet, err := bitutil.CheckBit(participationBitfield, i)
		if err != nil {
			return nil, fmt.Errorf("could not get participant bitfield: %v", err)
		}
		if bitSet {
			participantIndices = append(participantIndices, validatorIndex)
		}
	}
	return participantIndices, nil
}
