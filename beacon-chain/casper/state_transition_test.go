package casper

import (
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/params"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"

	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
)

func TestTallyVoteBalances(t *testing.T) {

	var validators []*pb.ValidatorRecord
	var blockHash [32]byte

	blockVoteCache := make(map[[32]byte]*utils.VoteCache)
	initialBalance := uint64(1e9)
	for i := 0; i < 1000; i++ {
		validator := &pb.ValidatorRecord{
			WithdrawalShard: 0,
			Balance:         initialBalance}

		validators = append(validators, validator)
	}

	validators[20].Status = uint64(params.Active)
	validators[10].Status = uint64(params.Active)

	voteCache := &utils.VoteCache{
		VoterIndices:     []uint32{20, 10},
		VoteTotalDeposit: 300,
	}
	copy(blockHash[:], []byte{'t', 'e', 's', 't', 'i', 'n', 'g'})

	blockVoteCache[blockHash] = voteCache

	zeroBalance, _ := TallyVoteBalances([32]byte{}, 10, blockVoteCache, validators, 2, true)

	if zeroBalance != 0 {
		t.Fatalf("votes have been calculated despite blockhash not existing in cache")
	}

	voteBalance, newValidators := TallyVoteBalances(blockHash, 10, blockVoteCache, validators, 2, true)
	if voteBalance != 300 {
		t.Fatalf("vote balances is not the amount expected %d", voteBalance)
	}

	if newValidators[1].Balance != initialBalance {
		t.Fatalf("validator balance changed %d ", newValidators[1].Balance)
	}

	if newValidators[20].Balance == initialBalance {
		t.Fatalf("validator balance not changed %d ", newValidators[20].Balance)
	}

	if newValidators[10].Balance == initialBalance {
		t.Errorf("validator balance not changed %d ", newValidators[10].Balance)
	}
}

func TestFinalizeAndJustifySlots(t *testing.T) {
	slot := uint64(10)
	justifiedSlot := uint64(8)
	finalizedSlot := uint64(6)
	justifiedStreak := uint64(2)
	blockVoteBalance := uint64(2e9)
	totalDeposit := uint64(4e9)

	justifiedSlot, finalizedSlot, justifiedStreak = FinalizeAndJustifySlots(slot, justifiedSlot, finalizedSlot,
		justifiedStreak, blockVoteBalance, totalDeposit)

	if justifiedSlot != 8 {
		t.Fatalf("justified slot has been updated %d", justifiedSlot)
	}

	if justifiedStreak != 0 {
		t.Fatalf("justified streak not updated %d", justifiedStreak)
	}

	if finalizedSlot != 6 {
		t.Fatalf("finalized slot changed when it was not supposed to %d", finalizedSlot)
	}

	blockVoteBalance = uint64(3e9)

	justifiedSlot, finalizedSlot, justifiedStreak = FinalizeAndJustifySlots(slot, justifiedSlot, finalizedSlot,
		justifiedStreak, blockVoteBalance, totalDeposit)

	if justifiedSlot != 10 {
		t.Fatalf("justified slot has not been updated %d", justifiedSlot)
	}

	if justifiedStreak != 1 {
		t.Fatalf("justified streak not updated %d", justifiedStreak)
	}

	if finalizedSlot != 6 {
		t.Fatalf("finalized slot changed when it was not supposed to %d", finalizedSlot)
	}

	slot = 100
	justifiedStreak = 70

	justifiedSlot, finalizedSlot, justifiedStreak = FinalizeAndJustifySlots(slot, justifiedSlot, finalizedSlot,
		justifiedStreak, blockVoteBalance, totalDeposit)

	if justifiedSlot != 100 {
		t.Fatalf("justified slot has not been updated %d", justifiedSlot)
	}

	if justifiedStreak != 71 {
		t.Fatalf("justified streak not updated %d", justifiedStreak)
	}

	if finalizedSlot == 6 {
		t.Fatalf("finalized slot not updated when it was supposed to %d", finalizedSlot)
	}

}
