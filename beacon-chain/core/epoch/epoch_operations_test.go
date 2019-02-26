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
		validatorBalances[i] = params.BeaconConfig().MaxDepositAmount
	}
	return &pb.BeaconState{
		ValidatorRegistry: validators,
		ValidatorBalances: validatorBalances,
		Slot:              slot,
	}
}

func TestEpochAttestations_AttestationSlotValid(t *testing.T) {
	if params.BeaconConfig().SlotsPerEpoch != 64 {
		t.Errorf("SlotsPerEpoch should be 64 for these tests to pass")
	}

	var pendingAttestations []*pb.PendingAttestation
	for i := uint64(0); i < params.BeaconConfig().SlotsPerEpoch*3; i++ {
		pendingAttestations = append(pendingAttestations, &pb.PendingAttestation{
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
			firstAttestationSlot: 10 - 10%params.BeaconConfig().SlotsPerEpoch,
		},
		{
			stateSlot:            63,
			firstAttestationSlot: 63 - 63%params.BeaconConfig().SlotsPerEpoch,
		},
		{
			stateSlot:            64,
			firstAttestationSlot: 64 - 64%params.BeaconConfig().SlotsPerEpoch,
		}, {
			stateSlot:            127,
			firstAttestationSlot: 127 - 127%params.BeaconConfig().SlotsPerEpoch,
		}, {
			stateSlot:            128,
			firstAttestationSlot: 128 - 128%params.BeaconConfig().SlotsPerEpoch,
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

func TestEpochBoundaryAttestations_AccurateAttestationData(t *testing.T) {
	if params.BeaconConfig().SlotsPerEpoch != 64 {
		t.Errorf("SlotsPerEpoch should be 64 for these tests to pass")
	}

	epochAttestations := []*pb.PendingAttestation{
		{
			Data: &pb.AttestationData{
				EpochBoundaryRootHash32: []byte{64},
				JustifiedEpoch:          params.BeaconConfig().GenesisEpoch,
			},
		},
		{
			Data: &pb.AttestationData{
				EpochBoundaryRootHash32: []byte{64},
				JustifiedEpoch:          params.BeaconConfig().GenesisEpoch,
			},
		},
		{
			Data: &pb.AttestationData{
				EpochBoundaryRootHash32: []byte{64},
				JustifiedEpoch:          params.BeaconConfig().GenesisEpoch,
			},
		},
		{
			Data: &pb.AttestationData{
				EpochBoundaryRootHash32: []byte{64},
				JustifiedEpoch:          params.BeaconConfig().GenesisEpoch,
			},
		},
	}

	var latestBlockRootHash [][]byte
	for i := uint64(0); i < params.BeaconConfig().LatestBlockRootsLength; i++ {
		latestBlockRootHash = append(latestBlockRootHash, []byte{byte(i)})
	}

	state := &pb.BeaconState{
		LatestAttestations:     epochAttestations,
		LatestBlockRootHash32S: latestBlockRootHash,
		JustifiedEpoch:         params.BeaconConfig().GenesisEpoch,
	}

	if _, err := CurrentBoundaryAttestations(state, epochAttestations); err == nil {
		t.Fatal("CurrentBoundaryAttestations should have failed with empty block root hash")
	}

	state.Slot = params.BeaconConfig().SlotsPerEpoch + params.BeaconConfig().GenesisSlot + 1
	epochBoundaryAttestation, err := CurrentBoundaryAttestations(state, epochAttestations)
	if err != nil {
		t.Fatalf("CurrentBoundaryAttestations failed: %v", err)
	}

	if epochBoundaryAttestation[0].Data.JustifiedEpoch != params.BeaconConfig().GenesisEpoch {
		t.Errorf("Wanted justified epoch 0 for epoch boundary attestation, got: %d",
			epochBoundaryAttestation[0].Data.JustifiedEpoch)
	}

	if !bytes.Equal(epochBoundaryAttestation[0].Data.EpochBoundaryRootHash32, []byte{64}) {
		t.Errorf("Wanted epoch boundary block hash [64] for epoch boundary attestation, got: %v",
			epochBoundaryAttestation[0].Data.EpochBoundaryRootHash32)
	}
}

func TestPrevEpochAttestations_AccurateAttestationSlots(t *testing.T) {
	if params.BeaconConfig().SlotsPerEpoch != 64 {
		t.Errorf("SlotsPerEpoch should be 64 for these tests to pass")
	}

	var pendingAttestations []*pb.PendingAttestation
	for i := uint64(0); i < params.BeaconConfig().SlotsPerEpoch*5; i++ {
		pendingAttestations = append(pendingAttestations, &pb.PendingAttestation{
			Data: &pb.AttestationData{
				Slot: i + params.BeaconConfig().GenesisSlot,
			},
		})
	}

	state := &pb.BeaconState{LatestAttestations: pendingAttestations}

	tests := []struct {
		stateSlot            uint64
		firstAttestationSlot uint64
	}{
		{
			stateSlot: 127 + params.BeaconConfig().GenesisSlot,
			firstAttestationSlot: 127 - params.BeaconConfig().SlotsPerEpoch -
				127%params.BeaconConfig().SlotsPerEpoch +
				params.BeaconConfig().GenesisSlot,
		},
		{
			stateSlot: 128 + params.BeaconConfig().GenesisSlot,
			firstAttestationSlot: 128 - params.BeaconConfig().SlotsPerEpoch -
				128%params.BeaconConfig().SlotsPerEpoch +
				params.BeaconConfig().GenesisSlot,
		},
		{
			stateSlot: 383 + params.BeaconConfig().GenesisSlot,
			firstAttestationSlot: 383 - params.BeaconConfig().SlotsPerEpoch -
				383%params.BeaconConfig().SlotsPerEpoch +
				params.BeaconConfig().GenesisSlot,
		},
		{
			stateSlot: 129 + params.BeaconConfig().GenesisSlot,
			firstAttestationSlot: 129 - params.BeaconConfig().SlotsPerEpoch -
				129%params.BeaconConfig().SlotsPerEpoch +
				params.BeaconConfig().GenesisSlot,
		},
		{
			stateSlot: 256 + params.BeaconConfig().GenesisSlot,
			firstAttestationSlot: 256 - params.BeaconConfig().SlotsPerEpoch -
				256%params.BeaconConfig().SlotsPerEpoch +
				params.BeaconConfig().GenesisSlot,
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

func TestPrevJustifiedAttestations_AccurateShardsAndEpoch(t *testing.T) {
	prevEpochAttestations := []*pb.PendingAttestation{
		{Data: &pb.AttestationData{JustifiedEpoch: 0}},
		{Data: &pb.AttestationData{JustifiedEpoch: 0}},
		{Data: &pb.AttestationData{JustifiedEpoch: 0}},
		{Data: &pb.AttestationData{Shard: 2, JustifiedEpoch: 1}},
		{Data: &pb.AttestationData{Shard: 3, JustifiedEpoch: 1}},
		{Data: &pb.AttestationData{JustifiedEpoch: 15}},
	}

	thisEpochAttestations := []*pb.PendingAttestation{
		{Data: &pb.AttestationData{JustifiedEpoch: 0}},
		{Data: &pb.AttestationData{JustifiedEpoch: 0}},
		{Data: &pb.AttestationData{JustifiedEpoch: 0}},
		{Data: &pb.AttestationData{JustifiedEpoch: 1}},
		{Data: &pb.AttestationData{Shard: 1, JustifiedEpoch: 1}},
		{Data: &pb.AttestationData{JustifiedEpoch: 13}},
	}

	state := &pb.BeaconState{PreviousJustifiedEpoch: 1}

	prevJustifiedAttestations := PrevJustifiedAttestations(state, thisEpochAttestations, prevEpochAttestations)

	for i, attestation := range prevJustifiedAttestations {
		if attestation.Data.Shard != uint64(i) {
			t.Errorf("Wanted shard %d, got %d", i, attestation.Data.Shard)
		}
		if attestation.Data.JustifiedEpoch != 1 {
			t.Errorf("Wanted justified epoch 0, got %d", attestation.Data.JustifiedEpoch)
		}
	}
}

func TestPrevEpochBoundaryAttestations_AccurateAttestationData(t *testing.T) {
	if params.BeaconConfig().SlotsPerEpoch != 64 {
		t.Errorf("SlotsPerEpoch should be 64 for these tests to pass")
	}

	epochAttestations := []*pb.PendingAttestation{
		{Data: &pb.AttestationData{EpochBoundaryRootHash32: []byte{100}}},
		{Data: &pb.AttestationData{EpochBoundaryRootHash32: []byte{0}}},
		{Data: &pb.AttestationData{EpochBoundaryRootHash32: []byte{128}}}, // selected
		{Data: &pb.AttestationData{EpochBoundaryRootHash32: []byte{55}}},
		{Data: &pb.AttestationData{EpochBoundaryRootHash32: []byte{128}}}, // selected
	}

	var latestBlockRootHash [][]byte
	for i := uint64(0); i < params.BeaconConfig().LatestBlockRootsLength; i++ {
		latestBlockRootHash = append(latestBlockRootHash, []byte{byte(i)})
	}

	state := &pb.BeaconState{
		Slot:                   3*params.BeaconConfig().SlotsPerEpoch + params.BeaconConfig().GenesisSlot,
		LatestBlockRootHash32S: latestBlockRootHash,
		JustifiedEpoch:         params.BeaconConfig().GenesisEpoch,
	}

	prevEpochBoundaryAttestation, err := PrevBoundaryAttestations(state, epochAttestations)
	if err != nil {
		t.Fatalf("EpochBoundaryAttestations failed: %v", err)
	}

	//128 is selected because that's the start root of prev boundary epoch.
	if !bytes.Equal(prevEpochBoundaryAttestation[0].Data.EpochBoundaryRootHash32, []byte{128}) {
		t.Errorf("Wanted justified block hash [128] for epoch boundary attestation, got: %v",
			prevEpochBoundaryAttestation[0].Data.EpochBoundaryRootHash32)
	}
	if !bytes.Equal(prevEpochBoundaryAttestation[1].Data.EpochBoundaryRootHash32, []byte{128}) {
		t.Errorf("Wanted justified block hash [128] for epoch boundary attestation, got: %v",
			prevEpochBoundaryAttestation[1].Data.EpochBoundaryRootHash32)
	}
}

func TestHeadAttestations_AccurateHeadData(t *testing.T) {
	if params.BeaconConfig().SlotsPerEpoch != 64 {
		t.Errorf("SlotsPerEpoch should be 64 for these tests to pass")
	}

	prevAttestations := []*pb.PendingAttestation{
		{Data: &pb.AttestationData{Slot: params.BeaconConfig().GenesisSlot + 1, BeaconBlockRootHash32: []byte{'A'}}},
		{Data: &pb.AttestationData{Slot: params.BeaconConfig().GenesisSlot + 2, BeaconBlockRootHash32: []byte{'A'}}},
		{Data: &pb.AttestationData{Slot: params.BeaconConfig().GenesisSlot + 3, BeaconBlockRootHash32: []byte{'A'}}},
		{Data: &pb.AttestationData{Slot: params.BeaconConfig().GenesisSlot + 4, BeaconBlockRootHash32: []byte{'A'}}},
	}

	var latestBlockRootHash [][]byte
	for i := uint64(0); i < params.BeaconConfig().LatestBlockRootsLength; i++ {
		latestBlockRootHash = append(latestBlockRootHash, []byte{byte('A')})
	}

	state := &pb.BeaconState{
		Slot:                   params.BeaconConfig().GenesisSlot + 5,
		LatestBlockRootHash32S: latestBlockRootHash,
		JustifiedEpoch:         params.BeaconConfig().GenesisEpoch}

	headAttestations, err := PrevHeadAttestations(state, prevAttestations)
	if err != nil {
		t.Fatalf("PrevHeadAttestations failed with %v", err)
	}

	if headAttestations[0].Data.Slot != params.BeaconConfig().GenesisSlot+1 {
		t.Errorf("headAttestations[0] wanted slot 9223372036854775809, got slot %d", headAttestations[0].Data.Slot)
	}
	if headAttestations[1].Data.Slot != params.BeaconConfig().GenesisSlot+2 {
		t.Errorf("headAttestations[1] wanted slot 9223372036854775810, got slot %d", headAttestations[1].Data.Slot)
	}
	if !bytes.Equal(headAttestations[0].Data.BeaconBlockRootHash32, []byte{'A'}) {
		t.Errorf("headAttestations[0] wanted hash [A], got slot %v",
			headAttestations[0].Data.BeaconBlockRootHash32)
	}
	if !bytes.Equal(headAttestations[1].Data.BeaconBlockRootHash32, []byte{'A'}) {
		t.Errorf("headAttestations[1] wanted hash [A], got slot %v",
			headAttestations[1].Data.BeaconBlockRootHash32)
	}
}

func TestHeadAttestations_InvalidRange(t *testing.T) {
	if params.BeaconConfig().SlotsPerEpoch != 64 {
		t.Errorf("SlotsPerEpoch should be 64 for these tests to pass")
	}

	prevAttestations := []*pb.PendingAttestation{{Data: &pb.AttestationData{Slot: 1}}}

	state := &pb.BeaconState{Slot: 0}

	if _, err := PrevHeadAttestations(state, prevAttestations); err == nil {
		t.Fatal("PrevHeadAttestations should have failed with invalid range")
	}
}

func TestWinningRoot_AccurateRoot(t *testing.T) {
	state := buildState(params.BeaconConfig().GenesisSlot, 100)
	var participationBitfield []byte
	participationBitfield = append(participationBitfield, byte(0x01))

	// Generate 10 roots ([]byte{100}...[]byte{110})
	var attestations []*pb.PendingAttestation
	for i := 0; i < 10; i++ {
		attestation := &pb.PendingAttestation{
			Data: &pb.AttestationData{
				Slot:                 params.BeaconConfig().GenesisSlot,
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

func TestWinningRoot_EmptyParticipantBitfield(t *testing.T) {
	state := buildState(params.BeaconConfig().GenesisSlot, params.BeaconConfig().DepositsForChainStart)

	attestations := []*pb.PendingAttestation{
		{Data: &pb.AttestationData{
			Slot:                 params.BeaconConfig().GenesisSlot,
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

func TestAttestingValidators_MatchActive(t *testing.T) {
	state := buildState(params.BeaconConfig().GenesisSlot, params.BeaconConfig().SlotsPerEpoch*2)

	var attestations []*pb.PendingAttestation
	for i := 0; i < 10; i++ {
		attestation := &pb.PendingAttestation{
			Data: &pb.AttestationData{
				Slot:                 params.BeaconConfig().GenesisSlot,
				ShardBlockRootHash32: []byte{byte(i + 100)},
			},
			AggregationBitfield: []byte{0x03},
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
		t.Errorf("Active validators don't match. Wanted:[109,97], Got: %v", attestedValidators)
	}
}

func TestAttestingValidators_EmptyWinningRoot(t *testing.T) {
	state := buildState(params.BeaconConfig().GenesisSlot, params.BeaconConfig().DepositsForChainStart)

	attestation := &pb.PendingAttestation{
		Data: &pb.AttestationData{
			Slot:                 params.BeaconConfig().GenesisSlot,
			ShardBlockRootHash32: []byte{},
		},
		AggregationBitfield: []byte{},
	}

	want := fmt.Sprintf("wanted participants bitfield length %d, got: %d", 16, 0)
	if _, err := AttestingValidators(state, 0, []*pb.PendingAttestation{attestation}, nil); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestTotalAttestingBalance_CorrectBalance(t *testing.T) {
	validatorsPerCommittee := uint64(2)
	state := buildState(params.BeaconConfig().GenesisSlot, 2*params.BeaconConfig().SlotsPerEpoch)

	// Generate 10 roots ([]byte{100}...[]byte{110})
	var attestations []*pb.PendingAttestation
	for i := 0; i < 10; i++ {
		attestation := &pb.PendingAttestation{
			Data: &pb.AttestationData{
				Slot:                 params.BeaconConfig().GenesisSlot,
				ShardBlockRootHash32: []byte{byte(i + 100)},
			},
			// All validators attested to the above roots.
			AggregationBitfield: []byte{0x03},
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

	if attestedBalance != params.BeaconConfig().MaxDepositAmount*validatorsPerCommittee {
		t.Errorf("Incorrect attested balance. Wanted:64*1e9, Got: %d", attestedBalance)
	}
}

func TestTotalAttestingBalance_EmptyWinningRoot(t *testing.T) {
	state := buildState(params.BeaconConfig().GenesisSlot, params.BeaconConfig().DepositsForChainStart)

	attestation := &pb.PendingAttestation{
		Data: &pb.AttestationData{
			Slot:                 params.BeaconConfig().GenesisSlot,
			ShardBlockRootHash32: []byte{},
		},
		AggregationBitfield: []byte{},
	}

	want := fmt.Sprintf("wanted participants bitfield length %d, got: %d", 16, 0)
	if _, err := TotalAttestingBalance(state, 0, []*pb.PendingAttestation{attestation}, nil); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestTotalBalance_CorrectBalance(t *testing.T) {
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

func TestInclusionSlot_GetsCorrectSlot(t *testing.T) {
	state := buildState(params.BeaconConfig().GenesisSlot, params.BeaconConfig().DepositsForChainStart)
	var participationBitfield []byte
	for i := 0; i < 16; i++ {
		participationBitfield = append(participationBitfield, byte(0xff))
	}

	state.LatestAttestations = []*pb.PendingAttestation{
		{Data: &pb.AttestationData{Slot: params.BeaconConfig().GenesisSlot},
			AggregationBitfield: participationBitfield,
			InclusionSlot:       101},
		{Data: &pb.AttestationData{Slot: params.BeaconConfig().GenesisSlot},
			AggregationBitfield: participationBitfield,
			InclusionSlot:       100},
		{Data: &pb.AttestationData{Slot: params.BeaconConfig().GenesisSlot},
			AggregationBitfield: participationBitfield,
			InclusionSlot:       102},
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

func TestInclusionSlot_InvalidBitfield(t *testing.T) {
	state := buildState(params.BeaconConfig().GenesisSlot, params.BeaconConfig().DepositsForChainStart)

	state.LatestAttestations = []*pb.PendingAttestation{
		{Data: &pb.AttestationData{Slot: params.BeaconConfig().GenesisSlot},
			AggregationBitfield: []byte{},
			InclusionSlot:       100},
	}

	want := fmt.Sprintf("wanted participants bitfield length %d, got: %d", 16, 0)
	if _, err := InclusionSlot(state, 0); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestInclusionSlot_SlotNotFound(t *testing.T) {
	state := buildState(params.BeaconConfig().GenesisSlot, params.BeaconConfig().SlotsPerEpoch)

	badIndex := uint64(10000)
	want := fmt.Sprintf("could not find inclusion slot for validator index %d", badIndex)
	if _, err := InclusionSlot(state, badIndex); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestInclusionDistance_CorrectDistance(t *testing.T) {
	state := buildState(params.BeaconConfig().GenesisSlot, params.BeaconConfig().DepositsForChainStart)
	var participationBitfield []byte
	for i := 0; i < 16; i++ {
		participationBitfield = append(participationBitfield, byte(0xff))
	}

	state.LatestAttestations = []*pb.PendingAttestation{
		{Data: &pb.AttestationData{Slot: params.BeaconConfig().GenesisSlot},
			AggregationBitfield: participationBitfield,
			InclusionSlot:       params.BeaconConfig().GenesisSlot + 100},
	}
	distance, err := InclusionDistance(state, 237)
	if err != nil {
		t.Fatalf("Could not execute InclusionDistance: %v", err)
	}

	fmt.Println(state.LatestAttestations[0].InclusionSlot)
	// Inclusion distance is 100 because input validator index is 45,
	// validator 45's attested slot 0 and got included slot 100.
	if distance != 100 {
		t.Errorf("Incorrect distance. Wanted: %d, got: %d",
			100, distance)
	}
}

func TestInclusionDistance_InvalidBitfield(t *testing.T) {
	state := buildState(params.BeaconConfig().GenesisSlot, params.BeaconConfig().DepositsForChainStart)

	state.LatestAttestations = []*pb.PendingAttestation{
		{Data: &pb.AttestationData{Slot: params.BeaconConfig().GenesisSlot},
			AggregationBitfield: []byte{},
			InclusionSlot:       100},
	}

	want := fmt.Sprintf("wanted participants bitfield length %d, got: %d", 16, 0)
	if _, err := InclusionDistance(state, 0); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestInclusionDistance_NotFound(t *testing.T) {
	state := buildState(0, params.BeaconConfig().SlotsPerEpoch)

	badIndex := uint64(10000)
	want := fmt.Sprintf("could not find inclusion distance for validator index %d", badIndex)
	if _, err := InclusionDistance(state, badIndex); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}
