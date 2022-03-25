package v1

import (
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestBeaconState_RotateAttestations(t *testing.T) {
	st, err := InitializeFromProto(&ethpb.BeaconState{
		Slot:                      1,
		CurrentEpochAttestations:  []*ethpb.PendingAttestation{{Data: &ethpb.AttestationData{Slot: 456}}},
		PreviousEpochAttestations: []*ethpb.PendingAttestation{{Data: &ethpb.AttestationData{Slot: 123}}},
	})
	require.NoError(t, err)

	require.NoError(t, st.RotateAttestations())
	currEpochAtts, err := st.CurrentEpochAttestations()
	require.NoError(t, err)
	require.Equal(t, 0, len(currEpochAtts))
	prevEpochAtts, err := st.PreviousEpochAttestations()
	require.NoError(t, err)
	require.Equal(t, types.Slot(456), prevEpochAtts[0].Data.Slot)
}
