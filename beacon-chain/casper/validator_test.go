package casper

import (
	"bytes"
	"math/big"
	"reflect"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/params"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bitutil"
)

func TestHasVoted(t *testing.T) {
	// Setting bit field to 11111111.
	pendingAttestation := &pb.AggregatedAttestation{
		AttesterBitfield: []byte{255},
	}

	for i := 0; i < len(pendingAttestation.AttesterBitfield); i++ {
		voted, err := bitutil.CheckBit(pendingAttestation.AttesterBitfield, i)
		if err != nil {
			t.Errorf("checking bit failed at index: %d with : %v", i, err)
		}

		if !voted {
			t.Error("validator voted but received didn't vote")
		}
	}

	// Setting bit field to 01010101.
	pendingAttestation = &pb.AggregatedAttestation{
		AttesterBitfield: []byte{85},
	}

	for i := 0; i < len(pendingAttestation.AttesterBitfield); i++ {
		voted, err := bitutil.CheckBit(pendingAttestation.AttesterBitfield, i)
		if err != nil {
			t.Errorf("checking bit failed at index: %d : %v", i, err)
		}

		if i%2 == 0 && voted {
			t.Error("validator didn't vote but received voted")
		}
		if i%2 == 1 && !voted {
			t.Error("validator voted but received didn't vote")
		}
	}
}

func TestInitialValidators(t *testing.T) {
	validators := InitialValidators()
	for _, validator := range validators {
		if validator.GetBalance() != uint64(params.GetConfig().DepositSize) {
			t.Fatalf("deposit size of validator is not expected %d", validator.GetBalance())
		}
		if validator.GetStatus() != uint64(params.Active) {
			t.Errorf("validator status is not active: %d", validator.GetStatus())
		}
	}
}
func TestValidatorIndices(t *testing.T) {
	data := &pb.CrystallizedState{
		Validators: []*pb.ValidatorRecord{
			{Pubkey: []byte{}, Status: uint64(params.Active)},            // active.
			{Pubkey: []byte{}, Status: uint64(params.Active)},            // active.
			{Pubkey: []byte{}, Status: uint64(params.Active)},            // active.
			{Pubkey: []byte{}, Status: uint64(params.Active)},            // active.
			{Pubkey: []byte{}, Status: uint64(params.Active)},            // active.
			{Pubkey: []byte{}, Status: uint64(params.PendingActivation)}, // queued.
		},
		ValidatorSetChangeSlot: 1,
	}

	if !reflect.DeepEqual(ActiveValidatorIndices(data.Validators), []uint32{0, 1, 2, 3, 4}) {
		t.Errorf("active validator indices should be [0 1 2 3 4], got: %v", ActiveValidatorIndices(data.Validators))
	}
	if !reflect.DeepEqual(QueuedValidatorIndices(data.Validators), []uint32{5}) {
		t.Errorf("queued validator indices should be [5], got: %v", QueuedValidatorIndices(data.Validators))
	}
	if len(ExitedValidatorIndices(data.Validators)) != 0 {
		t.Errorf("exited validator indices to be empty, got: %v", ExitedValidatorIndices(data.Validators))
	}

	data = &pb.CrystallizedState{
		Validators: []*pb.ValidatorRecord{
			{Pubkey: []byte{}, Status: uint64(params.Active)},            // active.
			{Pubkey: []byte{}, Status: uint64(params.Active)},            // active.
			{Pubkey: []byte{}, Status: uint64(params.PendingActivation)}, // queued.
			{Pubkey: []byte{}, Status: uint64(params.PendingActivation)}, // queued.
			{Pubkey: []byte{}, Status: uint64(params.PendingExit)},       // exited.
			{Pubkey: []byte{}, Status: uint64(params.PendingExit)},       // exited.
		},
	}

	if !reflect.DeepEqual(ActiveValidatorIndices(data.Validators), []uint32{0, 1}) {
		t.Errorf("active validator indices should be [0, 1], got: %v", ActiveValidatorIndices(data.Validators))
	}
	if !reflect.DeepEqual(QueuedValidatorIndices(data.Validators), []uint32{2, 3}) {
		t.Errorf("queued validator indices should be [2, 3], got: %v", QueuedValidatorIndices(data.Validators))
	}
	if !reflect.DeepEqual(ExitedValidatorIndices(data.Validators), []uint32{4, 5}) {
		t.Errorf("exited validator indices should be [4, 5], got: %v", ExitedValidatorIndices(data.Validators))
	}
}

