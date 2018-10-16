package casper

import (
	"strconv"

	"github.com/prysmaticlabs/prysm/beacon-chain/params"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bitutil"
)

// TallyVoteBalances calculates all the votes behind a block and then rewards validators for their
// participation in voting for that block.
func TallyVoteBalances(
	blockHash [32]byte,
	slot uint64,
	blockVoteCache map[[32]byte]*utils.VoteCache,
	validators []*pb.ValidatorRecord,
	timeSinceFinality uint64,
	enableRewardChecking bool) (uint64, []*pb.ValidatorRecord) {

	cache, ok := blockVoteCache[blockHash]

	if !ok {
		return 0, validators
	}

	blockVoteBalance := cache.VoteTotalDeposit
	voterIndices := cache.VoterIndices
	if enableRewardChecking {
		validators = CalculateRewards(slot, voterIndices, validators,
			blockVoteBalance, timeSinceFinality)
	}

	return blockVoteBalance, validators
}

// FinalizeAndJustifySlots justifies slots and sets the justified streak according to Casper FFG
// conditions. It also finalizes slots when the conditions are fulfilled.
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

// ApplyCrosslinkRewardsAndPenalties applies the appropriate rewards and penalties according to the attestation
// for a shard.
func ApplyCrosslinkRewardsAndPenalties(
	crosslinkRecords []*pb.CrosslinkRecord,
	slot uint64,
	attesterIndices []uint32,
	attestation *pb.AggregatedAttestation,
	validators []*pb.ValidatorRecord,
	totalBalance uint64,
	voteBalance uint64) error {

	rewardQuotient := RewardQuotient(validators)

	for _, attesterIndex := range attesterIndices {
		timeSinceLastConfirmation := slot - crosslinkRecords[attestation.Shard].GetSlot()

		if !crosslinkRecords[attestation.Shard].GetRecentlyChanged() {
			checkBit, err := bitutil.CheckBit(attestation.AttesterBitfield, int(attesterIndex))
			if err != nil {
				return err
			}

			if checkBit {
				RewardValidatorCrosslink(totalBalance, voteBalance, rewardQuotient, validators[attesterIndex])
			} else {
				PenaliseValidatorCrosslink(timeSinceLastConfirmation, rewardQuotient, validators[attesterIndex])
			}
		}
	}
	return nil
}

// ProcessBalancesInCrosslink checks the vote balances and if there is a supermajority it sets the crosslink
// for that shard.
func ProcessBalancesInCrosslink(slot uint64, voteBalance uint64, totalBalance uint64,
	attestation *pb.AggregatedAttestation, crosslinkRecords []*pb.CrosslinkRecord) []*pb.CrosslinkRecord {

	// if 2/3 of committee voted on this crosslink, update the crosslink
	// with latest dynasty number, shard block hash, and slot number.

	voteMajority := 3*voteBalance >= 2*totalBalance
	if voteMajority && !crosslinkRecords[attestation.Shard].RecentlyChanged {
		crosslinkRecords[attestation.Shard] = &pb.CrosslinkRecord{
			RecentlyChanged: true,
			ShardBlockHash:  attestation.ShardBlockHash,
			Slot:            slot,
		}
	}
	return crosslinkRecords
}

// ProcessSpeicalRecords processes the pending special record objects,
// this is called during cyrstallized state transition.
func ProcessSpeicalRecords(slotNumber uint64, validators []*pb.ValidatorRecord, pendingSpecials []*pb.SpecialRecord) ([]*pb.ValidatorRecord, error) {
	// For each special record object in active state.
	for _, specialRecord := range pendingSpecials {

		// Covers validators submitted logouts from last cycle.
		if specialRecord.Kind == uint32(params.Logout) {
			validatorIndex, err := strconv.Atoi(string(specialRecord.Data[0]))
			if err != nil {
				return nil, err
			}
			exitedValidator := ExitValidator(validators[validatorIndex], slotNumber, false)
			validators[validatorIndex] = exitedValidator
			// TODO(#633): Verify specialRecord.Data[1] as signature. BLSVerify(pubkey=validator.pubkey, msg=hash(LOGOUT_MESSAGE + bytes8(version))
		}

		// Covers RANDAO updates for all the validators from last cycle.
		if specialRecord.Kind == uint32(params.RandaoChange) {
			validatorIndex, err := strconv.Atoi(string(specialRecord.Data[0]))
			if err != nil {
				return nil, err
			}
			validators[validatorIndex].RandaoCommitment = specialRecord.Data[1]
		}
	}
	return validators, nil
}
