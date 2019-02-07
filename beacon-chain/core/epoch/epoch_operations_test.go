package epoch

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func buildState(slot uint64, validatorCount uint64) *pb.BeaconState {
	validators := make([]*pb.Validator, validatorCount)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}
	validatorBalances := make([]uint64, len(validators))
	for i := 0; i < len(validatorBalances); i++ {
		validatorBalances[i] = params.BeaconConfig().MaxDeposit
	}
	return &pb.BeaconState{
		ValidatorRegistry: validators,
		ValidatorBalances: validatorBalances,
		Slot:              slot,
	}
}

func TestEpochAttestations(t *testing.T) {
	if params.BeaconConfig().EpochLength != 64 {
		t.Errorf("EpochLength should be 64 for these tests to pass")
	}

	var pendingAttestations []*pb.PendingAttestationRecord
	for i := uint64(0); i < params.BeaconConfig().EpochLength*3; i++ {
		pendingAttestations = append(pendingAttestations, &pb.PendingAttestationRecord{
			Data: &pb.AttestationData{
				Slot: i,
			},
		})
	}

	state := &pb.BeaconState{LatestAttestations: pendingAttestations}

	tests := []struct {
		stateSlot            uint64
		firstAttestationSlot uint64
	}{
		{
			stateSlot:            10,
			firstAttestationSlot: 10 - 10%params.BeaconConfig().EpochLength,
		},
		{
			stateSlot:            63,
			firstAttestationSlot: 63 - 63%params.BeaconConfig().EpochLength,
		},
		{
			stateSlot:            64,
			firstAttestationSlot: 64 - 64%params.BeaconConfig().EpochLength,
		}, {
			stateSlot:            127,
			firstAttestationSlot: 127 - 127%params.BeaconConfig().EpochLength,
		}, {
			stateSlot:            128,
			firstAttestationSlot: 128 - 128%params.BeaconConfig().EpochLength,
		},
	}

	for _, tt := range tests {
		state.Slot = tt.stateSlot

		if CurrentAttestations(state)[0].Data.Slot != tt.firstAttestationSlot {
			t.Errorf(
				"Result slot was an unexpected value. Wanted %d, got %d",
				tt.firstAttestationSlot,
				CurrentAttestations(state)[0].Data.Slot,
			)
		}
	}
}

func TestEpochBoundaryAttestations(t *testing.T) {
	if params.BeaconConfig().EpochLength != 64 {
		t.Errorf("EpochLength should be 64 for these tests to pass")
	}

	epochAttestations := []*pb.PendingAttestationRecord{
		{Data: &pb.AttestationData{JustifiedBlockRootHash32: []byte{0}, JustifiedSlot: 0}},
		{Data: &pb.AttestationData{JustifiedBlockRootHash32: []byte{1}, JustifiedSlot: 1}},
		{Data: &pb.AttestationData{JustifiedBlockRootHash32: []byte{2}, JustifiedSlot: 2}},
		{Data: &pb.AttestationData{JustifiedBlockRootHash32: []byte{3}, JustifiedSlot: 3}},
	}

	var latestBlockRootHash [][]byte
	for i := uint64(0); i < params.BeaconConfig().EpochLength; i++ {
		latestBlockRootHash = append(latestBlockRootHash, []byte{byte(i)})
	}

	state := &pb.BeaconState{
		LatestAttestations:     epochAttestations,
		LatestBlockRootHash32S: latestBlockRootHash,
	}

	if _, err := CurrentBoundaryAttestations(state, epochAttestations); err == nil {
		t.Fatal("CurrentBoundaryAttestations should have failed with empty block root hash")
	}

	state.Slot = params.BeaconConfig().EpochLength
	epochBoundaryAttestation, err := CurrentBoundaryAttestations(state, epochAttestations)
	if err != nil {
		t.Fatalf("CurrentBoundaryAttestations failed: %v", err)
	}

	if epochBoundaryAttestation[0].Data.JustifiedEpoch != 0 {
		t.Errorf("Wanted justified epoch 0 for epoch boundary attestation, got: %d",
			epochBoundaryAttestation[0].Data.JustifiedEpoch)
	}

	if !bytes.Equal(epochBoundaryAttestation[0].Data.JustifiedBlockRootHash32, []byte{0}) {
		t.Errorf("Wanted justified block hash [0] for epoch boundary attestation, got: %v",
			epochBoundaryAttestation[0].Data.JustifiedBlockRootHash32)
	}
}