func TestAreAttesterBitfieldsValid(t *testing.T) {
	attestation := &pb.AggregatedAttestation{
		AttesterBitfield: []byte{'F'},
	}

	indices := []uint32{0, 1, 2, 3, 4, 5, 6, 7}

	isValid := AreAttesterBitfieldsValid(attestation, indices)
	if !isValid {
		t.Fatalf("expected validation to pass for bitfield %v and indices %v", attestation, indices)
	}
}

func TestAreAttesterBitfieldsValidFalse(t *testing.T) {
	attestation := &pb.AggregatedAttestation{
		AttesterBitfield: []byte{'F', 'F'},
	}

	indices := []uint32{0, 1, 2, 3, 4, 5, 6, 7}

	isValid := AreAttesterBitfieldsValid(attestation, indices)
	if isValid {
		t.Fatalf("expected validation to fail for bitfield %v and indices %v", attestation, indices)
	}
}

func TestAreAttesterBitfieldsValidZerofill(t *testing.T) {
	attestation := &pb.AggregatedAttestation{
		AttesterBitfield: []byte{'F'},
	}

	indices := []uint32{0, 1, 2, 3, 4, 5, 6}

	isValid := AreAttesterBitfieldsValid(attestation, indices)
	if !isValid {
		t.Fatalf("expected validation to pass for bitfield %v and indices %v", attestation, indices)
	}
}

func TestAreAttesterBitfieldsValidNoZerofill(t *testing.T) {
	attestation := &pb.AggregatedAttestation{
		AttesterBitfield: []byte{'E'},
	}

	indices := []uint32{0, 1, 2, 3, 4, 5, 6}

	isValid := AreAttesterBitfieldsValid(attestation, indices)
	if isValid {
		t.Fatalf("expected validation to fail for bitfield %v and indices %v", attestation, indices)
	}
}

func TestProposerShardAndIndex(t *testing.T) {
	shardCommittees := []*pb.ShardAndCommitteeArray{
		{ArrayShardAndCommittee: []*pb.ShardAndCommittee{
			{Shard: 0, Committee: []uint32{0, 1, 2, 3, 4}},
			{Shard: 1, Committee: []uint32{5, 6, 7, 8, 9}},
		}},
		{ArrayShardAndCommittee: []*pb.ShardAndCommittee{
			{Shard: 2, Committee: []uint32{10, 11, 12, 13, 14}},
			{Shard: 3, Committee: []uint32{15, 16, 17, 18, 19}},
		}},
		{ArrayShardAndCommittee: []*pb.ShardAndCommittee{
			{Shard: 4, Committee: []uint32{20, 21, 22, 23, 24}},
			{Shard: 5, Committee: []uint32{25, 26, 27, 28, 29}},
		}},
	}
	if _, _, err := ProposerShardAndIndex(shardCommittees, 100, 0); err == nil {
		t.Error("ProposerShardAndIndex should have failed with invalid lcs")
	}
	Shard, index, err := ProposerShardAndIndex(shardCommittees, 128, 64)
	if err != nil {
		t.Fatalf("ProposerShardAndIndex failed with %v", err)
	}
	if Shard != 0 {
		t.Errorf("Invalid shard ID. Wanted 0, got %d", Shard)
	}
	if index != 4 {
		t.Errorf("Invalid proposer index. Wanted 4, got %d", index)
	}
}

