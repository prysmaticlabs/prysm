package epoch

import (
	"bytes"
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
		Slot:                   params.BeaconConfig().EpochLength,
		LatestBlockRootHash32S: [][]byte{},
	}

	epochBoundaryAttestation, err := BoundaryAttestations(state, epochAttestations)
	if err == nil {
		t.Fatalf("EpochBoundaryAttestations should have failed with empty block root hash")
	}

	state.LatestBlockRootHash32S = latestBlockRootHash
	epochBoundaryAttestation, err = BoundaryAttestations(state, epochAttestations)
	if err != nil {
		t.Fatalf("EpochBoundaryAttestations failed: %v", err)
	}

	if epochBoundaryAttestation[0].GetData().JustifiedSlot != 0 {
		t.Errorf("Wanted justified slot 0 for epoch boundary attestation, got: %d", epochBoundaryAttestation[0].Data.JustifiedSlot)
	}

	if !bytes.Equal(epochBoundaryAttestation[0].GetData().JustifiedBlockRootHash32, []byte{0}) {
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
			stateSlot:            10,
			firstAttestationSlot: 0,
		},
		{
			stateSlot:            127,
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
				Attestations(state)[0].Data.Slot,
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
