package casper

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/prysmaticlabs/prysm/beacon-chain/params"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// ShuffleValidatorsToCommittees shuffles validator indices and splits them by slot and shard.
func ShuffleValidatorsToCommittees(seed common.Hash, activeValidators []*pb.ValidatorRecord, dynasty uint64, crosslinkStartShard uint64) ([]*pb.ShardAndCommitteeArray, error) {
	indices := ActiveValidatorIndices(activeValidators, dynasty)

	// split the shuffled list for slot.
	shuffledValidators, err := utils.ShuffleIndices(seed, indices)
	if err != nil {
		return nil, err
	}

	return splitBySlotShard(shuffledValidators, crosslinkStartShard), nil
}

// splitBySlotShard splits the validator list into evenly sized committees and assigns each
// committee to a slot and a shard. If the validator set is large, multiple committees are assigned
// to a single slot and shard. If the validator set is small, a single committee is assigned to a shard
// across multiple slots. See getCommitteeParams for more details.
func splitBySlotShard(shuffledValidators []uint32, crosslinkStartShard uint64) []*pb.ShardAndCommitteeArray {
	committeesPerSlot, slotsPerCommittee := getCommitteeParams(len(shuffledValidators))

	committeBySlotAndShard := []*pb.ShardAndCommitteeArray{}

	// split the validator indices by slot.
	validatorsBySlot := utils.SplitIndices(shuffledValidators, params.CycleLength)
	for i, validatorsForSlot := range validatorsBySlot {
		shardCommittees := []*pb.ShardAndCommittee{}
		validatorsByShard := utils.SplitIndices(validatorsForSlot, committeesPerSlot)
		shardStart := int(crosslinkStartShard) + i*committeesPerSlot/slotsPerCommittee

		for j, validatorsForShard := range validatorsByShard {
			shardID := (shardStart + j) % params.ShardCount
			shardCommittees = append(shardCommittees, &pb.ShardAndCommittee{
				ShardId:   uint64(shardID),
				Committee: validatorsForShard,
			})
		}

		committeBySlotAndShard = append(committeBySlotAndShard, &pb.ShardAndCommitteeArray{
			ArrayShardAndCommittee: shardCommittees,
		})
	}

	return committeBySlotAndShard
}

// getCommitteeParams calculates the parameters for ShuffleValidatorsToCommittees.
// If numActiveValidators > CycleLength * MinCommitteeSize, multiple committees are selected
// to attest the same shard in a single slot.
// If numActiveValidators < CycleLength * MinCommitteeSize, committees span across multiple slots
// to attest the same shard.
func getCommitteeParams(numValidators int) (committeesPerSlot, slotsPerCommittee int) {
	if numValidators >= params.CycleLength*params.MinCommiteeSize {
		committeesPerSlot := numValidators/(params.CycleLength*params.MinCommiteeSize*2) + 1
		return committeesPerSlot, 1
	}

	slotsPerCommittee = 1
	for numValidators*slotsPerCommittee < params.MinCommiteeSize*params.CycleLength &&
		slotsPerCommittee < params.CycleLength {
		slotsPerCommittee *= 2
	}

	return 1, slotsPerCommittee
}