func TestValidatorIndex(t *testing.T) {
	var validators []*pb.ValidatorRecord
	for i := 0; i < 10; i++ {
		validators = append(validators, &pb.ValidatorRecord{Pubkey: []byte{}, Status: uint64(params.Active)})
	}
	if _, err := ValidatorIndex([]byte("100"), validators); err == nil {
		t.Fatalf("ValidatorIndex should have failed,  there's no validator with pubkey 100")
	}
	validators[5].Pubkey = []byte("100")
	index, err := ValidatorIndex([]byte("100"), validators)
	if err != nil {
		t.Fatalf("call ValidatorIndex failed: %v", err)
	}
	if index != 5 {
		t.Errorf("Incorrect validator index. Wanted 5, Got %v", index)
	}
}

func TestValidatorShard(t *testing.T) {
	var validators []*pb.ValidatorRecord
	for i := 0; i < 21; i++ {
		validators = append(validators, &pb.ValidatorRecord{Pubkey: []byte{}, Status: uint64(params.Active)})
	}
	shardCommittees := []*pb.ShardAndCommitteeArray{
		{ArrayShardAndCommittee: []*pb.ShardAndCommittee{
			{Shard: 0, Committee: []uint32{0, 1, 2, 3, 4, 5, 6}},
			{Shard: 1, Committee: []uint32{7, 8, 9, 10, 11, 12, 13}},
			{Shard: 2, Committee: []uint32{14, 15, 16, 17, 18, 19}},
		}},
	}
	validators[19].Pubkey = []byte("100")
	Shard, err := ValidatorShardID([]byte("100"), validators, shardCommittees)
	if err != nil {
		t.Fatalf("call ValidatorShard failed: %v", err)
	}
	if Shard != 2 {
		t.Errorf("Incorrect validator shard ID. Wanted 2, Got %v", Shard)
	}

	validators[19].Pubkey = []byte{}
	if _, err := ValidatorShardID([]byte("100"), validators, shardCommittees); err == nil {
		t.Fatalf("ValidatorShard should have failed, there's no validator with pubkey 100")
	}

	validators[20].Pubkey = []byte("100")
	if _, err := ValidatorShardID([]byte("100"), validators, shardCommittees); err == nil {
		t.Fatalf("ValidatorShard should have failed, validator indexed at 20 is not in the committee")
	}
}

func TestValidatorSlotAndResponsibility(t *testing.T) {
	var validators []*pb.ValidatorRecord
	for i := 0; i < 61; i++ {
		validators = append(validators, &pb.ValidatorRecord{Pubkey: []byte{}, Status: uint64(params.Active)})
	}
	shardCommittees := []*pb.ShardAndCommitteeArray{
		{ArrayShardAndCommittee: []*pb.ShardAndCommittee{
			{Shard: 0, Committee: []uint32{0, 1, 2, 3, 4, 5, 6}},
			{Shard: 1, Committee: []uint32{7, 8, 9, 10, 11, 12, 13}},
			{Shard: 2, Committee: []uint32{14, 15, 16, 17, 18, 19}},
		}},
		{ArrayShardAndCommittee: []*pb.ShardAndCommittee{
			{Shard: 3, Committee: []uint32{20, 21, 22, 23, 24, 25, 26}},
			{Shard: 4, Committee: []uint32{27, 28, 29, 30, 31, 32, 33}},
			{Shard: 5, Committee: []uint32{34, 35, 36, 37, 38, 39}},
		}},
		{ArrayShardAndCommittee: []*pb.ShardAndCommittee{
			{Shard: 6, Committee: []uint32{40, 41, 42, 43, 44, 45, 46}},
			{Shard: 7, Committee: []uint32{47, 48, 49, 50, 51, 52, 53}},
			{Shard: 8, Committee: []uint32{54, 55, 56, 57, 58, 59}},
		}},
	}
	if _, _, err := ValidatorSlotAndRole([]byte("100"), validators, shardCommittees); err == nil {
		t.Fatalf("ValidatorSlot should have failed, there's no validator with pubkey 100")
	}

	validators[59].Pubkey = []byte("100")
	slot, _, err := ValidatorSlotAndRole([]byte("100"), validators, shardCommittees)
	if err != nil {
		t.Fatalf("call ValidatorSlot failed: %v", err)
	}
	if slot != 2 {
		t.Errorf("Incorrect validator slot ID. Wanted 1, Got %v", slot)
	}

	validators[60].Pubkey = []byte("101")
	if _, _, err := ValidatorSlotAndRole([]byte("101"), validators, shardCommittees); err == nil {
		t.Fatalf("ValidatorSlot should have failed, validator indexed at 60 is not in the committee")
	}
}

