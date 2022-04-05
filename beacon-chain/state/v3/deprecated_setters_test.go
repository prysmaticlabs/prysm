package v3

import (
	"testing"

	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestBeaconState_AppendCurrentEpochAttestations(t *testing.T) {
	s, err := InitializeFromProtoUnsafe(&ethpb.BeaconStateBellatrix{})
	require.NoError(t, err)
	require.ErrorContains(t, "AppendCurrentEpochAttestations is not supported for version Bellatrix beacon state", s.AppendCurrentEpochAttestations(nil))
}

func TestBeaconState_AppendPreviousEpochAttestations(t *testing.T) {
	s, err := InitializeFromProtoUnsafe(&ethpb.BeaconStateBellatrix{})
	require.NoError(t, err)
	require.ErrorContains(t, "AppendPreviousEpochAttestations is not supported for version Bellatrix beacon state", s.AppendPreviousEpochAttestations(nil))
}