func TestPrevEpochAttestations(t *testing.T) {
	if params.BeaconConfig().EpochLength != 64 {
		t.Errorf("EpochLength should be 64 for these tests to pass")
	}

	var pendingAttestations []*pb.PendingAttestationRecord
	for i := uint64(0); i < params.BeaconConfig().EpochLength*5; i++ {
		pendingAttestations = append(pendingAttestations, &pb.PendingAttestationRecord{
			Data: &pb.AttestationData{
				Slot: i,
			},
		})
	}

	state := &pb.BeaconState{LatestAttestations: pendingAttestations}

	tests := []struct {
		stateSlot            uint64
		firstAttestationSlot uint64
	}{
		{
			stateSlot:            127,
			firstAttestationSlot: 127 - params.BeaconConfig().EpochLength - 127%params.BeaconConfig().EpochLength,
		},
		{
			stateSlot:            128,
			firstAttestationSlot: 128 - params.BeaconConfig().EpochLength - 128%params.BeaconConfig().EpochLength,
		},
		{
			stateSlot:            383,
			firstAttestationSlot: 383 - params.BeaconConfig().EpochLength - 383%params.BeaconConfig().EpochLength,
		},
		{
			stateSlot:            129,
			firstAttestationSlot: 129 - params.BeaconConfig().EpochLength - 129%params.BeaconConfig().EpochLength,
		},
		{
			stateSlot:            256,
			firstAttestationSlot: 256 - params.BeaconConfig().EpochLength - 256%params.BeaconConfig().EpochLength,
		},
	}

	for _, tt := range tests {
		state.Slot = tt.stateSlot

		if PrevAttestations(state)[0].Data.Slot != tt.firstAttestationSlot {
			t.Errorf(
				"Result slot was an unexpected value. Wanted %d, got %d",
				tt.firstAttestationSlot,
				PrevAttestations(state)[0].Data.Slot,
			)
		}
	}
}

func TestPrevJustifiedAttestations(t *testing.T) {
	prevEpochAttestations := []*pb.PendingAttestationRecord{
		{Data: &pb.AttestationData{JustifiedSlot: 0}},
		{Data: &pb.AttestationData{JustifiedSlot: 2}},
		{Data: &pb.AttestationData{JustifiedSlot: 5}},
		{Data: &pb.AttestationData{Shard: 2, JustifiedSlot: 100}},
		{Data: &pb.AttestationData{Shard: 3, JustifiedSlot: 100}},
		{Data: &pb.AttestationData{JustifiedSlot: 999}},
	}

	thisEpochAttestations := []*pb.PendingAttestationRecord{
		{Data: &pb.AttestationData{JustifiedSlot: 0}},
		{Data: &pb.AttestationData{JustifiedSlot: 10}},
		{Data: &pb.AttestationData{JustifiedSlot: 15}},
		{Data: &pb.AttestationData{JustifiedSlot: 100}},
		{Data: &pb.AttestationData{Shard: 1, JustifiedSlot: 100}},
		{Data: &pb.AttestationData{JustifiedSlot: 888}},
	}

	state := &pb.BeaconState{PreviousJustifiedEpoch: 1}

	prevJustifiedAttestations := PrevJustifiedAttestations(state, thisEpochAttestations, prevEpochAttestations)

	for i, attestation := range prevJustifiedAttestations {
		if attestation.Data.Shard != uint64(i) {
			t.Errorf("Wanted shard %d, got %d", i, attestation.Data.Shard)
		}
		if attestation.Data.JustifiedSlot != 100 {
			t.Errorf("Wanted justified slot 100, got %d", attestation.Data.JustifiedSlot)
		}
	}
}

