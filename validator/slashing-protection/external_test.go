package slashingprotection

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	eth "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/assert"
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

func TestService_VerifyBlock(t *testing.T) {
	s := &Service{slasherClient: mockSlasher.MockSlasher{SlashBlock: true}}
	blk := &eth.BeaconBlockHeader{
		Slot:          0,
		ProposerIndex: 0,
		ParentRoot:    bytesutil.PadTo([]byte("parent"), 32),
		StateRoot:     bytesutil.PadTo([]byte("state"), 32),
		BodyRoot:      bytesutil.PadTo([]byte("body"), 32),
	}
	sblk := &eth.SignedBeaconBlockHeader{Header: blk, Signature: params.BeaconConfig().EmptySignature[:]}
	assert.Equal(t, false, s.CheckBlockSafety(context.Background(), sblk), "Expected verify block to fail verification")
	s = &Service{slasherClient: mockSlasher.MockSlasher{SlashBlock: false}}
	assert.Equal(t, true, s.CheckBlockSafety(context.Background(), sblk), "Expected verify block to pass verification")
}
