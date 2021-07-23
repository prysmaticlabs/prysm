package v1

import (
	"testing"

	statepb "github.com/prysmaticlabs/prysm/proto/prysm/v2/state"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestBeaconState_PreviousEpochAttestations(t *testing.T) {
	s, err := InitializeFromProto(&statepb.BeaconState{})
	require.NoError(t, err)
	atts, err := s.PreviousEpochAttestations()
	require.NoError(t, err)
	require.DeepEqual(t, []*statepb.PendingAttestation(nil), atts)

	want := []*statepb.PendingAttestation{{ProposerIndex: 100}}
	s, err = InitializeFromProto(&statepb.BeaconState{PreviousEpochAttestations: want})
	require.NoError(t, err)
	got, err := s.PreviousEpochAttestations()
	require.NoError(t, err)
	require.DeepEqual(t, want, got)

	// Test copy does not mutate.
	got[0].ProposerIndex = 101
	require.DeepNotEqual(t, want, got)
}

func TestBeaconState_CurrentEpochAttestations(t *testing.T) {
	s, err := InitializeFromProto(&statepb.BeaconState{})
	require.NoError(t, err)
	atts, err := s.CurrentEpochAttestations()
	require.NoError(t, err)
	require.DeepEqual(t, []*statepb.PendingAttestation(nil), atts)

	want := []*statepb.PendingAttestation{{ProposerIndex: 101}}
	s, err = InitializeFromProto(&statepb.BeaconState{CurrentEpochAttestations: want})
	require.NoError(t, err)
	got, err := s.CurrentEpochAttestations()
	require.NoError(t, err)
	require.DeepEqual(t, want, got)

	// Test copy does not mutate.
	got[0].ProposerIndex = 102
	require.DeepNotEqual(t, want, got)
}
