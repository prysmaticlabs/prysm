package casper

import (
	"github.com/prysmaticlabs/prysm/beacon-chain/params"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

type shardAttestation struct {
	Shard          uint64
	shardBlockHash [32]byte
}

func TallyVoteBalances(
	blockVoteBalance uint64,
	blockHash [32]byte,
	slot uint64,
	blockVoteCache map[[32]byte]*utils.VoteCache,
	validators []*pb.ValidatorRecord,
	timeSinceFinality uint64,
	enableRewardChecking bool) (uint64, []*pb.ValidatorRecord) {

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

func StateRecalculation(
	lastStateRecalcSlot uint64,
	validators []*pb.ValidatorRecord,
	recentBlockHashes [][32]byte,
	blockVoteCache map[[32]byte]*utils.VoteCache,
	timeSinceFinality uint64,
	totalDeposits uint64,
	justifiedStreak uint64,
	justifiedSlot uint64,
	enableRewardChecking bool,
	enableCrossLinks bool) []*pb.ValidatorRecord {

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

/*
func ProcessCrosslinks(
	pendingAttestations []*pb.AggregatedAttestation,
	validators []*pb.ValidatorRecord,
	dynasty uint64,
	crosslinks []*pb.CrosslinkRecord,
	slot uint64,
	currentSlot uint64) {

	crosslinkRecords := copyCrosslinks(crosslinks)
	rewardQuotient := RewardQuotient(validators)

	shardAttestationBalance := map[shardAttestation]uint64{}
	for _, attestation := range pendingAttestations {
		indices, err := c.getAttesterIndices(attestation)
		if err != nil {
			return nil, err
		}

		shardBlockHash := [32]byte{}
		copy(shardBlockHash[:], attestation.ShardBlockHash)
		shardAtt := shardAttestation{
			Shard:          attestation.Shard,
			shardBlockHash: shardBlockHash,
		}
		if _, ok := shardAttestationBalance[shardAtt]; !ok {
			shardAttestationBalance[shardAtt] = 0
		}

		// find the total and vote balance of the shard committee.
		var totalBalance uint64
		var voteBalance uint64
		for _, attesterIndex := range indices {
			// find balance of validators who voted.
			if bitutil.CheckBit(attestation.AttesterBitfield, int(attesterIndex)) {
				voteBalance += validators[attesterIndex].Balance
			}
			// add to total balance of the committee.
			totalBalance += validators[attesterIndex].Balance
		}

		for _, attesterIndex := range indices {
			timeSinceLastConfirmation := currentSlot - crosslinkRecords[attestation.Shard].GetSlot()

			if crosslinkRecords[attestation.Slot].GetDynasty() != dynasty {
				if bitutil.CheckBit(attestation.AttesterBitfield, int(attesterIndex)) {
					casper.RewardValidatorCrosslink(totalBalance, voteBalance, rewardQuotient, validators[attesterIndex])
				} else {
					casper.PenaliseValidatorCrosslink(timeSinceLastConfirmation, rewardQuotient, validators[attesterIndex])
				}
			}
		}

		shardAttestationBalance[shardAtt] += voteBalance

		// if 2/3 of committee voted on this crosslink, update the crosslink
		// with latest dynasty number, shard block hash, and slot number.
		if 3*voteBalance >= 2*totalBalance && dynasty > crosslinkRecords[attestation.Shard].Dynasty {
			crosslinkRecords[attestation.Shard] = &pb.CrosslinkRecord{
				Dynasty:        dynasty,
				ShardBlockHash: attestation.ShardBlockHash,
				Slot:           slot,
			}
		}
	}
	return crosslinkRecords, nil

}
*/

func copyCrosslinks(existing []*pb.CrosslinkRecord) []*pb.CrosslinkRecord {
	new := make([]*pb.CrosslinkRecord, len(existing))
	for i := 0; i < len(existing); i++ {
		oldCL := existing[i]
		newBlockhash := make([]byte, len(oldCL.ShardBlockHash))
		copy(newBlockhash, oldCL.ShardBlockHash)
		newCL := &pb.CrosslinkRecord{
			Dynasty:        oldCL.Dynasty,
			ShardBlockHash: newBlockhash,
			Slot:           oldCL.Slot,
		}
		new[i] = newCL
	}

	return new
}