func TestTotalActiveValidatorDeposit(t *testing.T) {
	var validators []*pb.ValidatorRecord
	for i := 0; i < 10; i++ {
		validators = append(validators, &pb.ValidatorRecord{Balance: 1e9, Status: uint64(params.Active)})
	}

	expectedTotalDeposit := new(big.Int)
	expectedTotalDeposit.SetString("10000000000", 10)

	totalDeposit := TotalActiveValidatorDeposit(validators)
	if expectedTotalDeposit.Cmp(new(big.Int).SetUint64(totalDeposit)) != 0 {
		t.Fatalf("incorrect total deposit calculated %d", totalDeposit)
	}

	totalDepositETH := TotalActiveValidatorDepositInEth(validators)
	if totalDepositETH != 10 {
		t.Fatalf("incorrect total deposit in ETH calculated %d", totalDepositETH)
	}
}

func TestCommitteeInShardAndSlot(t *testing.T) {

	testCommittee := []uint32{20, 21, 22, 23, 24, 25, 26}

	shardCommittees := []*pb.ShardAndCommitteeArray{
		{ArrayShardAndCommittee: []*pb.ShardAndCommittee{
			{Shard: 0, Committee: []uint32{0, 1, 2, 3, 4, 5, 6}},
			{Shard: 1, Committee: []uint32{7, 8, 9, 10, 11, 12, 13}},
			{Shard: 3, Committee: []uint32{14, 15, 16, 17, 18, 19}},
		}},
		{ArrayShardAndCommittee: []*pb.ShardAndCommittee{
			{Shard: 3, Committee: testCommittee},
			{Shard: 4, Committee: []uint32{27, 28, 29, 30, 31, 32, 33}},
			{Shard: 5, Committee: []uint32{34, 35, 36, 37, 38, 39}},
		}},
		{ArrayShardAndCommittee: []*pb.ShardAndCommittee{
			{Shard: 3, Committee: []uint32{40, 41, 42, 43, 44, 45, 46}},
			{Shard: 7, Committee: []uint32{47, 48, 49, 50, 51, 52, 53}},
			{Shard: 8, Committee: []uint32{54, 55, 56, 57, 58, 59}},
		}},
	}
	_, err := CommitteeInShardAndSlot(2, 5, shardCommittees)
	if err == nil {
		t.Fatalf("function did not return error even though committee for shard does not exist")
	}

	committee, err := CommitteeInShardAndSlot(1, 3, shardCommittees)
	if err != nil {
		t.Fatalf("unable to get committees for shard: %v", err)
	}

	if len(committee) != len(testCommittee) {
		t.Fatalf("the committees are not of the same sizes %d : %d", len(committee), len(testCommittee))
	}
	for i, indice := range committee {
		if indice != testCommittee[i] {
			t.Errorf("retrieved indice is not the same as the one put in the committee %d , %d", indice, testCommittee[i])
		}
	}
}

