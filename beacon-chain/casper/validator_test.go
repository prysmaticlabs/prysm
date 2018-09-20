package casper

import (
	"math"
	"reflect"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/params"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared"
)

func TestRotateValidatorSet(t *testing.T) {
	validators := []*pb.ValidatorRecord{
		{Balance: 10, StartDynasty: 0, EndDynasty: params.DefaultEndDynasty},  // half below default balance, should be moved to exit.
		{Balance: 15, StartDynasty: 1, EndDynasty: params.DefaultEndDynasty},  // half below default balance, should be moved to exit.
		{Balance: 20, StartDynasty: 2, EndDynasty: params.DefaultEndDynasty},  // stays in active.
		{Balance: 25, StartDynasty: 3, EndDynasty: params.DefaultEndDynasty},  // stays in active.
		{Balance: 30, StartDynasty: 4, EndDynasty: params.DefaultEndDynasty},  // stays in active.
		{Balance: 30, StartDynasty: 15, EndDynasty: params.DefaultEndDynasty}, // will trigger for loop in rotate val indices.
	}

	data := &pb.CrystallizedState{
		Validators:     validators,
		CurrentDynasty: 10,
	}

	// Rotate validator set and increment dynasty count by 1.
	rotatedValidators := RotateValidatorSet(data.Validators, data.CurrentDynasty)
	if !reflect.DeepEqual(ActiveValidatorIndices(rotatedValidators, data.CurrentDynasty), []uint32{2, 3, 4, 5}) {
		t.Errorf("active validator indices should be [2,3,4,5], got: %v", ActiveValidatorIndices(rotatedValidators, data.CurrentDynasty))
	}
	if len(QueuedValidatorIndices(rotatedValidators, data.CurrentDynasty)) != 0 {
		t.Errorf("queued validator indices should be [], got: %v", QueuedValidatorIndices(rotatedValidators, data.CurrentDynasty))
	}
	if !reflect.DeepEqual(ExitedValidatorIndices(rotatedValidators, data.CurrentDynasty), []uint32{0, 1}) {
		t.Errorf("exited validator indices should be [0,1], got: %v", ExitedValidatorIndices(rotatedValidators, data.CurrentDynasty))
	}

	// Another run without queuing validators.
	validators = []*pb.ValidatorRecord{
		{Balance: 10, StartDynasty: 0, EndDynasty: params.DefaultEndDynasty}, // half below default balance, should be moved to exit.
		{Balance: 15, StartDynasty: 1, EndDynasty: params.DefaultEndDynasty}, // half below default balance, should be moved to exit.
		{Balance: 20, StartDynasty: 2, EndDynasty: params.DefaultEndDynasty}, // stays in active.
		{Balance: 25, StartDynasty: 3, EndDynasty: params.DefaultEndDynasty}, // stays in active.
		{Balance: 30, StartDynasty: 4, EndDynasty: params.DefaultEndDynasty}, // stays in active.
	}

	data = &pb.CrystallizedState{
		Validators:     validators,
		CurrentDynasty: 10,
	}

	// rotate validator set and increment dynasty count by 1.
	RotateValidatorSet(data.Validators, data.CurrentDynasty)

	if !reflect.DeepEqual(ActiveValidatorIndices(data.Validators, data.CurrentDynasty), []uint32{2, 3, 4}) {
		t.Errorf("active validator indices should be [2,3,4], got: %v", ActiveValidatorIndices(data.Validators, data.CurrentDynasty))
	}
	if len(QueuedValidatorIndices(data.Validators, data.CurrentDynasty)) != 0 {
		t.Errorf("queued validator indices should be [], got: %v", QueuedValidatorIndices(data.Validators, data.CurrentDynasty))
	}
	if !reflect.DeepEqual(ExitedValidatorIndices(data.Validators, data.CurrentDynasty), []uint32{0, 1}) {
		t.Errorf("exited validator indices should be [0,1], got: %v", ExitedValidatorIndices(data.Validators, data.CurrentDynasty))
	}
}

func TestHasVoted(t *testing.T) {
	// Setting bit field to 11111111.
	pendingAttestation := &pb.AggregatedAttestation{
		AttesterBitfield: []byte{255},
	}

	for i := 0; i < len(pendingAttestation.AttesterBitfield); i++ {
		voted := shared.CheckBit(pendingAttestation.AttesterBitfield, i)
		if !voted {
			t.Error("validator voted but received didn't vote")
		}
	}

	// Setting bit field to 01010101.
	pendingAttestation = &pb.AggregatedAttestation{
		AttesterBitfield: []byte{85},
	}

	for i := 0; i < len(pendingAttestation.AttesterBitfield); i++ {
		voted := shared.CheckBit(pendingAttestation.AttesterBitfield, i)
		if i%2 == 0 && voted {
			t.Error("validator didn't vote but received voted")
		}
		if i%2 == 1 && !voted {
			t.Error("validator voted but received didn't vote")
		}
	}
}

