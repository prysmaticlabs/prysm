package v3

import (
	"testing"

	"github.com/prysmaticlabs/prysm/config/features"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestBeaconState_CurrentEpochAttestations(t *testing.T) {
	features.Init(&features.Flags{DisableNativeState: true})
	s, err := InitializeFromProtoUnsafe(&ethpb.BeaconStateBellatrix{})
	require.NoError(t, err)
	_, err = s.CurrentEpochAttestations()
	require.ErrorContains(t, "CurrentEpochAttestations is not supported for version Bellatrix beacon state", err)
}

func TestBeaconState_PreviousEpochAttestations(t *testing.T) {
	features.Init(&features.Flags{DisableNativeState: true})
	s, err := InitializeFromProtoUnsafe(&ethpb.BeaconStateBellatrix{})
	require.NoError(t, err)
	_, err = s.PreviousEpochAttestations()
	require.ErrorContains(t, "PreviousEpochAttestations is not supported for version Bellatrix beacon state", err)
}
