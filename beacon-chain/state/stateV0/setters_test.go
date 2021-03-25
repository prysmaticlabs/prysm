package stateV0

import (
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestBeaconState_RotateAttestations(t *testing.T) {
	st, err := InitializeFromProto(&pb.BeaconState{
		Slot:                      1,
		CurrentEpochAttestations:  []*pb.PendingAttestation{{Data: &eth.AttestationData{Slot: 456}}},
		PreviousEpochAttestations: []*pb.PendingAttestation{{Data: &eth.AttestationData{Slot: 123}}},
	})
	require.NoError(t, err)

	require.NoError(t, st.RotateAttestations())
	require.Equal(t, 0, len(st.CurrentEpochAttestations()))
	require.Equal(t, types.Slot(456), st.PreviousEpochAttestations()[0].Data.Slot)
}
