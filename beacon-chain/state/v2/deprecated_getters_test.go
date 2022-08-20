package v2

import (
	"testing"

	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
)

func TestBeaconState_PreviousEpochAttestations(t *testing.T) {
	s, err := InitializeFromProtoUnsafe(&ethpb.BeaconStateAltair{})
	require.NoError(t, err)
	_, err = s.PreviousEpochAttestations()
	require.ErrorContains(t, "PreviousEpochAttestations is not supported for hard fork 1 beacon state", err)
}

func TestBeaconState_CurrentEpochAttestations(t *testing.T) {
	s, err := InitializeFromProtoUnsafe(&ethpb.BeaconStateAltair{})
	require.NoError(t, err)
	_, err = s.CurrentEpochAttestations()
	require.ErrorContains(t, "CurrentEpochAttestations is not supported for hard fork 1 beacon state", err)
}

func TestBeaconState_LatestExecutionPayloadHeader(t *testing.T) {
	s, err := InitializeFromProtoUnsafe(&ethpb.BeaconStateAltair{})
	require.NoError(t, err)
	_, err = s.LatestExecutionPayloadHeader()
	require.ErrorContains(t, "LatestExecutionPayloadHeader is not supported for hard fork 1 beacon state", err)
}