func TestValidatorIndices(t *testing.T) {
	data := &pb.CrystallizedState{
		Validators: []*pb.ValidatorRecord{
			{PublicKey: 0, StartDynasty: 0, EndDynasty: 2},                   // active.
			{PublicKey: 0, StartDynasty: 0, EndDynasty: 2},                   // active.
			{PublicKey: 0, StartDynasty: 1, EndDynasty: 2},                   // active.
			{PublicKey: 0, StartDynasty: 0, EndDynasty: 2},                   // active.
			{PublicKey: 0, StartDynasty: 0, EndDynasty: 3},                   // active.
			{PublicKey: 0, StartDynasty: 2, EndDynasty: uint64(math.Inf(0))}, // queued.
		},
		CurrentDynasty: 1,
	}

	if !reflect.DeepEqual(ActiveValidatorIndices(data.Validators, data.CurrentDynasty), []uint32{0, 1, 2, 3, 4}) {
		t.Errorf("active validator indices should be [0 1 2 3 4], got: %v", ActiveValidatorIndices(data.Validators, data.CurrentDynasty))
	}
	if !reflect.DeepEqual(QueuedValidatorIndices(data.Validators, data.CurrentDynasty), []uint32{5}) {
		t.Errorf("queued validator indices should be [5], got: %v", QueuedValidatorIndices(data.Validators, data.CurrentDynasty))
	}
	if len(ExitedValidatorIndices(data.Validators, data.CurrentDynasty)) != 0 {
		t.Errorf("exited validator indices to be empty, got: %v", ExitedValidatorIndices(data.Validators, data.CurrentDynasty))
	}

	data = &pb.CrystallizedState{
		Validators: []*pb.ValidatorRecord{
			{PublicKey: 0, StartDynasty: 1, EndDynasty: uint64(math.Inf(0))}, // active.
			{PublicKey: 0, StartDynasty: 2, EndDynasty: uint64(math.Inf(0))}, // active.
			{PublicKey: 0, StartDynasty: 6, EndDynasty: uint64(math.Inf(0))}, // queued.
			{PublicKey: 0, StartDynasty: 7, EndDynasty: uint64(math.Inf(0))}, // queued.
			{PublicKey: 0, StartDynasty: 1, EndDynasty: 2},                   // exited.
			{PublicKey: 0, StartDynasty: 1, EndDynasty: 3},                   // exited.
		},
		CurrentDynasty: 5,
	}

	if !reflect.DeepEqual(ActiveValidatorIndices(data.Validators, data.CurrentDynasty), []uint32{0, 1}) {
		t.Errorf("active validator indices should be [0 1 2 4 5], got: %v", ActiveValidatorIndices(data.Validators, data.CurrentDynasty))
	}
	if !reflect.DeepEqual(QueuedValidatorIndices(data.Validators, data.CurrentDynasty), []uint32{2, 3}) {
		t.Errorf("queued validator indices should be [3], got: %v", QueuedValidatorIndices(data.Validators, data.CurrentDynasty))
	}
	if !reflect.DeepEqual(ExitedValidatorIndices(data.Validators, data.CurrentDynasty), []uint32{4, 5}) {
		t.Errorf("exited validator indices should be [3], got: %v", ExitedValidatorIndices(data.Validators, data.CurrentDynasty))
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
			{ShardId: 0, Committee: []uint32{0, 1, 2, 3, 4}},
			{ShardId: 1, Committee: []uint32{5, 6, 7, 8, 9}},
		}},
		{ArrayShardAndCommittee: []*pb.ShardAndCommittee{
			{ShardId: 2, Committee: []uint32{10, 11, 12, 13, 14}},
			{ShardId: 3, Committee: []uint32{15, 16, 17, 18, 19}},
		}},
		{ArrayShardAndCommittee: []*pb.ShardAndCommittee{
			{ShardId: 4, Committee: []uint32{20, 21, 22, 23, 24}},
			{ShardId: 5, Committee: []uint32{25, 26, 27, 28, 29}},
		}},
	}
	if _, _, err := ProposerShardAndIndex(shardCommittees, 100, 0); err == nil {
		t.Error("ProposerShardAndIndex should have failed with invalid lcs")
	}
	shardID, index, err := ProposerShardAndIndex(shardCommittees, 128, 64)
	if err != nil {
		t.Fatalf("ProposerShardAndIndex failed with %v", err)
	}
	if shardID != 0 {
		t.Errorf("Invalid shard ID. Wanted 0, got %d", shardID)
	}
	if index != 4 {
		t.Errorf("Invalid proposer index. Wanted 4, got %d", index)
	}
}

