package casper

import (
	"github.com/prysmaticlabs/prysm/beacon-chain/params"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TallyVoteBalances(
	blockHash [32]byte,
	slot uint64,
	blockVoteCache map[[32]byte]*utils.VoteCache,
	validators []*pb.ValidatorRecord,
	timeSinceFinality uint64,
	enableRewardChecking bool) (uint64, []*pb.ValidatorRecord) {
	var blockVoteBalance uint64

	if _, ok := blockVoteCache[blockHash]; ok {
		blockVoteBalance = blockVoteCache[blockHash].VoteTotalDeposit
		voterIndices := blockVoteCache[blockHash].VoterIndices

		// Apply Rewards for each slot.
		if enableRewardChecking {
			validators = CalculateRewards(
				slot,
				voterIndices,
				validators,
				blockVoteBalance,
				timeSinceFinality)
		}
	} else {
		blockVoteBalance = 0
	}

	return blockVoteBalance, validators
}

func FinalizeAndJustifySlots(
	slot uint64, justifiedSlot uint64, finalizedSlot uint64,
	justifiedStreak uint64, blockVoteBalance uint64, totalDeposits uint64) (uint64, uint64, uint64) {

	if 3*blockVoteBalance >= 2*totalDeposits {
		if slot > justifiedSlot {
			justifiedSlot = slot
		}
		justifiedStreak++
	} else {
		justifiedStreak = 0
	}

	if slot > params.GetConfig().CycleLength && justifiedStreak >= params.GetConfig().CycleLength+1 && slot-params.GetConfig().CycleLength-1 > finalizedSlot {
		finalizedSlot = slot - params.GetConfig().CycleLength - 1
	}

	return justifiedSlot, finalizedSlot, justifiedStreak
}
