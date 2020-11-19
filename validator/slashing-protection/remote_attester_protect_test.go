package slashingprotection

import (
	"context"
	"testing"

	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestRemoteProtector_IsSlashableAttestation(t *testing.T) {
	s := &RemoteProtector{slasherClient: mockSlasher{slashAttestation: true}}
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
	ctx := context.Background()
	slashable, err := s.IsSlashableAttestation(ctx, att, [48]byte{}, [32]byte{})
	require.NoError(t, err)
	assert.Equal(t, true, slashable, "Expected attestation to be slashable")
	s = &RemoteProtector{slasherClient: mockSlasher{slashAttestation: false}}
	slashable, err = s.IsSlashableAttestation(ctx, att, [48]byte{}, [32]byte{})
	require.NoError(t, err)
	assert.Equal(t, false, slashable, "Expected attestation to not be slashable")
}
