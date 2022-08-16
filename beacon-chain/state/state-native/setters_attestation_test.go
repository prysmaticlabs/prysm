package state_native

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v3/config/features"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
)

func TestBeaconState_RotateAttestations(t *testing.T) {
	features.Init(&features.Flags{EnableNativeState: true})
	st, err := InitializeFromProtoPhase0(&ethpb.BeaconState{
		Slot:                      1,
		CurrentEpochAttestations:  []*ethpb.PendingAttestation{{Data: &ethpb.AttestationData{Slot: 456}}},
		PreviousEpochAttestations: []*ethpb.PendingAttestation{{Data: &ethpb.AttestationData{Slot: 123}}},
	})
	require.NoError(t, err)

	require.NoError(t, st.RotateAttestations())
	s, ok := st.(*BeaconState)
	require.Equal(t, true, ok)
	require.Equal(t, 0, len(s.currentEpochAttestationsVal()))
	require.Equal(t, types.Slot(456), s.previousEpochAttestationsVal()[0].Data.Slot)
	features.Init(&features.Flags{EnableNativeState: false})
}
