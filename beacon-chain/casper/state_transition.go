package casper

import (
	"github.com/prysmaticlabs/prysm/beacon-chain/params"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bitutil"
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

	cycleLength := params.GetConfig().CycleLength

	if 3*blockVoteBalance >= 2*totalDeposits {
		if slot > justifiedSlot {
			justifiedSlot = slot
		}
		justifiedStreak++
	} else {
		justifiedStreak = 0
	}

	newFinalizedSlot := slot - cycleLength - 1

	if slot > cycleLength && justifiedStreak >= cycleLength+1 && newFinalizedSlot > finalizedSlot {
		finalizedSlot = newFinalizedSlot
	}

	return justifiedSlot, finalizedSlot, justifiedStreak
}

func ApplyCrosslinkRewardsAndPenalties(
	crosslinkRecords []*pb.CrosslinkRecord,
	currentSlot uint64,
	indices []uint32,
	attestation *pb.AggregatedAttestation,
	dynasty uint64,
	validators []*pb.ValidatorRecord,
	totalBalance uint64,
	voteBalance uint64) {
	rewardQuotient := RewardQuotient(validators)
	for _, attesterIndex := range indices {
		timeSinceLastConfirmation := currentSlot - crosslinkRecords[attestation.Shard].GetSlot()

		if crosslinkRecords[attestation.Shard].GetDynasty() != dynasty {
			if bitutil.CheckBit(attestation.AttesterBitfield, int(attesterIndex)) {
				RewardValidatorCrosslink(totalBalance, voteBalance, rewardQuotient, validators[attesterIndex])
			} else {
				PenaliseValidatorCrosslink(timeSinceLastConfirmation, rewardQuotient, validators[attesterIndex])
			}
		}
	}
}

func ProcessBalancesInCrosslink(slot uint64, voteBalance uint64, totalBalance uint64,
	dynasty uint64, attestation *pb.AggregatedAttestation, crosslinkRecords []*pb.CrosslinkRecord) []*pb.CrosslinkRecord {

	// if 2/3 of committee voted on this crosslink, update the crosslink
	// with latest dynasty number, shard block hash, and slot number.

	voteMajority := 3*voteBalance >= 2*totalBalance
	if voteMajority && dynasty > crosslinkRecords[attestation.Shard].Dynasty {
		crosslinkRecords[attestation.Shard] = &pb.CrosslinkRecord{
			Dynasty:        dynasty,
			ShardBlockHash: attestation.ShardBlockHash,
			Slot:           slot,
		}
	}
	return crosslinkRecords
}