func TestValidatorIndex(t *testing.T) {
	var validators []*pb.ValidatorRecord
	for i := 0; i < 10; i++ {
		validators = append(validators, &pb.ValidatorRecord{StartDynasty: 0, EndDynasty: 10, PublicKey: 0})
	}
	if _, err := ValidatorIndex(100, 0, validators); err == nil {
		t.Fatalf("ValidatorIndex should have failed,  there's no validator with pubkey 100")
	}
	validators[5].PublicKey = 100
	index, err := ValidatorIndex(100, 0, validators)
	if err != nil {
		t.Fatalf("call ValidatorIndex failed: %v", err)
	}
	if index != 5 {
		t.Errorf("Incorrect validator index. Wanted 5, Got %v", index)
	}
}

func TestValidatorShardID(t *testing.T) {
	var validators []*pb.ValidatorRecord
	for i := 0; i < 21; i++ {
		validators = append(validators, &pb.ValidatorRecord{StartDynasty: 0, EndDynasty: 10, PublicKey: 0})
	}
	shardCommittees := []*pb.ShardAndCommitteeArray{
		{ArrayShardAndCommittee: []*pb.ShardAndCommittee{
			{ShardId: 0, Committee: []uint32{0, 1, 2, 3, 4, 5, 6}},
			{ShardId: 1, Committee: []uint32{7, 8, 9, 10, 11, 12, 13}},
			{ShardId: 2, Committee: []uint32{14, 15, 16, 17, 18, 19}},
		}},
	}
	validators[19].PublicKey = 100
	shardID, err := ValidatorShardID(100, 0, validators, shardCommittees)
	if err != nil {
		t.Fatalf("call ValidatorShardID failed: %v", err)
	}
	if shardID != 2 {
		t.Errorf("Incorrect validator shard ID. Wanted 2, Got %v", shardID)
	}

	validators[19].PublicKey = 0
	if _, err := ValidatorShardID(100, 0, validators, shardCommittees); err == nil {
		t.Fatalf("ValidatorShardID should have failed, there's no validator with pubkey 100")
	}

	validators[20].PublicKey = 100
	if _, err := ValidatorShardID(100, 0, validators, shardCommittees); err == nil {
		t.Fatalf("ValidatorShardID should have failed, validator indexed at 20 is not in the committee")
	}
}

func TestValidatorSlot(t *testing.T) {
	var validators []*pb.ValidatorRecord
	for i := 0; i < 61; i++ {
		validators = append(validators, &pb.ValidatorRecord{StartDynasty: 0, EndDynasty: 10, PublicKey: 0})
	}
	shardCommittees := []*pb.ShardAndCommitteeArray{
		{ArrayShardAndCommittee: []*pb.ShardAndCommittee{
			{ShardId: 0, Committee: []uint32{0, 1, 2, 3, 4, 5, 6}},
			{ShardId: 1, Committee: []uint32{7, 8, 9, 10, 11, 12, 13}},
			{ShardId: 2, Committee: []uint32{14, 15, 16, 17, 18, 19}},
		}},
		{ArrayShardAndCommittee: []*pb.ShardAndCommittee{
			{ShardId: 3, Committee: []uint32{20, 21, 22, 23, 24, 25, 26}},
			{ShardId: 4, Committee: []uint32{27, 28, 29, 30, 31, 32, 33}},
			{ShardId: 5, Committee: []uint32{34, 35, 36, 37, 38, 39}},
		}},
		{ArrayShardAndCommittee: []*pb.ShardAndCommittee{
			{ShardId: 6, Committee: []uint32{40, 41, 42, 43, 44, 45, 46}},
			{ShardId: 7, Committee: []uint32{47, 48, 49, 50, 51, 52, 53}},
			{ShardId: 8, Committee: []uint32{54, 55, 56, 57, 58, 59}},
		}},
	}
	if _, err := ValidatorSlot(100, 0, validators, shardCommittees); err == nil {
		t.Fatalf("ValidatorSlot should have failed, there's no validator with pubkey 100")
	}

	validators[59].PublicKey = 100
	slot, err := ValidatorSlot(100, 0, validators, shardCommittees)
	if err != nil {
		t.Fatalf("call ValidatorSlot failed: %v", err)
	}
	if slot != 2 {
		t.Errorf("Incorrect validator slot ID. Wanted 1, Got %v", slot)
	}

	validators[60].PublicKey = 101
	if _, err := ValidatorSlot(101, 0, validators, shardCommittees); err == nil {
		t.Fatalf("ValidatorSlot should have failed, validator indexed at 60 is not in the committee")
	}
}

func TestTotalActiveValidatorDeposit(t *testing.T) {
	var validators []*pb.ValidatorRecord
	for i := 0; i < 10; i++ {
		validators = append(validators, &pb.ValidatorRecord{StartDynasty: 0, EndDynasty: 10, Balance: 1e18})
	}

	totalDeposit := TotalActiveValidatorDeposit(0, validators)
	if totalDeposit != 10e18 {
		t.Fatalf("incorrect total deposit calculated %d", totalDeposit)
	}

	totalDepositETH := TotalActiveValidatorDepositInEth(0, validators)
	if totalDepositETH != 10 {
		t.Fatalf("incorrect total deposit in ETH calculated %d", totalDepositETH)
	}
}
