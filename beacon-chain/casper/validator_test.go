package casper

import (
	"math"
	"reflect"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/params"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
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
	pendingAttestation := &pb.AttestationRecord{
		AttesterBitfield: []byte{255},
	}

	for i := 0; i < len(pendingAttestation.AttesterBitfield); i++ {
		voted := utils.CheckBit(pendingAttestation.AttesterBitfield, i)
		if !voted {
			t.Error("validator voted but received didn't vote")
		}
	}

	// Setting bit field to 01010101.
	pendingAttestation = &pb.AttestationRecord{
		AttesterBitfield: []byte{85},
	}

	for i := 0; i < len(pendingAttestation.AttesterBitfield); i++ {
		voted := utils.CheckBit(pendingAttestation.AttesterBitfield, i)
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
	attestation := &pb.AttestationRecord{
		AttesterBitfield: []byte{'F'},
	}

	indices := []uint32{0, 1, 2, 3, 4, 5, 6, 7}

	isValid := AreAttesterBitfieldsValid(attestation, indices)
	if !isValid {
		t.Fatalf("expected validation to pass for bitfield %v and indices %v", attestation, indices)
	}
}

func TestAreAttesterBitfieldsValidFalse(t *testing.T) {
	attestation := &pb.AttestationRecord{
		AttesterBitfield: []byte{'F', 'F'},
	}

	indices := []uint32{0, 1, 2, 3, 4, 5, 6, 7}

	isValid := AreAttesterBitfieldsValid(attestation, indices)
	if isValid {
		t.Fatalf("expected validation to fail for bitfield %v and indices %v", attestation, indices)
	}
}

func TestAreAttesterBitfieldsValidZerofill(t *testing.T) {
	attestation := &pb.AttestationRecord{
		AttesterBitfield: []byte{'F'},
	}

	indices := []uint32{0, 1, 2, 3, 4, 5, 6}

	isValid := AreAttesterBitfieldsValid(attestation, indices)
	if !isValid {
		t.Fatalf("expected validation to pass for bitfield %v and indices %v", attestation, indices)
	}
}

func TestAreAttesterBitfieldsValidNoZerofill(t *testing.T) {
	attestation := &pb.AttestationRecord{
		AttesterBitfield: []byte{'E'},
	}

	indices := []uint32{0, 1, 2, 3, 4, 5, 6}

	isValid := AreAttesterBitfieldsValid(attestation, indices)
	if isValid {
		t.Fatalf("expected validation to fail for bitfield %v and indices %v", attestation, indices)
	}
}
