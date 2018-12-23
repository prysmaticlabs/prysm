package epoch

import (
	"bytes"
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestCanProcessEpoch(t *testing.T) {
	if params.BeaconConfig().EpochLength != 64 {
		t.Errorf("EpochLength should be 64 for these tests to pass")
	}
	tests := []struct {
		slot            uint64
		canProcessEpoch bool
	}{
		{
			slot:            1,
			canProcessEpoch: false,
		},
		{
			slot:            63,
			canProcessEpoch: false,
		},
		{
			slot:            64,
			canProcessEpoch: true,
		}, {
			slot:            128,
			canProcessEpoch: true,
		}, {
			slot:            1000000000,
			canProcessEpoch: true,
		},
	}
	for _, tt := range tests {
		state := &pb.BeaconState{Slot: tt.slot}
		if CanProcessEpoch(state) != tt.canProcessEpoch {
			t.Errorf(
				"CanProcessEpoch(%d) = %v. Wanted %v",
				tt.slot,
				CanProcessEpoch(state),
				tt.canProcessEpoch,
			)
		}
	}
}

func TestCanProcessReceiptRoots(t *testing.T) {
	if params.BeaconConfig().PowReceiptRootVotingPeriod != 1024 {
		t.Errorf("PowReceiptRootVotingPeriod should be 1024 for these tests to pass")
	}
	tests := []struct {
		slot                   uint64
		canProcessReceiptRoots bool
	}{
		{
			slot:                   1,
			canProcessReceiptRoots: false,
		},
		{
			slot:                   1022,
			canProcessReceiptRoots: false,
		},
		{
			slot:                   1024,
			canProcessReceiptRoots: true,
		}, {
			slot:                   4096,
			canProcessReceiptRoots: true,
		}, {
			slot:                   234234,
			canProcessReceiptRoots: false,
		},
	}
	for _, tt := range tests {
		state := &pb.BeaconState{Slot: tt.slot}
		if CanProcessReceiptRoots(state) != tt.canProcessReceiptRoots {
			t.Errorf(
				"CanProcessReceiptRoots(%d) = %v. Wanted %v",
				tt.slot,
				CanProcessReceiptRoots(state),
				tt.canProcessReceiptRoots,
			)
		}
	}
}

func TestProcessReceipt(t *testing.T) {
	if params.BeaconConfig().PowReceiptRootVotingPeriod != 1024 {
		t.Errorf("PowReceiptRootVotingPeriod should be 1024 for these tests to pass")
	}
	requiredVoteCount := params.BeaconConfig().PowReceiptRootVotingPeriod
	state := &pb.BeaconState{
		CandidatePowReceiptRoots: []*pb.CandidatePoWReceiptRootRecord{
			{VoteCount: 0, CandidatePowReceiptRootHash32: []byte{'A'}},
			// CandidatePowReceiptRootHash32 ['B'] gets to process with sufficient vote count.
			{VoteCount: requiredVoteCount/2 + 1, CandidatePowReceiptRootHash32: []byte{'B'}},
			{VoteCount: requiredVoteCount / 2, CandidatePowReceiptRootHash32: []byte{'C'}},
		},
	}
	newState := ProcessReceipt(state)
	if !bytes.Equal(newState.ProcessedPowReceiptRootHash32, []byte{'B'}) {
		t.Errorf("Incorrect ProcessedPowReceiptRootHash32. Wanted: %v, got: %v",
			[]byte{'B'}, newState.ProcessedPowReceiptRootHash32)
	}

	// Adding a new receipt root ['D'] which should be the new processed receipt root.
	state.CandidatePowReceiptRoots = append(state.CandidatePowReceiptRoots,
		&pb.CandidatePoWReceiptRootRecord{VoteCount: requiredVoteCount,
			CandidatePowReceiptRootHash32: []byte{'D'}})
	newState = ProcessReceipt(state)
	if !bytes.Equal(newState.ProcessedPowReceiptRootHash32, []byte{'D'}) {
		t.Errorf("Incorrect ProcessedPowReceiptRootHash32. Wanted: %v, got: %v",
			[]byte{'D'}, newState.ProcessedPowReceiptRootHash32)
	}

	if len(newState.CandidatePowReceiptRoots) != 0 {
		t.Errorf("Failed to clean up CandidatePowReceiptRoots slice. Length: %d",
			len(newState.CandidatePowReceiptRoots))
	}
}

func TestProcessJustification(t *testing.T) {
	if params.BeaconConfig().EpochLength != 64 {
		t.Errorf("EpochLength should be 64 for these tests to pass")
	}

	state := &pb.BeaconState{
		Slot:                  300,
		JustifiedSlot:         200,
		JustificationBitfield: 4,
	}
	newState := ProcessJustification(state, 1, 1, 1)

	if newState.PreviousJustifiedSlot != state.JustifiedSlot {
		t.Errorf("New state's prev justified slot %d != old state's justified slot %d",
			newState.PreviousJustifiedSlot, state.JustifiedSlot)
	}
	// Since this epoch was justified (not prev), justified_slot = state.slot - EPOCH_LENGTH.
	if newState.JustifiedSlot != state.Slot-params.BeaconConfig().EpochLength {
		t.Errorf("New state's justified slot %d != state's slot - EPOCH_LENGTH %d",
			newState.JustifiedSlot, state.Slot-params.BeaconConfig().EpochLength)
	}
	// The new JustificationBitfield is 11, it went from 0100 to 1011. Two 1's were appended because both
	// prev epoch and this epoch were justified.
	if newState.JustificationBitfield != 11 {
		t.Errorf("New state's justification bitfield %d != 11", newState.JustificationBitfield)
	}

	// Assume for the case where only prev epoch got justified. Verify
	// justified_slot = state.slot - 2 * EPOCH_LENGTH.
	newState = ProcessJustification(state, 0, 1, 1)
	if newState.JustifiedSlot != state.Slot-2*params.BeaconConfig().EpochLength {
		t.Errorf("New state's justified slot %d != state's slot - 2 * EPOCH_LENGTH %d",
			newState.JustifiedSlot, state.Slot-params.BeaconConfig().EpochLength)
	}
}

func TestProcessFinalization(t *testing.T) {
	if params.BeaconConfig().EpochLength != 64 {
		t.Errorf("EpochLength should be 64 for these tests to pass")
	}
	epochLength := params.BeaconConfig().EpochLength

	// 2 consecutive justified slot in a row,
	// and previous justified slot is state slot - 2 * EPOCH_LENGTH.
	state := &pb.BeaconState{
		Slot:                  200,
		JustifiedSlot:         200 - epochLength,
		PreviousJustifiedSlot: 200 - 2*epochLength,
		JustificationBitfield: 3,
	}
	newState := ProcessFinalization(state)
	if newState.FinalizedSlot != state.JustifiedSlot {
		t.Errorf("Wanted finalized slot to be %d, got %d:",
			state.JustifiedSlot, newState.FinalizedSlot)
	}

	// 3 consecutive justified slot in a row.
	// and previous justified slot is state slot - 3 * EPOCH_LENGTH.
	state = &pb.BeaconState{
		Slot:                  300,
		JustifiedSlot:         300 - epochLength,
		PreviousJustifiedSlot: 300 - 3*epochLength,
		JustificationBitfield: 7,
	}
	newState = ProcessFinalization(state)
	if newState.FinalizedSlot != state.JustifiedSlot {
		t.Errorf("Wanted finalized slot to be %d, got %d:",
			state.JustifiedSlot, newState.FinalizedSlot)
	}

	// 4 consecutive justified slot in a row.
	// and previous justified slot is state slot - 3 * EPOCH_LENGTH.
	state = &pb.BeaconState{
		Slot:                  400,
		JustifiedSlot:         400 - epochLength,
		PreviousJustifiedSlot: 400 - 4*epochLength,
		JustificationBitfield: 15,
	}
	newState = ProcessFinalization(state)
	if newState.FinalizedSlot != state.JustifiedSlot {
		t.Errorf("Wanted finalized slot to be %d, got %d:",
			state.JustifiedSlot, newState.FinalizedSlot)
	}

	// if nothing gets finalized it just returns the same state.
	state = &pb.BeaconState{
		Slot:                  100,
		JustifiedSlot:         65,
		PreviousJustifiedSlot: 0,
		JustificationBitfield: 1,
	}
	newState = ProcessFinalization(state)
	if newState.FinalizedSlot != 0 {
		t.Errorf("Wanted finalized slot to be %d, got %d:",
			0, newState.FinalizedSlot)
	}
}
