package epoch

import (
	"bytes"
	"reflect"
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestEpochAttestations(t *testing.T) {
	if params.BeaconConfig().EpochLength != 64 {
		t.Errorf("EpochLength should be 64 for these tests to pass")
	}

	var pendingAttestations []*pb.PendingAttestationRecord
	for i := uint64(0); i < params.BeaconConfig().EpochLength*2; i++ {
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
			firstAttestationSlot: 0,
		},
		{
			stateSlot:            63,
			firstAttestationSlot: 0,
		},
		{
			stateSlot:            64,
			firstAttestationSlot: 64 - params.BeaconConfig().EpochLength,
		}, {
			stateSlot:            127,
			firstAttestationSlot: 127 - params.BeaconConfig().EpochLength,
		}, {
			stateSlot:            128,
			firstAttestationSlot: 128 - params.BeaconConfig().EpochLength,
		},
	}

	for _, tt := range tests {
		state.Slot = tt.stateSlot

		if Attestations(state)[0].Data.Slot != tt.firstAttestationSlot {
			t.Errorf(
				"Result slot was an unexpected value. Wanted %d, got %d",
				tt.firstAttestationSlot,
				Attestations(state)[0].Data.Slot,
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

	if _, err := BoundaryAttestations(state, epochAttestations); err == nil {
		t.Fatal("EpochBoundaryAttestations should have failed with empty block root hash")
	}

	state.Slot = params.BeaconConfig().EpochLength
	epochBoundaryAttestation, err := BoundaryAttestations(state, epochAttestations)
	if err != nil {
		t.Fatalf("EpochBoundaryAttestations failed: %v", err)
	}

	if epochBoundaryAttestation[0].Data.JustifiedSlot != 0 {
		t.Errorf("Wanted justified slot 0 for epoch boundary attestation, got: %d", epochBoundaryAttestation[0].Data.JustifiedSlot)
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
	for i := uint64(0); i < params.BeaconConfig().EpochLength*4; i++ {
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
			firstAttestationSlot: 0,
		},
		{
			stateSlot:            128,
			firstAttestationSlot: 0,
		},
		{
			stateSlot:            383,
			firstAttestationSlot: 383 - 2*params.BeaconConfig().EpochLength,
		},
		{
			stateSlot:            129,
			firstAttestationSlot: 129 - 2*params.BeaconConfig().EpochLength,
		},
		{
			stateSlot:            256,
			firstAttestationSlot: 256 - 2*params.BeaconConfig().EpochLength,
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
		{Data: &pb.AttestationData{Shard: 0, JustifiedSlot: 100}},
		{Data: &pb.AttestationData{Shard: 1, JustifiedSlot: 100}},
		{Data: &pb.AttestationData{JustifiedSlot: 888}},
	}

	state := &pb.BeaconState{PreviousJustifiedSlot: 100}

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

func TestHeadAttestations_Ok(t *testing.T) {
	if params.BeaconConfig().EpochLength != 64 {
		t.Errorf("EpochLength should be 64 for these tests to pass")
	}

	prevAttestations := []*pb.PendingAttestationRecord{
		{Data: &pb.AttestationData{Slot: 1, BeaconBlockRootHash32: []byte{'A'}}},
		{Data: &pb.AttestationData{Slot: 2, BeaconBlockRootHash32: []byte{'B'}}},
		{Data: &pb.AttestationData{Slot: 3, BeaconBlockRootHash32: []byte{'C'}}},
		{Data: &pb.AttestationData{Slot: 4, BeaconBlockRootHash32: []byte{'D'}}},
	}

	state := &pb.BeaconState{Slot: 5, LatestBlockRootHash32S: [][]byte{{'A'}, {'X'}, {'C'}, {'Y'}}}

	headAttestations, err := PrevHeadAttestations(state, prevAttestations)
	if err != nil {
		t.Fatalf("PrevHeadAttestations failed with %v", err)
	}

	if headAttestations[0].Data.Slot != 1 {
		t.Errorf("headAttestations[0] wanted slot 1, got slot %d", headAttestations[0].Data.Slot)
	}
	if headAttestations[1].Data.Slot != 3 {
		t.Errorf("headAttestations[1] wanted slot 3, got slot %d", headAttestations[1].Data.Slot)
	}
	if !bytes.Equal([]byte{'A'}, headAttestations[0].Data.BeaconBlockRootHash32) {
		t.Errorf("headAttestations[0] wanted hash [A], got slot %v",
			headAttestations[0].Data.BeaconBlockRootHash32)
	}
	if !bytes.Equal([]byte{'C'}, headAttestations[1].Data.BeaconBlockRootHash32) {
		t.Errorf("headAttestations[1] wanted hash [C], got slot %v",
			headAttestations[1].Data.BeaconBlockRootHash32)
	}
}

func TestHeadAttestations_NotOk(t *testing.T) {
	if params.BeaconConfig().EpochLength != 64 {
		t.Errorf("EpochLength should be 64 for these tests to pass")
	}

	prevAttestations := []*pb.PendingAttestationRecord{{Data: &pb.AttestationData{Slot: 1}}}

	state := &pb.BeaconState{Slot: 0}

	if _, err := PrevHeadAttestations(state, prevAttestations); err == nil {
		t.Fatal("PrevHeadAttestations should have failed with invalid range")
	}
}

func TestWinningRoot_Ok(t *testing.T) {
	defaultBalance := params.BeaconConfig().MaxDeposit

	ShardCommittees := []*pb.ShardCommitteeArray{
		{ArrayShardCommittee: []*pb.ShardCommittee{
			{Shard: 1, Committee: []uint32{0, 1, 2, 3, 4, 5, 6, 7}},
		}}}

	// Assign 32 ETH balance to every validator in ShardCommittees.
	state := &pb.BeaconState{
		ShardCommitteesAtSlots: ShardCommittees,
		Slot:                   5,
		ValidatorBalances: []uint64{
			defaultBalance, defaultBalance, defaultBalance, defaultBalance,
			defaultBalance, defaultBalance, defaultBalance, defaultBalance,
		},
	}

	// Generate 10 roots ([]byte{100}...[]byte{110})
	var attestations []*pb.PendingAttestationRecord
	for i := 0; i < 10; i++ {
		attestation := &pb.PendingAttestationRecord{
			Data: &pb.AttestationData{
				Slot:                 0,
				Shard:                1,
				ShardBlockRootHash32: []byte{byte(i + 100)},
			},
			// Validator 1 and 7 attested to all 10 roots.
			ParticipationBitfield: []byte{'A'},
		}
		attestations = append(attestations, attestation)
	}

	// Since all 10 roots have the balance of 64 ETHs
	// winningRoot chooses the lowest hash: []byte{100}
	winnerRoot, err := winningRoot(
		state,
		ShardCommittees[0].ArrayShardCommittee[0],
		attestations,
		nil)
	if err != nil {
		t.Fatalf("Could not execute winningRoot: %v", err)
	}
	if !bytes.Equal(winnerRoot, []byte{100}) {
		t.Errorf("Incorrect winner root, wanted:[100], got: %v", winnerRoot)
	}

	// Give root [105] one more attester
	attestations[5].ParticipationBitfield = []byte{'C'}
	winnerRoot, err = winningRoot(
		state,
		ShardCommittees[0].ArrayShardCommittee[0],
		attestations,
		nil)
	if err != nil {
		t.Fatalf("Could not execute winningRoot: %v", err)
	}
	if !bytes.Equal(winnerRoot, []byte{105}) {
		t.Errorf("Incorrect winner root, wanted:[105], got: %v", winnerRoot)
	}
}

func TestWinningRoot_OutOfBound(t *testing.T) {
	ShardCommittees := []*pb.ShardCommitteeArray{
		{ArrayShardCommittee: []*pb.ShardCommittee{
			{Shard: 1, Committee: []uint32{}},
		}}}

	state := &pb.BeaconState{
		ShardCommitteesAtSlots: ShardCommittees,
	}

	attestation := &pb.PendingAttestationRecord{
		Data: &pb.AttestationData{
			Shard:                1,
			ShardBlockRootHash32: []byte{},
		},
		ParticipationBitfield: []byte{'A'},
	}

	_, err := winningRoot(
		state,
		ShardCommittees[0].ArrayShardCommittee[0],
		[]*pb.PendingAttestationRecord{attestation},
		nil)
	if err == nil {
		t.Fatal("winningRoot should have failed")
	}
}

func TestAttestingValidators_Ok(t *testing.T) {
	defaultBalance := params.BeaconConfig().MaxDeposit

	ShardCommittees := []*pb.ShardCommitteeArray{
		{ArrayShardCommittee: []*pb.ShardCommittee{
			{Shard: 1, Committee: []uint32{0, 1, 2, 3, 4, 5, 6, 7}},
		}}}

	// Assign 32 ETH balance to every validator in ShardCommittees.
	state := &pb.BeaconState{
		ShardCommitteesAtSlots: ShardCommittees,
		Slot:                   5,
		ValidatorBalances: []uint64{
			defaultBalance, defaultBalance, defaultBalance, defaultBalance,
			defaultBalance, defaultBalance, defaultBalance, defaultBalance,
		},
	}

	// Generate 10 roots ([]byte{100}...[]byte{110})
	var attestations []*pb.PendingAttestationRecord
	for i := 0; i < 10; i++ {
		attestation := &pb.PendingAttestationRecord{
			Data: &pb.AttestationData{
				Slot:                 0,
				Shard:                1,
				ShardBlockRootHash32: []byte{byte(i + 100)},
			},
			// Validator 1 and 7 attested to the above roots.
			ParticipationBitfield: []byte{'A'},
		}
		attestations = append(attestations, attestation)
	}

	attestedValidators, err := AttestingValidators(
		state,
		ShardCommittees[0].ArrayShardCommittee[0],
		attestations,
		nil)
	if err != nil {
		t.Fatalf("Could not execute winningRoot: %v", err)
	}

	// Verify the winner root is attested by validator 1 and 7.
	if !reflect.DeepEqual(attestedValidators, []uint32{1, 7}) {
		t.Errorf("Active validators don't match. Wanted:[1,7], Got: %v", attestedValidators)
	}
}

func TestAttestingValidators_CantGetWinningRoot(t *testing.T) {
	ShardCommittees := []*pb.ShardCommitteeArray{
		{ArrayShardCommittee: []*pb.ShardCommittee{
			{Shard: 1, Committee: []uint32{}},
		}}}

	state := &pb.BeaconState{
		ShardCommitteesAtSlots: ShardCommittees,
	}

	attestation := &pb.PendingAttestationRecord{
		Data: &pb.AttestationData{
			Shard:                1,
			ShardBlockRootHash32: []byte{},
		},
		ParticipationBitfield: []byte{'A'},
	}

	_, err := AttestingValidators(
		state,
		ShardCommittees[0].ArrayShardCommittee[0],
		[]*pb.PendingAttestationRecord{attestation},
		nil)
	if err == nil {
		t.Fatal("attestingValidators should have failed")
	}
}

func TestTotalAttestingBalance_Ok(t *testing.T) {

	ShardCommittees := []*pb.ShardCommitteeArray{
		{ArrayShardCommittee: []*pb.ShardCommittee{
			{Shard: 1, Committee: []uint32{0, 1, 2, 3, 4, 5, 6, 7}},
		}}}

	// Assign validators to different balances.
	state := &pb.BeaconState{
		ShardCommitteesAtSlots: ShardCommittees,
		Slot:                   5,
		ValidatorBalances: []uint64{16 * 1e9, 18 * 1e9, 20 * 1e9, 31 * 1e9,
			32 * 1e9, 34 * 1e9, 50 * 1e9, 50 * 1e9},
	}

	// Generate 10 roots ([]byte{100}...[]byte{110})
	var attestations []*pb.PendingAttestationRecord
	for i := 0; i < 10; i++ {
		attestation := &pb.PendingAttestationRecord{
			Data: &pb.AttestationData{
				Slot:                 0,
				Shard:                1,
				ShardBlockRootHash32: []byte{byte(i + 100)},
			},
			// All validators attested to the above roots.
			ParticipationBitfield: []byte{0xff},
		}
		attestations = append(attestations, attestation)
	}

	attestedBalance, err := TotalAttestingBalance(
		state,
		ShardCommittees[0].ArrayShardCommittee[0],
		attestations,
		nil)
	if err != nil {
		t.Fatalf("Could not execute totalAttestingBalance: %v", err)
	}

	// Verify the Attested balances are 16+18+20+31+(32*4)=213.
	if attestedBalance != 213*1e9 {
		t.Errorf("Incorrect attested balance. Wanted:231*1e9, Got: %d", attestedBalance)
	}
}

func TestTotalAttestingBalance_NotOfBound(t *testing.T) {

	ShardCommittees := []*pb.ShardCommitteeArray{
		{ArrayShardCommittee: []*pb.ShardCommittee{
			{Shard: 1, Committee: []uint32{}},
		}}}

	state := &pb.BeaconState{
		ShardCommitteesAtSlots: ShardCommittees,
	}

	attestation := &pb.PendingAttestationRecord{
		Data: &pb.AttestationData{
			Shard:                1,
			ShardBlockRootHash32: []byte{},
		},
		ParticipationBitfield: []byte{'A'},
	}

	_, err := TotalAttestingBalance(
		state,
		ShardCommittees[0].ArrayShardCommittee[0],
		[]*pb.PendingAttestationRecord{attestation},
		nil)
	if err == nil {
		t.Fatal("totalAttestingBalance should have failed")
	}
}

func TestTotalBalance(t *testing.T) {

	ShardCommittees := &pb.ShardCommittee{Shard: 1, Committee: []uint32{0, 1, 2, 3, 4, 5, 6, 7}}

	// Assign validators to different balances.
	state := &pb.BeaconState{
		Slot: 5,
		ValidatorBalances: []uint64{20 * 1e9, 25 * 1e9, 30 * 1e9, 30 * 1e9,
			32 * 1e9, 34 * 1e9, 50 * 1e9, 50 * 1e9},
	}

	// 20 + 25 + 30 + 30 + 32 + 32 + 32 + 32 = 233
	totalBalance := TotalBalance(state, ShardCommittees.Committee)
	if totalBalance != 233*1e9 {
		t.Errorf("Incorrect total balance. Wanted: 233*1e9, got: %d", totalBalance)
	}
}

func TestInclusionSlot_Ok(t *testing.T) {

	ShardCommittees := []*pb.ShardCommitteeArray{
		{ArrayShardCommittee: []*pb.ShardCommittee{
			{Shard: 1, Committee: []uint32{0, 1, 2, 3, 4, 5, 6, 7}},
		}}}

	state := &pb.BeaconState{
		ShardCommitteesAtSlots: ShardCommittees,
		Slot:                   5,
		LatestAttestations: []*pb.PendingAttestationRecord{
			{Data: &pb.AttestationData{Shard: 1, Slot: 0},
				// Validator 1 and 7 participated.
				ParticipationBitfield: []byte{'A'},
				SlotIncluded:          100},
		},
	}

	slot, err := InclusionSlot(state, 1)
	if err != nil {
		t.Fatalf("Could not execute InclusionSlot: %v", err)
	}

	// validator 7's attestation got included in slot 100.
	if slot != 100 {
		t.Errorf("Incorrect slot. Wanted: 100, got: %d", slot)
	}
}

func TestInclusionSlot_BadBitfield(t *testing.T) {

	ShardCommittees := []*pb.ShardCommitteeArray{
		{ArrayShardCommittee: []*pb.ShardCommittee{
			{Shard: 1, Committee: []uint32{1}},
		}}}

	state := &pb.BeaconState{
		ShardCommitteesAtSlots: ShardCommittees,
		LatestAttestations: []*pb.PendingAttestationRecord{
			{Data: &pb.AttestationData{Shard: 1, Slot: 0},
				ParticipationBitfield: []byte{},
				SlotIncluded:          9},
		},
	}

	_, err := InclusionSlot(state, 1)
	if err == nil {
		t.Fatal("InclusionSlot should have failed")
	}
}

func TestInclusionSlot_NotFound(t *testing.T) {

	ShardCommittees := []*pb.ShardCommitteeArray{
		{ArrayShardCommittee: []*pb.ShardCommittee{
			{Shard: 1, Committee: []uint32{1}},
		}}}

	state := &pb.BeaconState{
		ShardCommitteesAtSlots: ShardCommittees,
	}

	_, err := InclusionSlot(state, 1)
	if err == nil {
		t.Fatal("InclusionSlot should have failed")
	}
}

func TestInclusionDistance_Ok(t *testing.T) {

	ShardCommittees := []*pb.ShardCommitteeArray{
		{ArrayShardCommittee: []*pb.ShardCommittee{
			{Shard: 1, Committee: []uint32{0, 1, 2, 3, 4, 5, 6, 7}},
		}}}

	state := &pb.BeaconState{
		ShardCommitteesAtSlots: ShardCommittees,
		Slot:                   5,
		LatestAttestations: []*pb.PendingAttestationRecord{
			{Data: &pb.AttestationData{Shard: 1, Slot: 0},
				// Validator 1 and 7 participated.
				ParticipationBitfield: []byte{'A'},
				SlotIncluded:          9},
		},
	}

	distance, err := InclusionDistance(state, 7)
	if err != nil {
		t.Fatalf("Could not execute InclusionDistance: %v", err)
	}

	// Inclusion distance is 9 because input validator index is 7,
	// validator 7's attested slot 0 and got included slot 9.
	if distance != 9 {
		t.Errorf("Incorrect distance. Wanted: 9, got: %d", distance)
	}
}

func TestInclusionDistance_BadBitfield(t *testing.T) {

	ShardCommittees := []*pb.ShardCommitteeArray{
		{ArrayShardCommittee: []*pb.ShardCommittee{
			{Shard: 1, Committee: []uint32{1}},
		}}}

	state := &pb.BeaconState{
		ShardCommitteesAtSlots: ShardCommittees,
		LatestAttestations: []*pb.PendingAttestationRecord{
			{Data: &pb.AttestationData{Shard: 1, Slot: 0},
				ParticipationBitfield: []byte{},
				SlotIncluded:          9},
		},
	}

	_, err := InclusionDistance(state, 1)
	if err == nil {
		t.Fatal("InclusionDistance should have failed")
	}
}

func TestInclusionDistance_NotFound(t *testing.T) {

	ShardCommittees := []*pb.ShardCommitteeArray{
		{ArrayShardCommittee: []*pb.ShardCommittee{
			{Shard: 1, Committee: []uint32{1}},
		}}}

	state := &pb.BeaconState{
		ShardCommitteesAtSlots: ShardCommittees,
	}

	_, err := InclusionDistance(state, 1)
	if err == nil {
		t.Fatal("InclusionDistance should have failed")
	}
}
