package slashingprotection

import (
	"context"
	"testing"

	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	mockSlasher "github.com/prysmaticlabs/prysm/validator/testing"
)

var _ = Protector(&RemoteProtector{})

func TestRemoteProtector_IsSlashableAttestation(t *testing.T) {
	s := &RemoteProtector{slasherClient: mockSlasher.MockSlasher{SlashAttestation: true}}
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
	assert.Equal(t, false, s.IsSlashableAttestation(context.Background(), att), "Expected verify attestation to fail verification")
	s = &RemoteProtector{slasherClient: mockSlasher.MockSlasher{SlashAttestation: false}}
	assert.Equal(t, true, s.IsSlashableAttestation(context.Background(), att), "Expected verify attestation to pass verification")
}

func TestRemoteProtector_IsSlashableBlock(t *testing.T) {
	s := &RemoteProtector{slasherClient: mockSlasher.MockSlasher{SlashBlock: true}}
	blk := &eth.SignedBeaconBlock{
		Block: &eth.BeaconBlock{
			Slot:          0,
			ProposerIndex: 0,
			ParentRoot:    bytesutil.PadTo([]byte("parent"), 32),
			StateRoot:     bytesutil.PadTo([]byte("state"), 32),
			Body:          &eth.BeaconBlockBody{},
		},
	}
	assert.Equal(t, false, s.IsSlashableBlock(context.Background(), blk), "Expected verify block to fail verification")
	s = &RemoteProtector{slasherClient: mockSlasher.MockSlasher{SlashBlock: false}}
	assert.Equal(t, true, s.IsSlashableBlock(context.Background(), blk), "Expected verify block to pass verification")
}
