package casper

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	
	
)

// ShuffleValidatorsToCommittees shuffles validator indices and splits them by slot and shard.
func ShuffleValidatorsToCommittees(seed common.Hash, validators []*pb.ValidatorRecord, crosslinkStartShard uint64) ([]*pb.ShardAndCommitteeArray, error) {
	indices := ActiveValidatorIndices(validators)

	// split the shuffled list for slot.
	shuffledValidators, err := utils.ShuffleIndices(seed, indices)
	if err != nil {
		return nil, err
	}

	return splitBySlotShard(shuffledValidators, crosslinkStartShard), nil
}

// InitialShardAndCommitteesForSlots initialises the committees for shards by shuffling the validators
// and assigning them to specific shards.
func InitialShardAndCommitteesForSlots(validators []*pb.ValidatorRecord) ([]*pb.ShardAndCommitteeArray, error) {
	seed := make([]byte, 0, 32)
	committees, err := ShuffleValidatorsToCommittees(common.BytesToHash(seed), validators, 1)
	if err != nil {
		return nil, err
	}

	// Initialize with 3 cycles of the same committees.
	initialCommittees := make([]*pb.ShardAndCommitteeArray, 0, 3*params.GetBeaconConfig().CycleLength)
	initialCommittees = append(initialCommittees, committees...)
	initialCommittees = append(initialCommittees, committees...)
	initialCommittees = append(initialCommittees, committees...)
	return initialCommittees, nil
}

// splitBySlotShard splits the validator list into evenly sized committees and assigns each
// committee to a slot and a shard. If the validator set is large, multiple committees are assigned
// to a single slot and shard. See getCommitteesPerSlot for more details.
func splitBySlotShard(shuffledValidators []uint32, crosslinkStartShard uint64) []*pb.ShardAndCommitteeArray {
	committeesPerSlot := getCommitteesPerSlot(uint64(len(shuffledValidators)))

	committeBySlotAndShard := []*pb.ShardAndCommitteeArray{}

	// split the validator indices by slot.
	validatorsBySlot := utils.SplitIndices(shuffledValidators, params.GetBeaconConfig().CycleLength)
	for i, validatorsForSlot := range validatorsBySlot {
		shardCommittees := []*pb.ShardAndCommittee{}
		validatorsByShard := utils.SplitIndices(validatorsForSlot, committeesPerSlot)
		shardStart := crosslinkStartShard + uint64(i)*committeesPerSlot

		for j, validatorsForShard := range validatorsByShard {
			shardID := (shardStart + uint64(j)) % params.GetBeaconConfig().ShardCount
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

// getCommitteesPerSlot calculates the parameters for ShuffleValidatorsToCommittees.
// The minimum value for committeesPerSlot is 1.
// Otherwise, the value for committeesPerSlot is the smaller of
// numActiveValidators / CycleLength /  (MinCommitteeSize*2) + 1 or
// ShardCount / CycleLength.
func getCommitteesPerSlot(numActiveValidators uint64) uint64 {
	cycleLength := params.GetBeaconConfig().CycleLength
	boundOnValidators := numActiveValidators/cycleLength/(params.GetBeaconConfig().MinCommitteeSize*2) + 1
	boundOnShardCount := params.GetBeaconConfig().ShardCount / cycleLength

	// Ensure that comitteesPerSlot is at least 1.
	if boundOnShardCount == 0 {
		return 1
	} else if boundOnValidators > boundOnShardCount {
		return boundOnShardCount
	}
	return boundOnValidators
}
