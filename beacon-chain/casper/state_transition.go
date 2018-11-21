package casper

import (
	"encoding/binary"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/incentives"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bitutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	v "github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
)

// TallyVoteBalances calculates all the votes behind a block and then rewards validators for their
// participation in voting for that block.
func TallyVoteBalances(
	blockHash [32]byte,
	slot uint64,
	blockVoteCache utils.BlockVoteCache,
	validators []*pb.ValidatorRecord,
	timeSinceFinality uint64) (uint64, []*pb.ValidatorRecord) {

	blockVote, ok := blockVoteCache[blockHash]
	if !ok {
		return 0, validators
	}

	blockVoteBalance := blockVote.VoteTotalDeposit
	voterIndices := blockVote.VoterIndices
	activeValidatorIndices := v.ActiveValidatorIndices(validators)
	totalDeposit := v.TotalActiveValidatorDeposit(validators)
	validators = incentives.CalculateRewards(
		slot,
		voterIndices,
		activeValidatorIndices,
		validators,
		totalDeposit,
		blockVoteBalance,
		timeSinceFinality,
	)

	return blockVoteBalance, validators
}

// FinalizeAndJustifySlots justifies slots and sets the justified streak according to Casper FFG
// conditions. It also finalizes slots when the conditions are fulfilled.
func FinalizeAndJustifySlots(
	slot uint64, justifiedSlot uint64, finalizedSlot uint64,
	justifiedStreak uint64, blockVoteBalance uint64, totalDeposits uint64) (uint64, uint64, uint64) {

	cycleLength := params.BeaconConfig().CycleLength

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

	totalDeposit := v.TotalActiveValidatorDeposit(validators)
	rewardQuotient := incentives.RewardQuotient(totalDeposit)

	for _, attesterIndex := range attesterIndices {
		timeSinceLastConfirmation := slot - crosslinkRecords[attestation.Shard].GetSlot()

		checkBit, err := bitutil.CheckBit(attestation.AttesterBitfield, int(attesterIndex))
		if err != nil {
			return err
		}
		if checkBit {
			incentives.RewardValidatorCrosslink(totalBalance, voteBalance, rewardQuotient, validators[attesterIndex])
		} else {
			incentives.PenaliseValidatorCrosslink(timeSinceLastConfirmation, rewardQuotient, validators[attesterIndex])
		}
	}
	return nil
}

// ProcessCrosslink checks the vote balances and if there is a supermajority it sets the crosslink
// for that shard.
func ProcessCrosslink(slot uint64, voteBalance uint64, totalBalance uint64,
	attestation *pb.AggregatedAttestation, crosslinkRecords []*pb.CrosslinkRecord) []*pb.CrosslinkRecord {

	// if 2/3 of committee voted on this crosslink, update the crosslink
	// with latest dynasty number, shard block hash, and slot number.
	voteMajority := 3*voteBalance >= 2*totalBalance
	if voteMajority {
		crosslinkRecords[attestation.Shard] = &pb.CrosslinkRecord{
			ShardBlockHash: attestation.ShardBlockHash,
			Slot:           slot,
		}
	}
	return crosslinkRecords
}

// ProcessSpecialRecords processes the pending special record objects,
// this is called during crystallized state transition.
func ProcessSpecialRecords(slotNumber uint64, validators []*pb.ValidatorRecord, pendingSpecials []*pb.SpecialRecord) ([]*pb.ValidatorRecord, error) {
	// For each special record object in active state.
	for _, specialRecord := range pendingSpecials {

		// Covers validators submitted logouts from last cycle.
		if specialRecord.Kind == uint32(params.Logout) {
			validatorIndex := binary.BigEndian.Uint64(specialRecord.Data[0])
			exitedValidator := v.ExitValidator(validators[validatorIndex], slotNumber, false)
			validators[validatorIndex] = exitedValidator
			// TODO(#633): Verify specialRecord.Data[1] as signature. BLSVerify(pubkey=validator.pubkey, msg=hash(LOGOUT_MESSAGE + bytes8(version))
		}

	}
	return validators, nil
}