func TestVotedBalanceInAttestation(t *testing.T) {
	var validators []*pb.ValidatorRecord
	defaultBalance := uint64(1e9)
	for i := 0; i < 100; i++ {
		validators = append(validators, &pb.ValidatorRecord{Balance: defaultBalance, Status: uint64(params.Active)})
	}

	// Calculateing balances with zero votes by attesters.
	attestation := &pb.AggregatedAttestation{
		AttesterBitfield: []byte{0, 0, 0, 0},
	}

	indices := []uint32{4, 8, 10, 14, 30}
	expectedTotalBalance := uint64(len(indices)) * defaultBalance

	totalBalance, voteBalance, err := VotedBalanceInAttestation(validators, indices, attestation)

	if err != nil {
		t.Fatalf("unable to get voted balances in attestation %v", err)
	}

	if totalBalance != expectedTotalBalance {
		t.Errorf("incorrect total balance calculated %d", totalBalance)
	}

	if voteBalance != 0 {
		t.Errorf("incorrect vote balance calculated %d", voteBalance)
	}

	// Calculating balances with 3 votes by attesters.

	newAttestation := &pb.AggregatedAttestation{
		AttesterBitfield: []byte{8, 128, 0, 2},
	}

	expectedTotalBalance = uint64(len(indices)) * defaultBalance

	totalBalance, voteBalance, err = VotedBalanceInAttestation(validators, indices, newAttestation)

	if err != nil {
		t.Fatalf("unable to get voted balances in attestation %v", err)
	}

	if totalBalance != expectedTotalBalance {
		t.Errorf("incorrect total balance calculated %d", totalBalance)
	}

	if voteBalance != defaultBalance*3 {
		t.Errorf("incorrect vote balance calculated %d", voteBalance)
	}

}

func TestAddValidators(t *testing.T) {
	var existingValidators []*pb.ValidatorRecord
	for i := 0; i < 10; i++ {
		existingValidators = append(existingValidators, &pb.ValidatorRecord{Status: uint64(params.Active)})
	}

	// Create a new validator.
	validators := AddPendingValidator(existingValidators, []byte{'A'}, 99, []byte{'B'}, []byte{'C'})

	// The newly added validator should be indexed 10.
	if validators[10].Status != uint64(params.PendingActivation) {
		t.Errorf("Newly added validator should be pending")
	}
	if validators[10].WithdrawalShard != 99 {
		t.Errorf("Newly added validator's withdrawal shard should be 99. Got: %d", validators[10].WithdrawalShard)
	}
	if validators[10].Balance != uint64(params.GetConfig().DepositSize*params.GetConfig().Gwei) {
		t.Errorf("Incorrect deposit size")
	}

	// Set validator 6 to withdrawn
	existingValidators[5].Status = uint64(params.Withdrawn)
	validators = AddPendingValidator(existingValidators, []byte{'D'}, 100, []byte{'E'}, []byte{'F'})

	// The newly added validator should be indexed 5.
	if validators[5].Status != uint64(params.PendingActivation) {
		t.Errorf("Newly added validator should be pending")
	}
	if validators[5].WithdrawalShard != 100 {
		t.Errorf("Newly added validator's withdrawal shard should be 100. Got: %d", validators[10].WithdrawalShard)
	}
	if validators[5].Balance != uint64(params.GetConfig().DepositSize*params.GetConfig().Gwei) {
		t.Errorf("Incorrect deposit size")
	}
}

