package casper

import (
	"bytes"
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	b "github.com/prysmaticlabs/prysm/shared/bytes"
	"github.com/prysmaticlabs/prysm/shared/params"
)

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

func TestCrosslinks(t *testing.T) {
	totalBalance := uint64(5e9)
	voteBalance := uint64(4e9)

	crossLinks := []*pb.CrosslinkRecord{
		{
			ShardBlockHash: []byte{'A'},
			Slot:           10,
		},
		{
			ShardBlockHash: []byte{'A'},
			Slot:           10,
		},
	}

	attestation := &pb.AggregatedAttestation{
		Slot:             10,
		Shard:            1,
		ShardBlockHash:   []byte{'B'},
		AttesterBitfield: []byte{100, 128, 8},
	}

	crossLinks = ProcessCrosslink(10, voteBalance, totalBalance, attestation, crossLinks)
	crossLinks = ProcessCrosslink(10, voteBalance, totalBalance, attestation, crossLinks)

	if !bytes.Equal(crossLinks[1].GetShardBlockHash(), []byte{'B'}) {
		t.Errorf("shard blockhash not saved in crosslink record %v", crossLinks[1].GetShardBlockHash())
	}

}

func TestProcessSpecialRecords(t *testing.T) {

	specialRecords := []*pb.SpecialRecord{
		{Kind: uint32(params.Logout), Data: [][]byte{b.Bytes8(4)}}, // Validator 4
		{Kind: uint32(params.Logout), Data: [][]byte{b.Bytes8(5)}}, // Validator 5
	}

	validators := make([]*pb.ValidatorRecord, 10)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.ValidatorRecord{Status: uint64(params.Active)}
	}

	newValidators, err := ProcessSpecialRecords(99, validators, specialRecords)
	if err != nil {
		t.Fatalf("Failed to call process special records %v", err)
	}
	if newValidators[4].Status != uint64(params.PendingExit) {
		t.Error("Validator 4 status is not PendingExit")
	}
	if newValidators[4].ExitSlot != 99 {
		t.Error("Validator 4 exit slot is not 99")
	}
	if newValidators[5].Status != uint64(params.PendingExit) {
		t.Error("Validator 5 status is not PendingExit")
	}
	if newValidators[5].ExitSlot != 99 {
		t.Error("Validator 5 exit slot is not 99")
	}
}