func TestPrevEpochBoundaryAttestations(t *testing.T) {
	if params.BeaconConfig().EpochLength != 64 {
		t.Errorf("EpochLength should be 64 for these tests to pass")
	}

	epochAttestations := []*pb.PendingAttestationRecord{
		{Data: &pb.AttestationData{EpochBoundaryRootHash32: []byte{100}}},
		{Data: &pb.AttestationData{EpochBoundaryRootHash32: []byte{0}}},
		{Data: &pb.AttestationData{EpochBoundaryRootHash32: []byte{64}}}, // selected
		{Data: &pb.AttestationData{EpochBoundaryRootHash32: []byte{55}}},
		{Data: &pb.AttestationData{EpochBoundaryRootHash32: []byte{64}}}, // selected
	}

	var latestBlockRootHash [][]byte
	for i := uint64(0); i < params.BeaconConfig().EpochLength*3; i++ {
		latestBlockRootHash = append(latestBlockRootHash, []byte{byte(i)})
	}

	state := &pb.BeaconState{
		Slot:                   3 * params.BeaconConfig().EpochLength,
		LatestBlockRootHash32S: latestBlockRootHash,
	}

	prevEpochBoundaryAttestation, err := PrevBoundaryAttestations(state, epochAttestations)
	if err != nil {
		t.Fatalf("EpochBoundaryAttestations failed: %v", err)
	}

	// 64 is selected because we start off with 3 epochs (192 slots)
	// The prev epoch boundary slot is 192 - 2 * epoch_length = 64
	if !bytes.Equal(prevEpochBoundaryAttestation[0].Data.EpochBoundaryRootHash32, []byte{64}) {
		t.Errorf("Wanted justified block hash [64] for epoch boundary attestation, got: %v",
			prevEpochBoundaryAttestation[0].Data.EpochBoundaryRootHash32)
	}
	if !bytes.Equal(prevEpochBoundaryAttestation[1].Data.EpochBoundaryRootHash32, []byte{64}) {
		t.Errorf("Wanted justified block hash [64] for epoch boundary attestation, got: %v",
			prevEpochBoundaryAttestation[1].Data.EpochBoundaryRootHash32)
	}
}

func TestHeadAttestationsOk(t *testing.T) {
	if params.BeaconConfig().EpochLength != 64 {
		t.Errorf("EpochLength should be 64 for these tests to pass")
	}

	prevAttestations := []*pb.PendingAttestationRecord{
		{Data: &pb.AttestationData{Slot: 1, BeaconBlockRootHash32: []byte{'A'}}},
		{Data: &pb.AttestationData{Slot: 2, BeaconBlockRootHash32: []byte{'A'}}},
		{Data: &pb.AttestationData{Slot: 3, BeaconBlockRootHash32: []byte{'A'}}},
		{Data: &pb.AttestationData{Slot: 4, BeaconBlockRootHash32: []byte{'A'}}},
	}

	state := &pb.BeaconState{Slot: 5, LatestBlockRootHash32S: [][]byte{{'A'}, {'A'}, {'A'}, {'A'}}}

	headAttestations, err := PrevHeadAttestations(state, prevAttestations)
	if err != nil {
		t.Fatalf("PrevHeadAttestations failed with %v", err)
	}

	if headAttestations[0].Data.Slot != 1 {
		t.Errorf("headAttestations[0] wanted slot 1, got slot %d", headAttestations[0].Data.Slot)
	}
	if headAttestations[1].Data.Slot != 2 {
		t.Errorf("headAttestations[1] wanted slot 2, got slot %d", headAttestations[1].Data.Slot)
	}
	if !bytes.Equal([]byte{'A'}, headAttestations[0].Data.BeaconBlockRootHash32) {
		t.Errorf("headAttestations[0] wanted hash [A], got slot %v",
			headAttestations[0].Data.BeaconBlockRootHash32)
	}
	if !bytes.Equal([]byte{'A'}, headAttestations[1].Data.BeaconBlockRootHash32) {
		t.Errorf("headAttestations[1] wanted hash [A], got slot %v",
			headAttestations[1].Data.BeaconBlockRootHash32)
	}
}

func TestHeadAttestationsNotOk(t *testing.T) {
	if params.BeaconConfig().EpochLength != 64 {
		t.Errorf("EpochLength should be 64 for these tests to pass")
	}

	prevAttestations := []*pb.PendingAttestationRecord{{Data: &pb.AttestationData{Slot: 1}}}

	state := &pb.BeaconState{Slot: 0}

	if _, err := PrevHeadAttestations(state, prevAttestations); err == nil {
		t.Fatal("PrevHeadAttestations should have failed with invalid range")
	}
}