func TestChangeValidators(t *testing.T) {
	existingValidators := []*pb.ValidatorRecord{
		{Pubkey: []byte{1}, Status: uint64(params.PendingActivation), Balance: uint64(params.GetConfig().DepositSize * params.GetConfig().Gwei), ExitSlot: params.GetConfig().WithdrawalPeriod},
		{Pubkey: []byte{2}, Status: uint64(params.PendingExit), Balance: uint64(params.GetConfig().DepositSize * params.GetConfig().Gwei), ExitSlot: params.GetConfig().WithdrawalPeriod},
		{Pubkey: []byte{3}, Status: uint64(params.PendingActivation), Balance: uint64(params.GetConfig().DepositSize * params.GetConfig().Gwei), ExitSlot: params.GetConfig().WithdrawalPeriod},
		{Pubkey: []byte{4}, Status: uint64(params.PendingExit), Balance: uint64(params.GetConfig().DepositSize * params.GetConfig().Gwei), ExitSlot: params.GetConfig().WithdrawalPeriod},
		{Pubkey: []byte{5}, Status: uint64(params.PendingActivation), Balance: uint64(params.GetConfig().DepositSize * params.GetConfig().Gwei), ExitSlot: params.GetConfig().WithdrawalPeriod},
		{Pubkey: []byte{6}, Status: uint64(params.PendingExit), Balance: uint64(params.GetConfig().DepositSize * params.GetConfig().Gwei), ExitSlot: params.GetConfig().WithdrawalPeriod},
		{Pubkey: []byte{7}, Status: uint64(params.PendingWithdraw), Balance: uint64(params.GetConfig().DepositSize * params.GetConfig().Gwei)},
		{Pubkey: []byte{8}, Status: uint64(params.PendingWithdraw), Balance: uint64(params.GetConfig().DepositSize * params.GetConfig().Gwei)},
		{Pubkey: []byte{9}, Status: uint64(params.Penalized), Balance: uint64(params.GetConfig().DepositSize * params.GetConfig().Gwei)},
		{Pubkey: []byte{10}, Status: uint64(params.Penalized), Balance: uint64(params.GetConfig().DepositSize * params.GetConfig().Gwei)},
		{Pubkey: []byte{11}, Status: uint64(params.Active), Balance: uint64(params.GetConfig().DepositSize * params.GetConfig().Gwei)},
		{Pubkey: []byte{12}, Status: uint64(params.Active), Balance: uint64(params.GetConfig().DepositSize * params.GetConfig().Gwei)},
		{Pubkey: []byte{13}, Status: uint64(params.Active), Balance: uint64(params.GetConfig().DepositSize * params.GetConfig().Gwei)},
		{Pubkey: []byte{14}, Status: uint64(params.Active), Balance: uint64(params.GetConfig().DepositSize * params.GetConfig().Gwei)},
	}

	validators := ChangeValidators(params.GetConfig().WithdrawalPeriod+1, 50*10e9, existingValidators)

	if validators[0].Status != uint64(params.Active) {
		t.Errorf("Wanted status Active. Got: %d", validators[0].Status)
	}
	if validators[0].Balance != uint64(params.GetConfig().DepositSize*params.GetConfig().Gwei) {
		t.Error("Failed to set validator balance")
	}
	if validators[1].Status != uint64(params.PendingWithdraw) {
		t.Errorf("Wanted status PendingWithdraw. Got: %d", validators[1].Status)
	}
	if validators[1].ExitSlot != params.GetConfig().WithdrawalPeriod+1 {
		t.Errorf("Failed to set validator exit slot")
	}
	if validators[2].Status != uint64(params.Active) {
		t.Errorf("Wanted status Active. Got: %d", validators[2].Status)
	}
	if validators[2].Balance != uint64(params.GetConfig().DepositSize*params.GetConfig().Gwei) {
		t.Error("Failed to set validator balance")
	}
	if validators[3].Status != uint64(params.PendingWithdraw) {
		t.Errorf("Wanted status PendingWithdraw. Got: %d", validators[3].Status)
	}
	if validators[3].ExitSlot != params.GetConfig().WithdrawalPeriod+1 {
		t.Errorf("Failed to set validator exit slot")
	}
	// Reach max validation rotation case, this validator couldn't be rotated.
	if validators[5].Status != uint64(params.PendingExit) {
		t.Errorf("Wanted status PendingExit. Got: %d", validators[5].Status)
	}
	if validators[7].Status != uint64(params.Withdrawn) {
		t.Errorf("Wanted status Withdrawn. Got: %d", validators[7].Status)
	}
	if validators[8].Status != uint64(params.Withdrawn) {
		t.Errorf("Wanted status Withdrawn. Got: %d", validators[8].Status)
	}
}

