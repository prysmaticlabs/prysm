package state

import (
	"bytes"
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestEpochAttestations_ok(t *testing.T) {
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

		if EpochAttestations(state)[0].Data.Slot != tt.firstAttestationSlot {
			t.Errorf(
				"Result slot was an unexpected value. Wanted %d, got %d",
				tt.firstAttestationSlot,
				EpochAttestations(state)[0].Data.Slot,
			)
		}
	}
}

func TestEpochBoundaryAttestations(t *testing.T) {
	if params.BeaconConfig().EpochLength != 64 {
		t.Errorf("EpochLength should be 64 for these tests to pass")
	}

	epochAttestations := []*pb.PendingAttestationRecord{
		{Data: &pb.AttestationData{JustifiedBlockHash32: []byte{0}, JustifiedSlot: 0}},
		{Data: &pb.AttestationData{JustifiedBlockHash32: []byte{1}, JustifiedSlot: 1}},
		{Data: &pb.AttestationData{JustifiedBlockHash32: []byte{2}, JustifiedSlot: 2}},
		{Data: &pb.AttestationData{JustifiedBlockHash32: []byte{3}, JustifiedSlot: 3}},
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

	epochBoundaryAttestation, err := EpochBoundaryAttestations(state, epochAttestations)
	if err == nil {
		t.Fatalf("EpochBoundaryAttestations should have failed with empty block root hash")
	}

	state.LatestBlockRootHash32S = latestBlockRootHash
	epochBoundaryAttestation, err = EpochBoundaryAttestations(state, epochAttestations)
	if err != nil {
		t.Fatalf("EpochBoundaryAttestations failed: %v", err)
	}

	if epochBoundaryAttestation[0].Data.JustifiedSlot != 0 {
		t.Errorf("Wanted justified slot 0 for epoch boundary attestation, got: %d", epochBoundaryAttestation[0].Data.JustifiedSlot)
	}

	if !bytes.Equal(epochBoundaryAttestation[0].Data.JustifiedBlockHash32, []byte{0}) {
		t.Errorf("Wanted justified block hash [0] for epoch boundary attestation, got: %v", epochBoundaryAttestation[0].Data.JustifiedBlockHash32)
	}
}
