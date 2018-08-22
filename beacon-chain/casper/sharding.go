package casper

import (
	"github.com/prysmaticlabs/prysm/beacon-chain/params"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
)

type beaconCommittee struct {
	shardID   int
	committee []int
}

// ValidatorsByHeightShard splits a shuffled validator list by height and by shard,
// it ensures there's enough validators per height and per shard, if not, it'll skip
// some heights and shards.
func ValidatorsByHeightShard(crystallized *types.CrystallizedState) ([]*beaconCommittee, error) {
	indices := ActiveValidatorIndices(crystallized)
	var committeesPerSlot int
	var slotsPerCommittee int
	var committees []*beaconCommittee

	if len(indices) >= params.CycleLength*params.MinCommiteeSize {
		committeesPerSlot = len(indices)/params.CycleLength/(params.MinCommiteeSize*2) + 1
		slotsPerCommittee = 1
	} else {
		committeesPerSlot = 1
		slotsPerCommittee = 1
		for len(indices)*slotsPerCommittee < params.MinCommiteeSize && slotsPerCommittee < params.CycleLength {
			slotsPerCommittee *= 2
		}
	}

	// split the shuffled list for heights.
	shuffledList, err := utils.ShuffleIndices(crystallized.DynastySeed(), indices)
	if err != nil {
		return nil, err
	}

	heightList := utils.SplitIndices(shuffledList, params.CycleLength)

	// split the shuffled height list for shards
	for i, subList := range heightList {
		shardList := utils.SplitIndices(subList, params.MinCommiteeSize)
		for _, shardIndex := range shardList {
			shardID := int(crystallized.CrosslinkingStartShard()) + i*committeesPerSlot/slotsPerCommittee
			committees = append(committees, &beaconCommittee{
				shardID:   shardID,
				committee: shardIndex,
			})
		}
	}
	return committees, nil
}
