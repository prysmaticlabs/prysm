package slashingprotection

import (
	"context"
	"testing"

	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	mockSlasher "github.com/prysmaticlabs/prysm/validator/testing"
)

func TestService_VerifyAttestation(t *testing.T) {
	s := &Service{slasherClient: mockSlasher.MockSlasher{SlashAttestation: true}}
	att := &eth.IndexedAttestation{
		AttestingIndices: []uint64{1, 2},
		Data: &eth.AttestationData{
			Slot:            5,
			CommitteeIndex:  2,
			BeaconBlockRoot: []byte("great block"),
			Source: &eth.Checkpoint{
				Epoch: 4,
				Root:  []byte("good source"),
			},
			Target: &eth.Checkpoint{
				Epoch: 10,
				Root:  []byte("good target"),
			},
		},
	}
	if s.VerifyAttestation(context.Background(), att) {
		t.Error("Expected verify attestation to fail verification")
	}
	s = &Service{slasherClient: mockSlasher.MockSlasher{SlashAttestation: false}}
	if !s.VerifyAttestation(context.Background(), att) {
		t.Error("Expected verify attestation to pass verification")
	}
}

func TestService_CommitAttestation(t *testing.T) {
	s := &Service{slasherClient: mockSlasher.MockSlasher{SlashAttestation: true}}
	att := &eth.IndexedAttestation{
		AttestingIndices: []uint64{1, 2},
		Data: &eth.AttestationData{
			Slot:            5,
			CommitteeIndex:  2,
			BeaconBlockRoot: []byte("great block"),
			Source: &eth.Checkpoint{
				Epoch: 4,
				Root:  []byte("good source"),
			},
			Target: &eth.Checkpoint{
				Epoch: 10,
				Root:  []byte("good target"),
			},
		},
	}
	if s.CommitAttestation(context.Background(), att) {
		t.Error("Expected commit attestation to fail verification")
	}
	s = &Service{slasherClient: mockSlasher.MockSlasher{SlashAttestation: false}}
	if !s.CommitAttestation(context.Background(), att) {
		t.Error("Expected commit attestation to pass verification")
	}
}

func TestService_CommitBlock(t *testing.T) {
	s := &Service{slasherClient: mockSlasher.MockSlasher{SlashBlock: true}}
	blk := &eth.SignedBeaconBlockHeader{
		Header: &eth.BeaconBlockHeader{
			Slot:          0,
			ProposerIndex: 0,
			ParentRoot:    []byte("parent"),
			StateRoot:     []byte("state"),
			BodyRoot:      []byte("body"),
		},
	}
	if s.CommitBlock(context.Background(), blk) {
		t.Error("Expected commit block to fail verification")
	}
	s = &Service{slasherClient: mockSlasher.MockSlasher{SlashBlock: false}}
	if !s.CommitBlock(context.Background(), blk) {
		t.Error("Expected commit block to pass verification")
	}
}

func TestService_VerifyBlock(t *testing.T) {
	s := &Service{slasherClient: mockSlasher.MockSlasher{SlashBlock: true}}
	blk := &eth.BeaconBlockHeader{
		Slot:          0,
		ProposerIndex: 0,
		ParentRoot:    []byte("parent"),
		StateRoot:     []byte("state"),
		BodyRoot:      []byte("body"),
	}
	if s.VerifyBlock(context.Background(), blk) {
		t.Error("Expected verify block to fail verification")
	}
	s = &Service{slasherClient: mockSlasher.MockSlasher{SlashBlock: false}}
	if !s.VerifyBlock(context.Background(), blk) {
		t.Error("Expected verify block to pass verification")
	}
}
