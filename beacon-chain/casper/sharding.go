package casper

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/prysmaticlabs/prysm/beacon-chain/params"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// ShuffleValidatorsToCommittees shuffles validator indices and splits them by slot and shard.
func ShuffleValidatorsToCommittees(seed common.Hash, activeValidators []*pb.ValidatorRecord, crosslinkStartShard uint64) ([]*pb.ShardAndCommitteeArray, error) {
	indices := ActiveValidatorIndices(activeValidators)

	// split the shuffled list for slot.
	shuffledValidators, err := utils.ShuffleIndices(seed, indices)
	if err != nil {
		return nil, err
	}

	return splitBySlotShard(shuffledValidators, crosslinkStartShard), nil
}

// splitBySlotShard splits the validator list into evenly sized committees and assigns each
// committee to a slot and a shard. If the validator set is large, multiple committees are assigned
// to a single slot and shard. See getCommitteeParams for more details.
func splitBySlotShard(shuffledValidators []uint32, crosslinkStartShard uint64) []*pb.ShardAndCommitteeArray {
	committeesPerSlot := getCommitteeParams(len(shuffledValidators))

	committeBySlotAndShard := []*pb.ShardAndCommitteeArray{}

	// split the validator indices by slot.
	validatorsBySlot := utils.SplitIndices(shuffledValidators, int(params.GetConfig().CycleLength))
	for i, validatorsForSlot := range validatorsBySlot {
		shardCommittees := []*pb.ShardAndCommittee{}
		validatorsByShard := utils.SplitIndices(validatorsForSlot, committeesPerSlot)
		shardStart := int(crosslinkStartShard) + i*committeesPerSlot

		for j, validatorsForShard := range validatorsByShard {
			shardID := (shardStart + j) % params.GetConfig().ShardCount
			shardCommittees = append(shardCommittees, &pb.ShardAndCommittee{
				Shard:     uint64(shardID),
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
// If numActiveValidators > CycleLength * MinCommitteeSize, committees are based off a max amount
// of len(avs) // CYCLE_LENGTH // (MIN_COMMITTEE_SIZE * 2) + 1 or SHARD_COUNT // CYCLE_LENGTH
func getCommitteeParams(numValidators int) (committeesPerSlot int) {
	cycleLength := int(params.GetConfig().CycleLength)
	boundOnAVS := numValidators/cycleLength/int(params.GetConfig().MinCommiteeSize*2) + 1
	boundOnShardCount := params.GetConfig().ShardCount / cycleLength
	if boundOnAVS > boundOnShardCount {
		return boundOnShardCount
	}
	return boundOnAVS
}
