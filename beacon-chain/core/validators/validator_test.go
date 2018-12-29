package validators

import (
	"bytes"
	"math/big"
	"reflect"
	"strings"
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/proto/common"
	"github.com/prysmaticlabs/prysm/shared/bitutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestHasVoted(t *testing.T) {
	// Setting bit field to 11111111.
	pendingAttestation := &pb.Attestation{
		ParticipationBitfield: []byte{255},
	}

	for i := 0; i < len(pendingAttestation.GetParticipationBitfield()); i++ {
		voted, err := bitutil.CheckBit(pendingAttestation.GetParticipationBitfield(), i)
		if err != nil {
			t.Errorf("checking bit failed at index: %d with : %v", i, err)
		}

		if !voted {
			t.Error("validator voted but received didn't vote")
		}
	}

	// Setting bit field to 01010101.
	pendingAttestation = &pb.Attestation{
		ParticipationBitfield: []byte{85},
	}

	for i := 0; i < len(pendingAttestation.GetParticipationBitfield()); i++ {
		voted, err := bitutil.CheckBit(pendingAttestation.GetParticipationBitfield(), i)
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

func TestInitialValidatorRegistry(t *testing.T) {
	validators := InitialValidatorRegistry()
	for _, validator := range validators {
		if validator.GetStatus() != pb.ValidatorRecord_ACTIVE {
			t.Errorf("validator status is not active: %d", validator.GetStatus())
		}
	}
}

func TestAreAttesterBitfieldsValid(t *testing.T) {
	attestation := &pb.Attestation{
		ParticipationBitfield: []byte{'F'},
	}

	indices := []uint32{0, 1, 2, 3, 4, 5, 6, 7}

	isValid := AreAttesterBitfieldsValid(attestation, indices)
	if !isValid {
		t.Fatalf("expected validation to pass for bitfield %v and indices %v", attestation, indices)
	}
}

func TestAreAttesterBitfieldsValidFalse(t *testing.T) {
	attestation := &pb.Attestation{
		ParticipationBitfield: []byte{'F', 'F'},
	}

	indices := []uint32{0, 1, 2, 3, 4, 5, 6, 7}

	isValid := AreAttesterBitfieldsValid(attestation, indices)
	if isValid {
		t.Fatalf("expected validation to fail for bitfield %v and indices %v", attestation, indices)
	}
}

func TestAreAttesterBitfieldsValidZerofill(t *testing.T) {
	attestation := &pb.Attestation{
		ParticipationBitfield: []byte{'F'},
	}

	indices := []uint32{0, 1, 2, 3, 4, 5, 6}

	isValid := AreAttesterBitfieldsValid(attestation, indices)
	if !isValid {
		t.Fatalf("expected validation to pass for bitfield %v and indices %v", attestation, indices)
	}
}

func TestAreAttesterBitfieldsValidNoZerofill(t *testing.T) {
	attestation := &pb.Attestation{
		ParticipationBitfield: []byte{'E'},
	}

	var indices []uint32
	for i := uint32(0); i < uint32(params.BeaconConfig().TargetCommitteeSize)+1; i++ {
		indices = append(indices, i)
	}

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
	shard, index, err := ProposerShardAndIndex(shardCommittees, 128, 65)
	if err != nil {
		t.Fatalf("ProposerShardAndIndex failed with %v", err)
	}
	if shard != 2 {
		t.Errorf("Invalid shard ID. Wanted 2, got %d", shard)
	}
	if index != 0 {
		t.Errorf("Invalid proposer index. Wanted 0, got %d", index)
	}
}

func TestValidatorIndex(t *testing.T) {
	var validators []*pb.ValidatorRecord
	for i := 0; i < 10; i++ {
		validators = append(validators, &pb.ValidatorRecord{Pubkey: []byte{}, Status: pb.ValidatorRecord_ACTIVE})
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
		validators = append(validators, &pb.ValidatorRecord{Pubkey: []byte{}, Status: pb.ValidatorRecord_ACTIVE})
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
		validators = append(validators, &pb.ValidatorRecord{Pubkey: []byte{}, Status: pb.ValidatorRecord_ACTIVE})
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
		validators = append(validators, &pb.ValidatorRecord{Balance: 1e9, Status: pb.ValidatorRecord_ACTIVE})
	}

	expectedTotalDeposit := new(big.Int)
	expectedTotalDeposit.SetString("10000000000", 10)

	totalDeposit := TotalActiveValidatorBalance(validators)
	if expectedTotalDeposit.Cmp(new(big.Int).SetUint64(totalDeposit)) != 0 {
		t.Fatalf("incorrect total deposit calculated %d", totalDeposit)
	}

	totalDepositETH := TotalActiveValidatorDepositInEth(validators)
	if totalDepositETH != 10 {
		t.Fatalf("incorrect total deposit in ETH calculated %d", totalDepositETH)
	}
}

func TestVotedBalanceInAttestation(t *testing.T) {
	var validators []*pb.ValidatorRecord
	defaultBalance := uint64(1e9)
	for i := 0; i < 100; i++ {
		validators = append(validators, &pb.ValidatorRecord{Balance: defaultBalance, Status: pb.ValidatorRecord_ACTIVE})
	}

	// Calculating balances with zero votes by attesters.
	attestation := &pb.Attestation{
		ParticipationBitfield: []byte{0},
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

	newAttestation := &pb.Attestation{
		ParticipationBitfield: []byte{224}, // 128 + 64 + 32
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

func TestAddValidatorRegistry(t *testing.T) {
	var existingValidatorRegistry []*pb.ValidatorRecord
	for i := 0; i < 10; i++ {
		existingValidatorRegistry = append(existingValidatorRegistry, &pb.ValidatorRecord{Status: pb.ValidatorRecord_ACTIVE})
	}

	// Create a new validator.
	validators := AddPendingValidator(existingValidatorRegistry, []byte{'A'}, []byte{'C'}, pb.ValidatorRecord_PENDING_ACTIVATION)

	// The newly added validator should be indexed 10.
	if validators[10].Status != pb.ValidatorRecord_PENDING_ACTIVATION {
		t.Errorf("Newly added validator should be pending")
	}
	if validators[10].Balance != uint64(params.BeaconConfig().MaxDeposit*params.BeaconConfig().Gwei) {
		t.Errorf("Incorrect deposit size")
	}

	// Set validator 6 to withdrawn
	existingValidatorRegistry[5].Status = pb.ValidatorRecord_EXITED_WITHOUT_PENALTY
	validators = AddPendingValidator(existingValidatorRegistry, []byte{'E'}, []byte{'F'}, pb.ValidatorRecord_PENDING_ACTIVATION)

	// The newly added validator should be indexed 5.
	if validators[5].Status != pb.ValidatorRecord_PENDING_ACTIVATION {
		t.Errorf("Newly added validator should be pending")
	}
	if validators[5].Balance != uint64(params.BeaconConfig().MaxDeposit*params.BeaconConfig().Gwei) {
		t.Errorf("Incorrect deposit size")
	}
}

func TestChangeValidatorRegistry(t *testing.T) {
	existingValidatorRegistry := []*pb.ValidatorRecord{
		{Pubkey: []byte{1}, Status: pb.ValidatorRecord_PENDING_ACTIVATION, Balance: uint64(params.BeaconConfig().MaxDeposit * params.BeaconConfig().Gwei), LatestStatusChangeSlot: params.BeaconConfig().MinWithdrawalPeriod},
		{Pubkey: []byte{2}, Status: pb.ValidatorRecord_ACTIVE_PENDING_EXIT, Balance: uint64(params.BeaconConfig().MaxDeposit * params.BeaconConfig().Gwei), LatestStatusChangeSlot: params.BeaconConfig().MinWithdrawalPeriod},
		{Pubkey: []byte{3}, Status: pb.ValidatorRecord_PENDING_ACTIVATION, Balance: uint64(params.BeaconConfig().MaxDeposit * params.BeaconConfig().Gwei), LatestStatusChangeSlot: params.BeaconConfig().MinWithdrawalPeriod},
		{Pubkey: []byte{4}, Status: pb.ValidatorRecord_ACTIVE_PENDING_EXIT, Balance: uint64(params.BeaconConfig().MaxDeposit * params.BeaconConfig().Gwei), LatestStatusChangeSlot: params.BeaconConfig().MinWithdrawalPeriod},
		{Pubkey: []byte{5}, Status: pb.ValidatorRecord_PENDING_ACTIVATION, Balance: uint64(params.BeaconConfig().MaxDeposit * params.BeaconConfig().Gwei), LatestStatusChangeSlot: params.BeaconConfig().MinWithdrawalPeriod},
		{Pubkey: []byte{6}, Status: pb.ValidatorRecord_ACTIVE_PENDING_EXIT, Balance: uint64(params.BeaconConfig().MaxDeposit * params.BeaconConfig().Gwei), LatestStatusChangeSlot: params.BeaconConfig().MinWithdrawalPeriod},
		{Pubkey: []byte{7}, Status: pb.ValidatorRecord_ACTIVE_PENDING_EXIT, Balance: uint64(params.BeaconConfig().MaxDeposit * params.BeaconConfig().Gwei)},
		{Pubkey: []byte{8}, Status: pb.ValidatorRecord_ACTIVE_PENDING_EXIT, Balance: uint64(params.BeaconConfig().MaxDeposit * params.BeaconConfig().Gwei)},
		{Pubkey: []byte{9}, Status: pb.ValidatorRecord_EXITED_WITH_PENALTY, Balance: uint64(params.BeaconConfig().MaxDeposit * params.BeaconConfig().Gwei)},
		{Pubkey: []byte{10}, Status: pb.ValidatorRecord_EXITED_WITH_PENALTY, Balance: uint64(params.BeaconConfig().MaxDeposit * params.BeaconConfig().Gwei)},
		{Pubkey: []byte{11}, Status: pb.ValidatorRecord_ACTIVE, Balance: uint64(params.BeaconConfig().MaxDeposit * params.BeaconConfig().Gwei)},
		{Pubkey: []byte{12}, Status: pb.ValidatorRecord_ACTIVE, Balance: uint64(params.BeaconConfig().MaxDeposit * params.BeaconConfig().Gwei)},
		{Pubkey: []byte{13}, Status: pb.ValidatorRecord_ACTIVE, Balance: uint64(params.BeaconConfig().MaxDeposit * params.BeaconConfig().Gwei)},
		{Pubkey: []byte{14}, Status: pb.ValidatorRecord_ACTIVE, Balance: uint64(params.BeaconConfig().MaxDeposit * params.BeaconConfig().Gwei)},
	}

	validators := ChangeValidatorRegistry(params.BeaconConfig().MinWithdrawalPeriod+1, 50*10e9, existingValidatorRegistry)

	if validators[0].Status != pb.ValidatorRecord_ACTIVE {
		t.Errorf("Wanted status Active. Got: %d", validators[0].Status)
	}
	if validators[0].Balance != uint64(params.BeaconConfig().MaxDeposit*params.BeaconConfig().Gwei) {
		t.Error("Failed to set validator balance")
	}
	if validators[1].Status != pb.ValidatorRecord_ACTIVE_PENDING_EXIT {
		t.Errorf("Wanted status PendingWithdraw. Got: %d", validators[1].Status)
	}
	if validators[1].LatestStatusChangeSlot != params.BeaconConfig().MinWithdrawalPeriod+1 {
		t.Errorf("Failed to set validator lastest status change slot")
	}
	if validators[2].Status != pb.ValidatorRecord_ACTIVE {
		t.Errorf("Wanted status Active. Got: %d", validators[2].Status)
	}
	if validators[2].Balance != uint64(params.BeaconConfig().MaxDeposit*params.BeaconConfig().Gwei) {
		t.Error("Failed to set validator balance")
	}
	if validators[3].Status != pb.ValidatorRecord_ACTIVE_PENDING_EXIT {
		t.Errorf("Wanted status PendingWithdraw. Got: %d", validators[3].Status)
	}
	if validators[3].LatestStatusChangeSlot != params.BeaconConfig().MinWithdrawalPeriod+1 {
		t.Errorf("Failed to set validator lastest status change slot")
	}
	// Reach max validation rotation case, this validator Couldn't be rotated.
	if validators[5].Status != pb.ValidatorRecord_ACTIVE_PENDING_EXIT {
		t.Errorf("Wanted status PendingExit. Got: %d", validators[5].Status)
	}
	if validators[7].Status != pb.ValidatorRecord_ACTIVE_PENDING_EXIT {
		t.Errorf("Wanted status Withdrawn. Got: %d", validators[7].Status)
	}
	if validators[8].Status != pb.ValidatorRecord_EXITED_WITHOUT_PENALTY {
		t.Errorf("Wanted status Withdrawn. Got: %d", validators[8].Status)
	}
}

func TestValidatorMinDeposit(t *testing.T) {
	minDeposit := params.BeaconConfig().MinOnlineDepositSize * params.BeaconConfig().Gwei
	currentSlot := uint64(99)
	validators := []*pb.ValidatorRecord{
		{Status: pb.ValidatorRecord_ACTIVE, Balance: uint64(minDeposit) + 1},
		{Status: pb.ValidatorRecord_ACTIVE, Balance: uint64(minDeposit)},
		{Status: pb.ValidatorRecord_ACTIVE, Balance: uint64(minDeposit) - 1},
	}
	newValidatorRegistry := CheckValidatorMinDeposit(validators, currentSlot)
	if newValidatorRegistry[0].Status != pb.ValidatorRecord_ACTIVE {
		t.Error("Validator should be active")
	}
	if newValidatorRegistry[1].Status != pb.ValidatorRecord_ACTIVE {
		t.Error("Validator should be active")
	}
	if newValidatorRegistry[2].Status != pb.ValidatorRecord_ACTIVE_PENDING_EXIT {
		t.Error("Validator should be pending exit")
	}
	if newValidatorRegistry[2].LatestStatusChangeSlot != currentSlot {
		t.Errorf("Validator's lastest status change slot should be %d got %d", currentSlot, newValidatorRegistry[2].LatestStatusChangeSlot)
	}
}

func TestMinEmptyExitedValidator(t *testing.T) {
	validators := []*pb.ValidatorRecord{
		{Status: pb.ValidatorRecord_ACTIVE},
		{Status: pb.ValidatorRecord_EXITED_WITHOUT_PENALTY},
		{Status: pb.ValidatorRecord_ACTIVE},
	}
	if minEmptyExitedValidator(validators) != 1 {
		t.Errorf("Min vaidator index should be 1")
	}

	validators[1].Status = pb.ValidatorRecord_ACTIVE
	if minEmptyExitedValidator(validators) != -1 {
		t.Errorf("Min vaidator index should be -1")
	}
}

func TestDeepCopyValidatorRegistry(t *testing.T) {
	var validators []*pb.ValidatorRecord
	defaultValidator := &pb.ValidatorRecord{
		Pubkey:                 []byte{'k', 'e', 'y'},
		RandaoCommitmentHash32: []byte{'r', 'a', 'n', 'd', 'a', 'o'},
		Balance:                uint64(1e9),
		Status:                 pb.ValidatorRecord_ACTIVE,
		LatestStatusChangeSlot: 10,
	}
	for i := 0; i < 100; i++ {
		validators = append(validators, defaultValidator)
	}

	newValidatorSet := CopyValidatorRegistry(validators)

	defaultValidator.Pubkey = []byte{'n', 'e', 'w', 'k', 'e', 'y'}
	defaultValidator.RandaoCommitmentHash32 = []byte{'n', 'e', 'w', 'r', 'a', 'n', 'd', 'a', 'o'}
	defaultValidator.Balance = uint64(2e9)
	defaultValidator.Status = pb.ValidatorRecord_ACTIVE_PENDING_EXIT
	defaultValidator.LatestStatusChangeSlot = 5

	if len(newValidatorSet) != len(validators) {
		t.Fatalf("validator set length is unequal, copy of set failed: %d", len(newValidatorSet))
	}

	for i, validator := range newValidatorSet {
		if bytes.Equal(validator.Pubkey, defaultValidator.Pubkey) {
			t.Errorf("validator with index %d was unable to have their pubkey copied correctly %v", i, validator.Pubkey)
		}

		if bytes.Equal(validator.RandaoCommitmentHash32, defaultValidator.RandaoCommitmentHash32) {
			t.Errorf("validator with index %d was unable to have their randao commitment copied correctly %v", i, validator.RandaoCommitmentHash32)
		}

		if validator.Balance == defaultValidator.Balance {
			t.Errorf("validator with index %d was unable to have their balance copied correctly %d", i, validator.Balance)
		}

		if validator.Status == defaultValidator.Status {
			t.Errorf("validator with index %d was unable to have their status copied correctly %d", i, validator.Status)
		}

		if validator.LatestStatusChangeSlot == defaultValidator.LatestStatusChangeSlot {
			t.Errorf("validator with index %d was unable to have their lastest status change slot copied correctly %d", i, validator.LatestStatusChangeSlot)
		}
	}

}

func TestShardAndCommitteesAtSlot_OK(t *testing.T) {
	if params.BeaconConfig().EpochLength != 64 {
		t.Errorf("EpochLength should be 64 for these tests to pass")
	}

	var shardAndCommittees []*pb.ShardAndCommitteeArray
	for i := uint64(0); i < params.BeaconConfig().EpochLength*2; i++ {
		shardAndCommittees = append(shardAndCommittees, &pb.ShardAndCommitteeArray{
			ArrayShardAndCommittee: []*pb.ShardAndCommittee{
				{Shard: i},
			},
		})
	}

	state := &pb.BeaconState{
		ShardAndCommitteesAtSlots: shardAndCommittees,
	}

	tests := []struct {
		slot          uint64
		stateSlot     uint64
		expectedShard uint64
	}{
		{
			slot:          0,
			stateSlot:     0,
			expectedShard: 0,
		},
		{
			slot:          1,
			stateSlot:     5,
			expectedShard: 1,
		},
		{
			stateSlot:     1024,
			slot:          1024,
			expectedShard: 64 - 0,
		}, {
			stateSlot:     2048,
			slot:          2000,
			expectedShard: 64 - 48,
		}, {
			stateSlot:     2048,
			slot:          2058,
			expectedShard: 64 + 10,
		},
	}

	for _, tt := range tests {
		state.Slot = tt.stateSlot

		result, err := ShardAndCommitteesAtSlot(state, tt.slot)
		if err != nil {
			t.Errorf("Failed to get shard and committees at slot: %v", err)
		}

		if result.ArrayShardAndCommittee[0].Shard != tt.expectedShard {
			t.Errorf(
				"Result shard was an unexpected value. Wanted %d, got %d",
				tt.expectedShard,
				result.ArrayShardAndCommittee[0].Shard,
			)
		}
	}
}

func TestShardAndCommitteesAtSlot_OutOfBounds(t *testing.T) {
	if params.BeaconConfig().EpochLength != 64 {
		t.Errorf("EpochLength should be 64 for these tests to pass")
	}

	state := &pb.BeaconState{
		Slot: params.BeaconConfig().EpochLength,
	}

	tests := []struct {
		expectedErr string
		slot        uint64
	}{
		{
			expectedErr: "slot 5000 out of bounds: 0 <= slot < 128",
			slot:        5000,
		},
		{
			expectedErr: "slot 129 out of bounds: 0 <= slot < 128",
			slot:        129,
		},
	}

	for _, tt := range tests {
		_, err := ShardAndCommitteesAtSlot(state, tt.slot)
		if err != nil && err.Error() != tt.expectedErr {
			t.Fatalf("Expected error \"%s\" got \"%v\"", tt.expectedErr, err)
		}

	}
}

func TestEffectiveBalance(t *testing.T) {
	defaultBalance := params.BeaconConfig().MaxDeposit * params.BeaconConfig().Gwei

	tests := []struct {
		a uint64
		b uint64
	}{
		{a: 0, b: 0},
		{a: defaultBalance - 1, b: defaultBalance - 1},
		{a: defaultBalance, b: defaultBalance},
		{a: defaultBalance + 1, b: defaultBalance},
		{a: defaultBalance * 100, b: defaultBalance},
	}
	for _, test := range tests {
		state := &pb.BeaconState{ValidatorBalances: []uint64{test.a}}
		if EffectiveBalance(state, 0) != test.b {
			t.Errorf("EffectiveBalance(%d) = %d, want = %d", test.a, EffectiveBalance(state, 0), test.b)
		}
	}
}

func TestTotalEffectiveBalance(t *testing.T) {
	state := &pb.BeaconState{ValidatorBalances: []uint64{
		27 * 1e9, 28 * 1e9, 32 * 1e9, 40 * 1e9,
	}}

	// 27 + 28 + 32 + 32 = 119
	if TotalEffectiveBalance(state, []uint32{0, 1, 2, 3}) != 119*1e9 {
		t.Errorf("Incorrect TotalEffectiveBalance. Wanted: 119, got: %d",
			TotalEffectiveBalance(state, []uint32{0, 1, 2, 3})/1e9)
	}
}

func TestIsActiveValidator(t *testing.T) {

	tests := []struct {
		a pb.ValidatorRecord_StatusCodes
		b bool
	}{
		{a: pb.ValidatorRecord_PENDING_ACTIVATION, b: false},
		{a: pb.ValidatorRecord_ACTIVE, b: true},
		{a: pb.ValidatorRecord_ACTIVE_PENDING_EXIT, b: true},
		{a: pb.ValidatorRecord_EXITED_WITHOUT_PENALTY + 1, b: false},
		{a: pb.ValidatorRecord_EXITED_WITH_PENALTY * 100, b: false},
	}
	for _, test := range tests {
		validator := &pb.ValidatorRecord{Status: test.a}
		if isActiveValidator(validator) != test.b {
			t.Errorf("isActiveValidator(%d) = %v, want = %v", validator.Status, isActiveValidator(validator), test.b)
		}
	}
}

func TestGetActiveValidatorRecord(t *testing.T) {
	inputValidators := []*pb.ValidatorRecord{
		{ExitCount: 0},
		{ExitCount: 1},
		{ExitCount: 2},
		{ExitCount: 3},
		{ExitCount: 4},
	}

	outputValidators := []*pb.ValidatorRecord{
		{ExitCount: 1},
		{ExitCount: 3},
	}

	state := &pb.BeaconState{
		ValidatorRegistry: inputValidators,
	}

	validators := ActiveValidator(state, []uint32{1, 3})

	if !reflect.DeepEqual(outputValidators, validators) {
		t.Errorf("Active validators don't match. Wanted: %v, Got: %v", outputValidators, validators)
	}
}

func TestBoundaryAttestingBalance(t *testing.T) {
	state := &pb.BeaconState{ValidatorBalances: []uint64{
		25 * 1e9, 26 * 1e9, 32 * 1e9, 33 * 1e9, 100 * 1e9,
	}}

	attestedBalances := AttestingBalance(state, []uint32{0, 1, 2, 3, 4})

	// 25 + 26 + 32 + 32 + 32 = 147
	if attestedBalances != 147*1e9 {
		t.Errorf("Incorrect attested balances. Wanted: %f, got: %d", 147*1e9, attestedBalances)
	}
}

func TestBoundaryAttesters(t *testing.T) {
	var validators []*pb.ValidatorRecord

	for i := 0; i < 100; i++ {
		validators = append(validators, &pb.ValidatorRecord{Pubkey: []byte{byte(i)}})
	}

	state := &pb.BeaconState{ValidatorRegistry: validators}

	boundaryAttesters := Attesters(state, []uint32{5, 2, 87, 42, 99, 0})

	expectedBoundaryAttesters := []*pb.ValidatorRecord{
		{Pubkey: []byte{byte(5)}},
		{Pubkey: []byte{byte(2)}},
		{Pubkey: []byte{byte(87)}},
		{Pubkey: []byte{byte(42)}},
		{Pubkey: []byte{byte(99)}},
		{Pubkey: []byte{byte(0)}},
	}

	if !reflect.DeepEqual(expectedBoundaryAttesters, boundaryAttesters) {
		t.Errorf("Incorrect boundary attesters. Wanted: %v, got: %v", expectedBoundaryAttesters, boundaryAttesters)
	}
}

func TestBoundaryAttesterIndices(t *testing.T) {
	if params.BeaconConfig().EpochLength != 64 {
		t.Errorf("EpochLength should be 64 for these tests to pass")
	}
	var committeeIndices []uint32
	for i := uint32(0); i < 8; i++ {
		committeeIndices = append(committeeIndices, i)
	}
	var shardAndCommittees []*pb.ShardAndCommitteeArray
	for i := uint64(0); i < params.BeaconConfig().EpochLength*2; i++ {
		shardAndCommittees = append(shardAndCommittees, &pb.ShardAndCommitteeArray{
			ArrayShardAndCommittee: []*pb.ShardAndCommittee{
				{Shard: 100, Committee: committeeIndices},
			},
		})
	}

	state := &pb.BeaconState{
		ShardAndCommitteesAtSlots: shardAndCommittees,
		Slot:                      5,
	}

	boundaryAttestations := []*pb.PendingAttestationRecord{
		{Data: &pb.AttestationData{Slot: 2, Shard: 100}, ParticipationBitfield: []byte{'F'}}, // returns indices 1,5,6
		{Data: &pb.AttestationData{Slot: 2, Shard: 100}, ParticipationBitfield: []byte{3}},   // returns indices 6,7
		{Data: &pb.AttestationData{Slot: 2, Shard: 100}, ParticipationBitfield: []byte{'A'}}, // returns indices 1,7
	}

	attesterIndices, err := ValidatorIndices(state, boundaryAttestations)
	if err != nil {
		t.Fatalf("Failed to run BoundaryAttesterIndices: %v", err)
	}

	if !reflect.DeepEqual(attesterIndices, []uint32{1, 5, 6, 7}) {
		t.Errorf("Incorrect boundary attester indices. Wanted: %v, got: %v", []uint32{1, 5, 6, 7}, attesterIndices)
	}
}

func TestBeaconProposerIndex(t *testing.T) {
	if params.BeaconConfig().EpochLength != 64 {
		t.Errorf("EpochLength should be 64 for these tests to pass")
	}

	var shardAndCommittees []*pb.ShardAndCommitteeArray
	for i := uint64(0); i < params.BeaconConfig().EpochLength*2; i++ {
		shardAndCommittees = append(shardAndCommittees, &pb.ShardAndCommitteeArray{
			ArrayShardAndCommittee: []*pb.ShardAndCommittee{
				{Committee: []uint32{9, 8, 311, 12, 92, 1, 23, 17}},
			},
		})
	}

	state := &pb.BeaconState{
		ShardAndCommitteesAtSlots: shardAndCommittees,
	}

	tests := []struct {
		slot  uint64
		index uint32
	}{
		{
			slot:  1,
			index: 8,
		},
		{
			slot:  10,
			index: 311,
		},
		{
			slot:  19,
			index: 12,
		},
		{
			slot:  30,
			index: 23,
		},
		{
			slot:  39,
			index: 17,
		},
	}

	for _, tt := range tests {
		result, err := BeaconProposerIndex(state, tt.slot)
		if err != nil {
			t.Errorf("Failed to get shard and committees at slot: %v", err)
		}

		if result != tt.index {
			t.Errorf(
				"Result index was an unexpected value. Wanted %d, got %d",
				tt.index,
				result,
			)
		}
	}
}

func TestAttestingValidatorIndices_Ok(t *testing.T) {
	if params.BeaconConfig().EpochLength != 64 {
		t.Errorf("EpochLength should be 64 for these tests to pass")
	}

	var committeeIndices []uint32
	for i := uint32(0); i < 8; i++ {
		committeeIndices = append(committeeIndices, i)
	}

	var shardAndCommittees []*pb.ShardAndCommitteeArray
	for i := uint64(0); i < params.BeaconConfig().EpochLength*2; i++ {
		shardAndCommittees = append(shardAndCommittees, &pb.ShardAndCommitteeArray{
			ArrayShardAndCommittee: []*pb.ShardAndCommittee{
				{Shard: i, Committee: committeeIndices},
			},
		})
	}

	state := &pb.BeaconState{
		ShardAndCommitteesAtSlots: shardAndCommittees,
		Slot:                      5,
	}

	prevAttestation := &pb.PendingAttestationRecord{
		Data: &pb.AttestationData{
			Slot:                 3,
			Shard:                3,
			ShardBlockRootHash32: []byte{'B'},
		},
		ParticipationBitfield: []byte{'A'}, // 01000001 = 1,7
	}

	thisAttestation := &pb.PendingAttestationRecord{
		Data: &pb.AttestationData{
			Slot:                 3,
			Shard:                3,
			ShardBlockRootHash32: []byte{'B'},
		},
		ParticipationBitfield: []byte{'F'}, // 01000110 = 1,5,6
	}

	indices, err := AttestingValidatorIndices(
		state,
		shardAndCommittees[3].ArrayShardAndCommittee[0],
		[]byte{'B'},
		[]*pb.PendingAttestationRecord{thisAttestation},
		[]*pb.PendingAttestationRecord{prevAttestation})
	if err != nil {
		t.Fatalf("Could not execute AttestingValidatorIndices: %v", err)
	}

	// Union(1,7,1,5,6) = 1,5,6,7
	if !reflect.DeepEqual(indices, []uint32{1, 5, 6, 7}) {
		t.Errorf("Could not get incorrect validator indices. Wanted: %v, got: %v",
			[]uint32{1, 5, 6, 7}, indices)
	}
}

func TestAttestingValidatorIndices_OutOfBound(t *testing.T) {
	shardAndCommittees := []*pb.ShardAndCommitteeArray{
		{ArrayShardAndCommittee: []*pb.ShardAndCommittee{
			{Shard: 1},
		}},
	}

	state := &pb.BeaconState{
		ShardAndCommitteesAtSlots: shardAndCommittees,
		Slot:                      5,
	}

	attestation := &pb.PendingAttestationRecord{
		Data: &pb.AttestationData{
			Slot:                 0,
			Shard:                1,
			ShardBlockRootHash32: []byte{'B'},
		},
		ParticipationBitfield: []byte{'A'}, // 01000001 = 1,7
	}

	_, err := AttestingValidatorIndices(
		state,
		shardAndCommittees[0].ArrayShardAndCommittee[0],
		[]byte{'B'},
		[]*pb.PendingAttestationRecord{attestation},
		nil)

	// This will fail because participation bitfield is length:1, committee bitfield is length 0.
	if err == nil {
		t.Fatal("AttestingValidatorIndices should have failed with incorrect bitfield")
	}
}

func TestAllValidatorIndices(t *testing.T) {
	tests := []struct {
		registries []*pb.ValidatorRecord
		indices    []uint32
	}{
		{registries: []*pb.ValidatorRecord{}, indices: []uint32{}},
		{registries: []*pb.ValidatorRecord{{}}, indices: []uint32{0}},
		{registries: []*pb.ValidatorRecord{{}, {}, {}, {}}, indices: []uint32{0, 1, 2, 3}},
	}
	for _, tt := range tests {
		state := &pb.BeaconState{ValidatorRegistry: tt.registries}
		if !reflect.DeepEqual(AllValidatorsIndices(state), tt.indices) {
			t.Errorf("AllValidatorsIndices(%v) = %v, wanted:%v",
				tt.registries, AllValidatorsIndices(state), tt.indices)
		}
	}
}

func TestAllActiveValidatorIndices(t *testing.T) {
	tests := []struct {
		registries []*pb.ValidatorRecord
		indices    []uint32
	}{
		{registries: []*pb.ValidatorRecord{
			{Status: pb.ValidatorRecord_ACTIVE},
			{Status: pb.ValidatorRecord_EXITED_WITH_PENALTY},
			{Status: pb.ValidatorRecord_PENDING_ACTIVATION},
			{Status: pb.ValidatorRecord_EXITED_WITHOUT_PENALTY}},
			indices: []uint32{0}},
		{registries: []*pb.ValidatorRecord{
			{Status: pb.ValidatorRecord_ACTIVE},
			{Status: pb.ValidatorRecord_ACTIVE},
			{Status: pb.ValidatorRecord_ACTIVE},
			{Status: pb.ValidatorRecord_ACTIVE}},
			indices: []uint32{0, 1, 2, 3}},
	}
	for _, tt := range tests {
		state := &pb.BeaconState{ValidatorRegistry: tt.registries}
		if !reflect.DeepEqual(AllActiveValidatorsIndices(state), tt.indices) {
			t.Errorf("AllActiveValidatorsIndices(%v) = %v, wanted:%v",
				tt.registries, AllActiveValidatorsIndices(state), tt.indices)
		}
	}
}

func TestNewRegistryDeltaChainTip(t *testing.T) {
	tests := []struct {
		flag                         uint64
		index                        uint32
		pubKey                       []byte
		currentRegistryDeltaChainTip []byte
		newRegistryDeltaChainTip     []byte
	}{
		{0, 100, []byte{'A'}, []byte{'B'},
			[]byte{35, 123, 149, 41, 92, 226, 26, 73, 96, 40, 4, 219, 59, 254, 27,
				38, 220, 125, 83, 177, 78, 12, 187, 74, 72, 115, 64, 91, 16, 144, 37, 245}},
		{2, 64, []byte{'Y'}, []byte{'Z'},
			[]byte{105, 155, 218, 237, 2, 246, 129, 117, 122, 234, 129, 145, 140,
				42, 123, 133, 57, 241, 58, 237, 43, 180, 158, 123, 236, 47, 141, 21, 71, 150, 237, 246}},
	}
	for _, tt := range tests {
		newChainTip, err := NewRegistryDeltaChainTip(
			pb.ValidatorRegistryDeltaBlock_ValidatorRegistryDeltaFlags(tt.flag),
			tt.index,
			tt.pubKey,
			tt.currentRegistryDeltaChainTip,
		)
		if err != nil {
			t.Fatalf("Could not execute NewRegistryDeltaChainTip:%v", err)
		}
		if !bytes.Equal(newChainTip[:], tt.newRegistryDeltaChainTip) {
			t.Errorf("Incorrect new chain tip. Wanted %#x, got %#x",
				tt.newRegistryDeltaChainTip, newChainTip[:])
		}
	}
}

func TestProcessDeposit_PublicKeyExistsBadWithdrawalCredentials(t *testing.T) {
	registry := []*pb.ValidatorRecord{
		{
			Pubkey: []byte{1, 2, 3},
		},
		{
			Pubkey:                []byte{4, 5, 6},
			WithdrawalCredentials: []byte{0},
		},
	}
	beaconState := &pb.BeaconState{
		ValidatorRegistry: registry,
	}
	pubkey := []byte{4, 5, 6}
	deposit := uint64(1000)
	proofOfPossession := []byte{}
	withdrawalCredentials := []byte{1}
	randaoCommitment := []byte{}
	pocCommitment := []byte{}

	want := "expected withdrawal credentials to match"
	if _, _, err := ProcessDeposit(
		beaconState,
		pubkey,
		deposit,
		proofOfPossession,
		withdrawalCredentials,
		randaoCommitment,
		pocCommitment,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Wanted error to contain %s, received %v", want, err)
	}
}

func TestProcessDeposit_PublicKeyExistsGoodWithdrawalCredentials(t *testing.T) {
	registry := []*pb.ValidatorRecord{
		{
			Pubkey: []byte{1, 2, 3},
		},
		{
			Pubkey:                []byte{4, 5, 6},
			WithdrawalCredentials: []byte{1},
		},
	}
	balances := []uint64{0, 0}
	beaconState := &pb.BeaconState{
		ValidatorBalances: balances,
		ValidatorRegistry: registry,
	}
	pubkey := []byte{4, 5, 6}
	deposit := uint64(1000)
	proofOfPossession := []byte{}
	withdrawalCredentials := []byte{1}
	randaoCommitment := []byte{}
	pocCommitment := []byte{}

	newState, _, err := ProcessDeposit(
		beaconState,
		pubkey,
		deposit,
		proofOfPossession,
		withdrawalCredentials,
		randaoCommitment,
		pocCommitment,
	)
	if err != nil {
		t.Fatalf("Process deposit failed: %v", err)
	}
	if newState.ValidatorBalances[1] != 1000 {
		t.Errorf("Expected balance at index 1 to be 1000, received %d", newState.ValidatorBalances[1])
	}
}

func TestProcessDeposit_PublicKeyDoesNotExistNoEmptyValidator(t *testing.T) {
	registry := []*pb.ValidatorRecord{
		{
			Pubkey:                []byte{1, 2, 3},
			WithdrawalCredentials: []byte{2},
		},
		{
			Pubkey:                []byte{4, 5, 6},
			WithdrawalCredentials: []byte{1},
		},
	}
	balances := []uint64{1000, 1000}
	beaconState := &pb.BeaconState{
		ValidatorBalances: balances,
		ValidatorRegistry: registry,
	}
	pubkey := []byte{7, 8, 9}
	deposit := uint64(2000)
	proofOfPossession := []byte{}
	withdrawalCredentials := []byte{1}
	randaoCommitment := []byte{}
	pocCommitment := []byte{}

	newState, _, err := ProcessDeposit(
		beaconState,
		pubkey,
		deposit,
		proofOfPossession,
		withdrawalCredentials,
		randaoCommitment,
		pocCommitment,
	)
	if err != nil {
		t.Fatalf("Process deposit failed: %v", err)
	}
	if len(newState.ValidatorBalances) != 3 {
		t.Errorf("Expected validator balances list to increase by 1, received len %d", len(newState.ValidatorBalances))
	}
	if newState.ValidatorBalances[2] != 2000 {
		t.Errorf("Expected new validator have balance of %d, received %d", 2000, newState.ValidatorBalances[2])
	}
}

func TestProcessDeposit_PublicKeyDoesNotExistEmptyValidatorExists(t *testing.T) {
	registry := []*pb.ValidatorRecord{
		{
			Pubkey:                 []byte{1, 2, 3},
			WithdrawalCredentials:  []byte{2},
			LatestStatusChangeSlot: 0,
		},
		{
			Pubkey:                 []byte{4, 5, 6},
			WithdrawalCredentials:  []byte{1},
			LatestStatusChangeSlot: 0,
		},
	}
	balances := []uint64{0, 1000}
	beaconState := &pb.BeaconState{
		Slot:              0 + params.BeaconConfig().ZeroBalanceValidatorTTL,
		ValidatorBalances: balances,
		ValidatorRegistry: registry,
	}
	pubkey := []byte{7, 8, 9}
	deposit := uint64(2000)
	proofOfPossession := []byte{}
	withdrawalCredentials := []byte{1}
	randaoCommitment := []byte{}
	pocCommitment := []byte{}

	newState, _, err := ProcessDeposit(
		beaconState,
		pubkey,
		deposit,
		proofOfPossession,
		withdrawalCredentials,
		randaoCommitment,
		pocCommitment,
	)
	if err != nil {
		t.Fatalf("Process deposit failed: %v", err)
	}
	if len(newState.ValidatorBalances) != 2 {
		t.Errorf("Expected validator balances list to stay the same, received len %d", len(newState.ValidatorBalances))
	}
	if newState.ValidatorBalances[0] != 2000 {
		t.Errorf("Expected validator at index 0 to have balance of %d, received %d", 2000, newState.ValidatorBalances[0])
	}
}

func TestActivateValidator_Ok(t *testing.T) {
	state := &pb.BeaconState{
		Slot:                                 100,
		ValidatorRegistryDeltaChainTipHash32: []byte{'A'},
		ValidatorRegistry: []*pb.ValidatorRecord{
			{Status: pb.ValidatorRecord_PENDING_ACTIVATION, Pubkey: []byte{'B'}},
		},
	}
	newState, err := activateValidator(state, 0)
	if err != nil {
		t.Fatalf("Could not execute activateValidator:%v", err)
	}
	if newState.ValidatorRegistry[0].Status != pb.ValidatorRecord_ACTIVE {
		t.Errorf("Wanted status ACTIVE, got %v", newState.ValidatorRegistry[0].Status)
	}
	if newState.ValidatorRegistry[0].LatestStatusChangeSlot != state.Slot {
		t.Errorf("Wanted last change slot %d, got %v",
			state.Slot, newState.ValidatorRegistry[0].LatestStatusChangeSlot)
	}
}

func TestActivateValidator_BadStatus(t *testing.T) {
	state := &pb.BeaconState{
		ValidatorRegistry: []*pb.ValidatorRecord{
			{Status: pb.ValidatorRecord_ACTIVE},
		},
	}
	if _, err := activateValidator(state, 0); err == nil {
		t.Fatal("activateValidator should have failed with incorrect status")
	}
}

func TestInitiateValidatorExit_Ok(t *testing.T) {
	state := &pb.BeaconState{
		Slot: 200,
		ValidatorRegistry: []*pb.ValidatorRecord{
			{Status: pb.ValidatorRecord_ACTIVE},
		},
	}
	newState, err := initiateValidatorExit(state, 0)
	if err != nil {
		t.Fatalf("Could not execute initiateValidatorExit:%v", err)
	}
	if newState.ValidatorRegistry[0].Status != pb.ValidatorRecord_ACTIVE_PENDING_EXIT {
		t.Errorf("Wanted status ACTIVE_PENDING_EXIT, got %v", newState.ValidatorRegistry[0].Status)
	}
	if newState.ValidatorRegistry[0].LatestStatusChangeSlot != state.Slot {
		t.Errorf("Wanted last change slot %d, got %v",
			state.Slot, newState.ValidatorRegistry[0].LatestStatusChangeSlot)
	}
}

func TestInitiateValidatorExit_BadStatus(t *testing.T) {
	state := &pb.BeaconState{
		ValidatorRegistry: []*pb.ValidatorRecord{
			{Status: pb.ValidatorRecord_ACTIVE_PENDING_EXIT},
		},
	}
	if _, err := initiateValidatorExit(state, 0); err == nil {
		t.Fatal("initiateValidatorExit should have failed with incorrect status")
	}
}

func TestExitValidatorWithPenalty_Ok(t *testing.T) {
	var shardAndCommittees []*pb.ShardAndCommitteeArray
	for i := uint64(0); i < params.BeaconConfig().EpochLength*2; i++ {
		shardAndCommittees = append(shardAndCommittees, &pb.ShardAndCommitteeArray{
			ArrayShardAndCommittee: []*pb.ShardAndCommittee{
				{Committee: []uint32{0, 1, 2, 3, 4, 5, 6, 7}},
			},
		})
	}
	state := &pb.BeaconState{
		Slot:                      100,
		ShardAndCommitteesAtSlots: shardAndCommittees,
		ValidatorBalances: []uint64{params.BeaconConfig().MaxDepositInGwei, params.BeaconConfig().MaxDepositInGwei,
			params.BeaconConfig().MaxDepositInGwei, params.BeaconConfig().MaxDepositInGwei, params.BeaconConfig().MaxDepositInGwei},
		ValidatorRegistryDeltaChainTipHash32: []byte{'A'},
		LatestPenalizedExitBalances:          []uint64{0},
		ValidatorRegistry: []*pb.ValidatorRecord{
			{Status: pb.ValidatorRecord_ACTIVE, Pubkey: []byte{'B'}},
		},
		PersistentCommittees: []*common.Uint32List{
			{List: []uint32{1, 2, 0, 4, 6}},
		},
	}
	newStatus := pb.ValidatorRecord_EXITED_WITH_PENALTY
	newState, err := exitValidator(state, 0, newStatus)
	if err != nil {
		t.Fatalf("Could not execute exitValidator:%v", err)
	}

	if newState.ValidatorRegistry[0].Status != newStatus {
		t.Errorf("Wanted status %v, got %v", newStatus, newState.ValidatorRegistry[0].Status)
	}
	if newState.ValidatorRegistry[0].LatestStatusChangeSlot != state.Slot {
		t.Errorf("Wanted last change slot %d, got %v",
			state.Slot, newState.ValidatorRegistry[0].LatestStatusChangeSlot)
	}
	if newState.ValidatorRegistry[0].ExitCount != 1 {
		t.Errorf("Wanted exit count 1, got %d", newState.ValidatorRegistry[0].ExitCount)
	}
	if newState.ValidatorBalances[0] != 0 {
		t.Errorf("Wanted validator balance 0, got %d", newState.ValidatorBalances[0])
	}
	if newState.ValidatorBalances[4] != 2*params.BeaconConfig().MaxDepositInGwei {
		t.Errorf("Wanted validator balance %d, got %d",
			2*params.BeaconConfig().MaxDepositInGwei, newState.ValidatorBalances[4])
	}
	for _, i := range newState.PersistentCommittees[0].List {
		if i == 0 {
			t.Errorf("Validator index 0 should be removed from persistent committee. Got: %v",
				newState.PersistentCommittees[0].List)
		}
	}
}

func TestExitValidator_AlreadyExitedWithPenalty(t *testing.T) {
	state := &pb.BeaconState{
		ValidatorRegistry: []*pb.ValidatorRecord{
			{Status: pb.ValidatorRecord_EXITED_WITH_PENALTY},
		},
	}
	if _, err := exitValidator(state, 0, pb.ValidatorRecord_EXITED_WITH_PENALTY); err == nil {
		t.Fatal("exitValidator should have failed with incorrect status")
	}
}

func TestExitValidator_AlreadyExitedWithOutPenalty(t *testing.T) {
	state := &pb.BeaconState{
		ValidatorRegistry: []*pb.ValidatorRecord{
			{Status: pb.ValidatorRecord_EXITED_WITHOUT_PENALTY},
		},
	}
	if _, err := exitValidator(state, 0, pb.ValidatorRecord_EXITED_WITHOUT_PENALTY); err == nil {
		t.Fatal("exitValidator should have failed with incorrect status")
	}
}

func TestUpdateValidatorStatus_Ok(t *testing.T) {
	var shardAndCommittees []*pb.ShardAndCommitteeArray
	for i := uint64(0); i < params.BeaconConfig().EpochLength*2; i++ {
		shardAndCommittees = append(shardAndCommittees, &pb.ShardAndCommitteeArray{
			ArrayShardAndCommittee: []*pb.ShardAndCommittee{
				{Committee: []uint32{0, 1, 2, 3, 4, 5, 6, 7}},
			},
		})
	}
	state := &pb.BeaconState{
		ShardAndCommitteesAtSlots:   shardAndCommittees,
		ValidatorBalances:           []uint64{params.BeaconConfig().MaxDepositInGwei},
		LatestPenalizedExitBalances: []uint64{0},
		ValidatorRegistry:           []*pb.ValidatorRecord{{}},
	}
	tests := []struct {
		currentStatus pb.ValidatorRecord_StatusCodes
		newStatus     pb.ValidatorRecord_StatusCodes
	}{
		{pb.ValidatorRecord_PENDING_ACTIVATION, pb.ValidatorRecord_ACTIVE},
		{pb.ValidatorRecord_ACTIVE, pb.ValidatorRecord_ACTIVE_PENDING_EXIT},
		{pb.ValidatorRecord_ACTIVE, pb.ValidatorRecord_EXITED_WITH_PENALTY},
		{pb.ValidatorRecord_ACTIVE, pb.ValidatorRecord_EXITED_WITHOUT_PENALTY},
	}
	for _, tt := range tests {
		state.ValidatorRegistry[0].Status = tt.currentStatus
		newState, err := UpdateStatus(state, 0, tt.newStatus)
		if err != nil {
			t.Fatalf("Could not execute UpdateStatus: %v", err)
		}
		if newState.ValidatorRegistry[0].Status != tt.newStatus {
			t.Errorf("Expected status:%v, got:%v",
				tt.newStatus, newState.ValidatorRegistry[0].Status)
		}
	}
}

func TestUpdateValidatorStatus_IncorrectStatus(t *testing.T) {
	if _, err := UpdateStatus(
		&pb.BeaconState{}, 0, pb.ValidatorRecord_PENDING_ACTIVATION); err == nil {
		t.Fatal("UpdateStatus should have failed with incorrect status")
	}
}
