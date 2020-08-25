package slashingprotection

import (
	"context"
	"testing"

	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
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
	assert.Equal(t, false, s.CheckAttestationSafety(context.Background(), att), "Expected verify attestation to fail verification")
	s = &Service{slasherClient: mockSlasher.MockSlasher{SlashAttestation: false}}
	assert.Equal(t, true, s.CheckAttestationSafety(context.Background(), att), "Expected verify attestation to pass verification")
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
	assert.Equal(t, false, s.CommitAttestation(context.Background(), att), "Expected commit attestation to fail verification")
	s = &Service{slasherClient: mockSlasher.MockSlasher{SlashAttestation: false}}
	assert.Equal(t, true, s.CommitAttestation(context.Background(), att), "Expected commit attestation to pass verification")
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
	assert.Equal(t, false, s.CommitBlock(context.Background(), blk), "Expected commit block to fail verification")
	s = &Service{slasherClient: mockSlasher.MockSlasher{SlashBlock: false}}
	assert.Equal(t, true, s.CommitBlock(context.Background(), blk), "Expected commit block to pass verification")
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
	assert.Equal(t, false, s.CheckBlockSafety(context.Background(), blk), "Expected verify block to fail verification")
	s = &Service{slasherClient: mockSlasher.MockSlasher{SlashBlock: false}}
	assert.Equal(t, true, s.CheckBlockSafety(context.Background(), blk), "Expected verify block to pass verification")
}