func TestWinningRootOk(t *testing.T) {
	state := buildState(0, params.BeaconConfig().DepositsForChainStart)
	var participationBitfield []byte
	for i := 0; i < 16; i++ {
		participationBitfield = append(participationBitfield, byte(0x01))
	}

	// Generate 10 roots ([]byte{100}...[]byte{110})
	var attestations []*pb.PendingAttestationRecord
	for i := 0; i < 10; i++ {
		attestation := &pb.PendingAttestationRecord{
			Data: &pb.AttestationData{
				Slot:                 0,
				ShardBlockRootHash32: []byte{byte(i + 100)},
			},
			AggregationBitfield: participationBitfield,
		}
		attestations = append(attestations, attestation)
	}

	// Since all 10 roots have the balance of 64 ETHs
	// winningRoot chooses the lowest hash: []byte{100}
	winnerRoot, err := winningRoot(
		state,
		0,
		attestations,
		nil)
	if err != nil {
		t.Fatalf("Could not execute winningRoot: %v", err)
	}
	if !bytes.Equal(winnerRoot, []byte{100}) {
		t.Errorf("Incorrect winner root, wanted:[100], got: %v", winnerRoot)
	}
}

func TestWinningRootCantGetParticipantBitfield(t *testing.T) {
	state := buildState(0, params.BeaconConfig().DepositsForChainStart)

	attestations := []*pb.PendingAttestationRecord{
		{Data: &pb.AttestationData{
			ShardBlockRootHash32: []byte{},
		},
			AggregationBitfield: []byte{},
		},
	}

	want := fmt.Sprintf("wanted participants bitfield length %d, got: %d", 16, 0)
	if _, err := winningRoot(state, 0, attestations, nil); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestAttestingValidatorsOk(t *testing.T) {
	state := buildState(0, params.BeaconConfig().EpochLength*2)

	var attestations []*pb.PendingAttestationRecord
	for i := 0; i < 10; i++ {
		attestation := &pb.PendingAttestationRecord{
			Data: &pb.AttestationData{
				ShardBlockRootHash32: []byte{byte(i + 100)},
			},
			AggregationBitfield: []byte{0xFF},
		}
		attestations = append(attestations, attestation)
	}

	attestedValidators, err := AttestingValidators(
		state,
		0,
		attestations,
		nil)
	if err != nil {
		t.Fatalf("Could not execute AttestingValidators: %v", err)
	}

	// Verify the winner root is attested by validator 109 97 based on shuffling.
	if !reflect.DeepEqual(attestedValidators, []uint64{109, 97}) {
		t.Errorf("Active validators don't match. Wanted:[237,224], Got: %v", attestedValidators)
	}
}

func TestAttestingValidatorsCantGetWinningRoot(t *testing.T) {
	state := buildState(0, params.BeaconConfig().DepositsForChainStart)

	attestation := &pb.PendingAttestationRecord{
		Data: &pb.AttestationData{
			ShardBlockRootHash32: []byte{},
		},
		AggregationBitfield: []byte{},
	}

	want := fmt.Sprintf("wanted participants bitfield length %d, got: %d", 16, 0)
	if _, err := AttestingValidators(state, 0, []*pb.PendingAttestationRecord{attestation}, nil); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestTotalAttestingBalanceOk(t *testing.T) {
	validatorsPerCommittee := uint64(2)
	state := buildState(0, 2*params.BeaconConfig().EpochLength)

	// Generate 10 roots ([]byte{100}...[]byte{110})
	var attestations []*pb.PendingAttestationRecord
	for i := 0; i < 10; i++ {
		attestation := &pb.PendingAttestationRecord{
			Data: &pb.AttestationData{
				ShardBlockRootHash32: []byte{byte(i + 100)},
			},
			// All validators attested to the above roots.
			AggregationBitfield: []byte{0xff},
		}
		attestations = append(attestations, attestation)
	}

	attestedBalance, err := TotalAttestingBalance(
		state,
		0,
		attestations,
		nil)
	if err != nil {
		t.Fatalf("Could not execute totalAttestingBalance: %v", err)
	}

	if attestedBalance != params.BeaconConfig().MaxDeposit*validatorsPerCommittee {
		t.Errorf("Incorrect attested balance. Wanted:64*1e9, Got: %d", attestedBalance)
	}
}

func TestTotalAttestingBalanceCantGetWinningRoot(t *testing.T) {
	state := buildState(0, params.BeaconConfig().DepositsForChainStart)

	attestation := &pb.PendingAttestationRecord{
		Data: &pb.AttestationData{
			ShardBlockRootHash32: []byte{},
		},
		AggregationBitfield: []byte{},
	}

	want := fmt.Sprintf("wanted participants bitfield length %d, got: %d", 16, 0)
	if _, err := TotalAttestingBalance(state, 0, []*pb.PendingAttestationRecord{attestation}, nil); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestTotalBalance(t *testing.T) {
	// Assign validators to different balances.
	state := &pb.BeaconState{
		Slot: 5,
		ValidatorBalances: []uint64{20 * 1e9, 25 * 1e9, 30 * 1e9, 30 * 1e9,
			32 * 1e9, 34 * 1e9, 50 * 1e9, 50 * 1e9},
	}

	// 20 + 25 + 30 + 30 + 32 + 32 + 32 + 32 = 233
	totalBalance := TotalBalance(state, []uint64{0, 1, 2, 3, 4, 5, 6, 7})
	if totalBalance != 233*1e9 {
		t.Errorf("Incorrect total balance. Wanted: 233*1e9, got: %d", totalBalance)
	}
}

func TestInclusionSlotOk(t *testing.T) {
	state := buildState(0, params.BeaconConfig().DepositsForChainStart)
	var participationBitfield []byte
	for i := 0; i < 16; i++ {
		participationBitfield = append(participationBitfield, byte(0xff))
	}

	state.LatestAttestations = []*pb.PendingAttestationRecord{
		{Data: &pb.AttestationData{},
			AggregationBitfield: participationBitfield,
			SlotIncluded:        101},
		{Data: &pb.AttestationData{},
			AggregationBitfield: participationBitfield,
			SlotIncluded:        100},
		{Data: &pb.AttestationData{},
			AggregationBitfield: participationBitfield,
			SlotIncluded:        102},
	}
	slot, err := InclusionSlot(state, 237)
	if err != nil {
		t.Fatalf("Could not execute InclusionSlot: %v", err)
	}
	// validator 45's attestation got included in slot 100.
	if slot != 100 {
		t.Errorf("Incorrect slot. Wanted: 100, got: %d", slot)
	}
}

func TestInclusionSlotBadBitfield(t *testing.T) {
	state := buildState(0, params.BeaconConfig().DepositsForChainStart)

	state.LatestAttestations = []*pb.PendingAttestationRecord{
		{Data: &pb.AttestationData{},
			AggregationBitfield: []byte{},
			SlotIncluded:        100},
	}

	want := fmt.Sprintf("wanted participants bitfield length %d, got: %d", 16, 0)
	if _, err := InclusionSlot(state, 0); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestInclusionSlotNotFound(t *testing.T) {
	state := buildState(0, params.BeaconConfig().EpochLength)

	badIndex := uint64(10000)
	want := fmt.Sprintf("could not find inclusion slot for validator index %d", badIndex)
	if _, err := InclusionSlot(state, badIndex); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestInclusionDistanceOk(t *testing.T) {
	state := buildState(0, params.BeaconConfig().DepositsForChainStart)
	var participationBitfield []byte
	for i := 0; i < 16; i++ {
		participationBitfield = append(participationBitfield, byte(0xff))
	}

	state.LatestAttestations = []*pb.PendingAttestationRecord{
		{Data: &pb.AttestationData{},
			AggregationBitfield: participationBitfield,
			SlotIncluded:        100},
	}
	distance, err := InclusionDistance(state, 237)
	if err != nil {
		t.Fatalf("Could not execute InclusionDistance: %v", err)
	}

	// Inclusion distance is 100 because input validator index is 45,
	// validator 45's attested slot 0 and got included slot 100.
	if distance != state.LatestAttestations[0].SlotIncluded {
		t.Errorf("Incorrect distance. Wanted: %d, got: %d",
			state.LatestAttestations[0].SlotIncluded, distance)
	}
}

func TestInclusionDistanceBadBitfield(t *testing.T) {
	state := buildState(0, params.BeaconConfig().DepositsForChainStart)

	state.LatestAttestations = []*pb.PendingAttestationRecord{
		{Data: &pb.AttestationData{},
			AggregationBitfield: []byte{},
			SlotIncluded:        100},
	}

	want := fmt.Sprintf("wanted participants bitfield length %d, got: %d", 16, 0)
	if _, err := InclusionDistance(state, 0); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestInclusionDistanceNotFound(t *testing.T) {
	state := buildState(0, params.BeaconConfig().EpochLength)

	badIndex := uint64(10000)
	want := fmt.Sprintf("could not find inclusion distance for validator index %d", badIndex)
	if _, err := InclusionDistance(state, badIndex); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}
