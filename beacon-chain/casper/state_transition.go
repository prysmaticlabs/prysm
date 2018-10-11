package casper

import (
	"github.com/prysmaticlabs/prysm/beacon-chain/params"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func StateRecalculation(
	lastStateRecalcSlot uint64,
	validators []*pb.ValidatorRecord,
	recentBlockHashes [][32]byte,
	blockVoteCache map[[32]byte]*utils.VoteCache,
	timeSinceFinality uint64,
	totalDeposits uint64,
	justifiedStreak uint64,
	justifiedSlot uint64,
	enableRewardChecking bool) []*pb.ValidatorRecord {

	var LastStateRecalculationSlotCycleBack uint64
	var blockVoteBalance uint64
	var newValidators []*pb.ValidatorRecord
	var finalizedSlot uint64

	if lastStateRecalcSlot < params.GetConfig().CycleLength {
		LastStateRecalculationSlotCycleBack = 0
	} else {
		LastStateRecalculationSlotCycleBack = lastStateRecalcSlot - params.GetConfig().CycleLength
	}

	if !enableRewardChecking {
		newValidators = validators
	}

	// walk through all the slots from LastStateRecalculationSlot - cycleLength to LastStateRecalculationSlot - 1.
	for i := uint64(0); i < params.GetConfig().CycleLength; i++ {
		var voterIndices []uint32

		slot := LastStateRecalculationSlotCycleBack + i
		blockHash := recentBlockHashes[i]
		if _, ok := blockVoteCache[blockHash]; ok {
			blockVoteBalance = blockVoteCache[blockHash].VoteTotalDeposit
			voterIndices = blockVoteCache[blockHash].VoterIndices

			// Apply Rewards for each slot.
			if enableRewardChecking {
				newValidators = CalculateRewards(
					slot,
					voterIndices,
					validators,
					blockVoteBalance,
					timeSinceFinality)
			}
		} else {
			blockVoteBalance = 0
		}

		// TODO(#542): This should have been total balance of the validators in the slot committee.
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
		/*
			if enableCrossLinks {
				newCrosslinks, err = c.processCrosslinks(aState.PendingAttestations(), slot, block.SlotNumber())
				if err != nil {
					return nil, nil, err
				}
			}
		*/
	}

	return newValidators
}
