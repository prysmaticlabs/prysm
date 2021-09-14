package v1

import (
	"testing"

	"github.com/prysmaticlabs/prysm/encoding/bytes"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	v1alpha1 "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestBeaconState_LatestBlockHeader(t *testing.T) {
	s, err := InitializeFromProto(&ethpb.BeaconState{})
	require.NoError(t, err)
	got := s.LatestBlockHeader()
	require.DeepEqual(t, (*v1alpha1.BeaconBlockHeader)(nil), got)

	want := &v1alpha1.BeaconBlockHeader{Slot: 100}
	s, err = InitializeFromProto(&ethpb.BeaconState{LatestBlockHeader: want})
	require.NoError(t, err)
	got = s.LatestBlockHeader()
	require.DeepEqual(t, want, got)

	// Test copy does not mutate.
	got.Slot = 101
	require.DeepNotEqual(t, want, got)
}

func TestBeaconState_BlockRoots(t *testing.T) {
	s, err := InitializeFromProto(&ethpb.BeaconState{})
	require.NoError(t, err)
	got := s.BlockRoots()
	require.DeepEqual(t, ([][]byte)(nil), got)

	want := [][]byte{{'a'}}
	s, err = InitializeFromProto(&ethpb.BeaconState{BlockRoots: want})
	require.NoError(t, err)
	got = s.BlockRoots()
	require.DeepEqual(t, want, got)

	// Test copy does not mutate.
	got[0][0] = 'b'
	require.DeepNotEqual(t, want, got)
}

func TestBeaconState_BlockRootAtIndex(t *testing.T) {
	s, err := InitializeFromProto(&ethpb.BeaconState{})
	require.NoError(t, err)
	got, err := s.BlockRootAtIndex(0)
	require.NoError(t, err)
	require.DeepEqual(t, ([]byte)(nil), got)

	r := [][]byte{{'a'}}
	s, err = InitializeFromProto(&ethpb.BeaconState{BlockRoots: r})
	require.NoError(t, err)
	got, err = s.BlockRootAtIndex(0)
	require.NoError(t, err)
	want := bytes.PadTo([]byte{'a'}, 32)
	require.DeepSSZEqual(t, want, got)
}
