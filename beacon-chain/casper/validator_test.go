package casper

import (
	"bytes"
	"math"
	"reflect"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/params"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestRotateValidatorSet(t *testing.T) {
	validators := []*pb.ValidatorRecord{
		{Balance: 10, StartDynasty: 0, EndDynasty: params.DefaultEndDynasty}, // half below default balance, should be moved to exit.
		{Balance: 15, StartDynasty: 1, EndDynasty: params.DefaultEndDynasty}, // half below default balance, should be moved to exit.
		{Balance: 20, StartDynasty: 2, EndDynasty: params.DefaultEndDynasty}, // stays in active.
		{Balance: 25, StartDynasty: 3, EndDynasty: params.DefaultEndDynasty}, // stays in active.
		{Balance: 30, StartDynasty: 4, EndDynasty: params.DefaultEndDynasty}, // stays in active.
	}

	data := &pb.CrystallizedState{
		Validators:     validators,
		CurrentDynasty: 10,
	}
	state := types.NewCrystallizedState(data)

	// rotate validator set and increment dynasty count by 1.
	RotateValidatorSet(state)
	state.IncrementCurrentDynasty()

	if !reflect.DeepEqual(ActiveValidatorIndices(state), []int{2, 3, 4}) {
		t.Errorf("active validator indices should be [2,3,4], got: %v", ActiveValidatorIndices(state))
	}
	if len(QueuedValidatorIndices(state)) != 0 {
		t.Errorf("queued validator indices should be [], got: %v", QueuedValidatorIndices(state))
	}
	if !reflect.DeepEqual(ExitedValidatorIndices(state), []int{0, 1}) {
		t.Errorf("exited validator indices should be [0,1], got: %v", ExitedValidatorIndices(state))
	}
}

func TestHasVoted(t *testing.T) {
	// Setting bit field to 11111111.
	pendingAttestation := &pb.AttestationRecord{
		AttesterBitfield: []byte{255},
	}
	active := types.NewActiveState(&pb.ActiveState{})
	active.NewPendingAttestation(pendingAttestation)

	for i := 0; i < len(active.LatestPendingAttestation().AttesterBitfield); i++ {
		voted, err := utils.CheckBit(active.LatestPendingAttestation().AttesterBitfield, i)
		if err != nil {
			t.Errorf("checking bitfield for vote failed: %v", err)
		}
		if !voted {
			t.Error("validator voted but received didn't vote")
		}
	}

	// Setting bit field to 01010101.
	pendingAttestation = &pb.AttestationRecord{
		AttesterBitfield: []byte{85},
	}
	active.NewPendingAttestation(pendingAttestation)

	for i := 0; i < len(active.LatestPendingAttestation().AttesterBitfield); i++ {
		voted, err := utils.CheckBit(active.LatestPendingAttestation().AttesterBitfield, i)
		if err != nil {
			t.Errorf("checking bitfield for vote failed: %v", err)
		}
		if i%2 == 0 && voted {
			t.Error("validator didn't vote but received voted")
		}
		if i%2 == 1 && !voted {
			t.Error("validator voted but received didn't vote")
		}
	}
}

func TestClearAttesterBitfields(t *testing.T) {
	// Testing validator set sizes from 1 to 100.
	for j := 1; j <= 100; j++ {
		var validators []*pb.ValidatorRecord

		for i := 0; i < j; i++ {
			validator := &pb.ValidatorRecord{WithdrawalAddress: []byte{}, PublicKey: 0}
			validators = append(validators, validator)
		}

		crystallized := types.NewCrystallizedState(&pb.CrystallizedState{})
		active := types.NewActiveState(&pb.ActiveState{})

		testAttesterBitfield := []byte{1, 2, 3, 4}
		crystallized.SetValidators(validators)
		active.NewPendingAttestation(&pb.AttestationRecord{AttesterBitfield: testAttesterBitfield})
		active.ClearPendingAttestations()

		if bytes.Equal(testAttesterBitfield, active.LatestPendingAttestation().AttesterBitfield) {
			t.Fatalf("attester bitfields have not been able to be reset: %v", testAttesterBitfield)
		}

		if !bytes.Equal(active.LatestPendingAttestation().AttesterBitfield, []byte{}) {
			t.Fatalf("attester bitfields are not zeroed out: %v", active.LatestPendingAttestation().AttesterBitfield)
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

	crystallized := types.NewCrystallizedState(data)

	if !reflect.DeepEqual(ActiveValidatorIndices(crystallized), []int{0, 1, 2, 3, 4}) {
		t.Errorf("active validator indices should be [0 1 2 3 4], got: %v", ActiveValidatorIndices(crystallized))
	}
	if !reflect.DeepEqual(QueuedValidatorIndices(crystallized), []int{5}) {
		t.Errorf("queued validator indices should be [5], got: %v", QueuedValidatorIndices(crystallized))
	}
	if len(ExitedValidatorIndices(crystallized)) != 0 {
		t.Errorf("exited validator indices to be empty, got: %v", ExitedValidatorIndices(crystallized))
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

	crystallized = types.NewCrystallizedState(data)

	if !reflect.DeepEqual(ActiveValidatorIndices(crystallized), []int{0, 1}) {
		t.Errorf("active validator indices should be [0 1 2 4 5], got: %v", ActiveValidatorIndices(crystallized))
	}
	if !reflect.DeepEqual(QueuedValidatorIndices(crystallized), []int{2, 3}) {
		t.Errorf("queued validator indices should be [3], got: %v", QueuedValidatorIndices(crystallized))
	}
	if !reflect.DeepEqual(ExitedValidatorIndices(crystallized), []int{4, 5}) {
		t.Errorf("exited validator indices should be [3], got: %v", ExitedValidatorIndices(crystallized))
	}
}

// NewBlock is a helper method to create blocks with valid defaults.
// For a generic block, use NewBlock(t, nil).
func NewBlock(t *testing.T, b *pb.BeaconBlock) *types.Block {
	if b == nil {
		b = &pb.BeaconBlock{}
	}
	if b.ActiveStateHash == nil {
		b.ActiveStateHash = make([]byte, 32)
	}
	if b.CrystallizedStateHash == nil {
		b.CrystallizedStateHash = make([]byte, 32)
	}
	if b.ParentHash == nil {
		b.ParentHash = make([]byte, 32)
	}

	return types.NewBlock(b)
}
