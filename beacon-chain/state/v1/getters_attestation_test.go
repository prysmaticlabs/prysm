package v1

import (
	"testing"

	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestBeaconState_PreviousEpochAttestations(t *testing.T) {
	s, err := InitializeFromProto(&pbp2p.BeaconState{})
	require.NoError(t, err)
	atts, err := s.PreviousEpochAttestations()
	require.DeepEqual(t, []*pbp2p.PendingAttestation(nil), atts)

	want := []*pbp2p.PendingAttestation{{ProposerIndex: 100}}
	s, err = InitializeFromProto(&pbp2p.BeaconState{PreviousEpochAttestations: want})
	require.NoError(t, err)
	got, err := s.PreviousEpochAttestations()
	require.NoError(t, err)
	require.DeepEqual(t, want, got)

	// Test copy does not mutate.
	want = []*pbp2p.PendingAttestation{{ProposerIndex: 101}}
	require.DeepNotEqual(t, want, got)
}

func TestBeaconState_CurrentEpochAttestations(t *testing.T) {
	s, err := InitializeFromProto(&pbp2p.BeaconState{})
	require.NoError(t, err)
	atts, err := s.CurrentEpochAttestations()
	require.DeepEqual(t, []*pbp2p.PendingAttestation(nil), atts)

	want := []*pbp2p.PendingAttestation{{ProposerIndex: 101}}
	s, err = InitializeFromProto(&pbp2p.BeaconState{CurrentEpochAttestations: want})
	require.NoError(t, err)
	got, err := s.CurrentEpochAttestations()
	require.NoError(t, err)
	require.DeepEqual(t, want, got)

	// Test copy does not mutate.
	want = []*pbp2p.PendingAttestation{{ProposerIndex: 102}}
	require.DeepNotEqual(t, want, got)
}
