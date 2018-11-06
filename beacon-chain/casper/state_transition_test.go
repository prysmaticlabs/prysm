package casper

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/params"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
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

	zeroBalance, _ := TallyVoteBalances([32]byte{}, 10, blockVoteCache, validators, 2)

	if zeroBalance != 0 {
		t.Fatalf("votes have been calculated despite blockhash not existing in cache")
	}

	voteBalance, newValidators := TallyVoteBalances(blockHash, 10, blockVoteCache, validators, 2)
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

func TestApplyCrosslinkRewardsAndPenalties(t *testing.T) {
	var validators []*pb.ValidatorRecord
	initialBalance := uint64(1e9)
	totalBalance := uint64(5e9)
	voteBalance := uint64(4e9)
	indices := []uint32{20, 10}

	for i := 0; i < 1000; i++ {
		validator := &pb.ValidatorRecord{
			WithdrawalShard: 0,
			Balance:         initialBalance}

		validators = append(validators, validator)
	}

	validators[20].Status = uint64(params.Active)
	validators[10].Status = uint64(params.Active)

	crossLinks := []*pb.CrosslinkRecord{
		{
			RecentlyChanged: false,
			ShardBlockHash:  []byte{'A'},
			Slot:            10,
		},
		{
			RecentlyChanged: false,
			ShardBlockHash:  []byte{'B'},
			Slot:            10,
		},
		{
			RecentlyChanged: false,
			ShardBlockHash:  []byte{'C'},
			Slot:            10,
		},
		{
			RecentlyChanged: false,
			ShardBlockHash:  []byte{'D'},
			Slot:            10,
		},
	}

	attestation := &pb.AggregatedAttestation{
		Slot:             10,
		Shard:            1,
		AttesterBitfield: []byte{100, 128, 8},
	}

	ApplyCrosslinkRewardsAndPenalties(crossLinks, 12, indices, attestation, validators, totalBalance, voteBalance)

	if validators[20].Balance <= initialBalance {
		t.Fatalf("validator balance has not been updated %d", validators[20].Balance)
	}

	if validators[10].Balance >= initialBalance {
		t.Fatalf("validator balance has not been updated %d", validators[10].Balance)
	}

	if validators[1].Balance != initialBalance {
		t.Fatalf("validator balance updated when it was not supposed to %d", validators[1].Balance)
	}

}

func TestProcessBalancesInCrosslinks(t *testing.T) {
	totalBalance := uint64(5e9)
	voteBalance := uint64(4e9)

	crossLinks := []*pb.CrosslinkRecord{
		{
			RecentlyChanged: false,
			ShardBlockHash:  []byte{'A'},
			Slot:            10,
		},
		{
			RecentlyChanged: true,
			ShardBlockHash:  []byte{'A'},
			Slot:            10,
		},
	}

	attestation := &pb.AggregatedAttestation{
		Slot:             10,
		Shard:            1,
		ShardBlockHash:   []byte{'B'},
		AttesterBitfield: []byte{100, 128, 8},
	}

	crossLinks = ProcessBalancesInCrosslink(10, voteBalance, totalBalance, attestation, crossLinks)

	if bytes.Equal(crossLinks[1].GetShardBlockHash(), []byte{'B'}) {
		t.Fatal("crosslink updated when it was not supposed to")
	}

	crossLinks[1].RecentlyChanged = false

	crossLinks = ProcessBalancesInCrosslink(10, voteBalance, totalBalance, attestation, crossLinks)

	if !bytes.Equal(crossLinks[1].GetShardBlockHash(), []byte{'B'}) {
		t.Errorf("shard blockhash not saved in crosslink record %v", crossLinks[1].GetShardBlockHash())
	}

}

func TestProcessSpecialRecords(t *testing.T) {
	validator4Index := make([]byte, 8)
	binary.BigEndian.PutUint64(validator4Index, 4)
	validator5Index := make([]byte, 8)
	binary.BigEndian.PutUint64(validator5Index, 5)
	validator6Index := make([]byte, 8)
	binary.BigEndian.PutUint64(validator6Index, 6)
	validator7Index := make([]byte, 8)
	binary.BigEndian.PutUint64(validator7Index, 7)

	specialRecords := []*pb.SpecialRecord{
		{Kind: uint32(params.Logout), Data: [][]byte{validator4Index}},                    // Validator 4
		{Kind: uint32(params.Logout), Data: [][]byte{validator5Index}},                    // Validator 5
		{Kind: uint32(params.RandaoChange), Data: [][]byte{validator6Index, {byte('A')}}}, // Validator 6
		{Kind: uint32(params.RandaoChange), Data: [][]byte{validator7Index, {byte('B')}}}, // Validator 7
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
	if !(bytes.Equal(newValidators[6].RandaoCommitment, []byte{'A'})) {
		t.Error("Failed to set validator 7's randao reveal")
	}
	if !(bytes.Equal(newValidators[7].RandaoCommitment, []byte{'B'})) {
		t.Error("Failed to set validator 8's randao reveal")
	}
}