func TestValidatorMinDeposit(t *testing.T) {
	minDeposit := params.GetConfig().MinDeposit * params.GetConfig().Gwei
	currentSlot := uint64(99)
	validators := []*pb.ValidatorRecord{
		{Status: uint64(params.Active), Balance: uint64(minDeposit) + 1},
		{Status: uint64(params.Active), Balance: uint64(minDeposit)},
		{Status: uint64(params.Active), Balance: uint64(minDeposit) - 1},
	}
	newValidators := CheckValidatorMinDeposit(validators, currentSlot)
	if newValidators[0].Status != uint64(params.Active) {
		t.Error("Validator should be active")
	}
	if newValidators[1].Status != uint64(params.Active) {
		t.Error("Validator should be active")
	}
	if newValidators[2].Status != uint64(params.PendingExit) {
		t.Error("Validator should be pending exit")
	}
	if newValidators[2].ExitSlot != currentSlot {
		t.Errorf("Validator's exit slot should be %d got %d", currentSlot, newValidators[2].ExitSlot)
	}
}

func TestMinEmptyValidator(t *testing.T) {
	validators := []*pb.ValidatorRecord{
		{Status: uint64(params.Active)},
		{Status: uint64(params.Withdrawn)},
		{Status: uint64(params.Active)},
	}
	if minEmptyValidator(validators) != 1 {
		t.Errorf("Min vaidator index should be 1")
	}

	validators[1].Status = uint64(params.Active)
	if minEmptyValidator(validators) != -1 {
		t.Errorf("Min vaidator index should be -1")
	}
}

func TestDeepCopyValidators(t *testing.T) {
	var validators []*pb.ValidatorRecord
	defaultValidator := &pb.ValidatorRecord{
		Pubkey:            []byte{'k', 'e', 'y'},
		WithdrawalShard:   2,
		WithdrawalAddress: []byte{'a', 'd', 'd', 'r', 'e', 's', 's'},
		RandaoCommitment:  []byte{'r', 'a', 'n', 'd', 'a', 'o'},
		Balance:           uint64(1e9),
		Status:            uint64(params.Active),
		ExitSlot:          10,
	}
	for i := 0; i < 100; i++ {
		validators = append(validators, defaultValidator)
	}

	newValidatorSet := CopyValidators(validators)

	defaultValidator.Pubkey = []byte{'n', 'e', 'w', 'k', 'e', 'y'}
	defaultValidator.WithdrawalShard = 3
	defaultValidator.WithdrawalAddress = []byte{'n', 'e', 'w', 'a', 'd', 'd', 'r', 'e', 's', 's'}
	defaultValidator.RandaoCommitment = []byte{'n', 'e', 'w', 'r', 'a', 'n', 'd', 'a', 'o'}
	defaultValidator.Balance = uint64(2e9)
	defaultValidator.Status = uint64(params.PendingExit)
	defaultValidator.ExitSlot = 5

	if len(newValidatorSet) != len(validators) {
		t.Fatalf("validator set length is unequal, copy of set failed: %d", len(newValidatorSet))
	}

	for i, validator := range newValidatorSet {
		if bytes.Equal(validator.Pubkey, defaultValidator.Pubkey) {
			t.Errorf("validator with index %d was unable to have their pubkey copied correctly %v", i, validator.Pubkey)
		}

		if validator.WithdrawalShard == defaultValidator.WithdrawalShard {
			t.Errorf("validator with index %d was unable to have their withdrawal shard copied correctly %v", i, validator.WithdrawalShard)
		}

		if bytes.Equal(validator.WithdrawalAddress, defaultValidator.WithdrawalAddress) {
			t.Errorf("validator with index %d was unable to have their withdrawal address copied correctly %v", i, validator.WithdrawalAddress)
		}

		if bytes.Equal(validator.RandaoCommitment, defaultValidator.RandaoCommitment) {
			t.Errorf("validator with index %d was unable to have their randao commitment copied correctly %v", i, validator.RandaoCommitment)
		}

		if validator.Balance == defaultValidator.Balance {
			t.Errorf("validator with index %d was unable to have their balance copied correctly %d", i, validator.Balance)
		}

		if validator.Status == defaultValidator.Status {
			t.Errorf("validator with index %d was unable to have their status copied correctly %d", i, validator.Status)
		}

		if validator.ExitSlot == defaultValidator.ExitSlot {
			t.Errorf("validator with index %d was unable to have their exit slot copied correctly %d", i, validator.ExitSlot)
		}
	}

}
