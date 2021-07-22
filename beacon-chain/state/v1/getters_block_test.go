package v1

import (
	"testing"

	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	v1alpha1 "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestBeaconState_LatestBlockHeader(t *testing.T) {
	s, err := InitializeFromProto(&pbp2p.BeaconState{})
	require.NoError(t, err)
	got := s.LatestBlockHeader()
	require.DeepEqual(t, (*v1alpha1.BeaconBlockHeader)(nil), got)

	want := &v1alpha1.BeaconBlockHeader{Slot: 100}
	s, err = InitializeFromProto(&pbp2p.BeaconState{LatestBlockHeader: want})
	require.NoError(t, err)
	got = s.LatestBlockHeader()
	require.DeepEqual(t, want, got)

	// Test copy does not mutate.
	got = &v1alpha1.BeaconBlockHeader{Slot: 101}
	require.DeepNotEqual(t, want, got)
}

func TestBeaconState_BlockRoots(t *testing.T) {
	s, err := InitializeFromProto(&pbp2p.BeaconState{})
	require.NoError(t, err)
	got := s.BlockRoots()
	require.DeepEqual(t, ([][]byte)(nil), got)

	want := [][]byte{{'a'}}
	s, err = InitializeFromProto(&pbp2p.BeaconState{BlockRoots: want})
	require.NoError(t, err)
	got = s.BlockRoots()
	require.DeepEqual(t, want, got)

	// Test copy does not mutate.
	got = [][]byte{{'b'}}
	require.DeepNotEqual(t, want, got)
}

func TestBeaconState_BlockRootAtIndex(t *testing.T) {
	s, err := InitializeFromProto(&pbp2p.BeaconState{})
	require.NoError(t, err)
	got, err := s.BlockRootAtIndex(0)
	require.NoError(t, err)
	require.DeepEqual(t, ([]byte)(nil), got)

	r := [][]byte{{'a'}}
	s, err = InitializeFromProto(&pbp2p.BeaconState{BlockRoots: r})
	require.NoError(t, err)
	got, err = s.BlockRootAtIndex(0)
	require.NoError(t, err)
	want := bytesutil.PadTo([]byte{'a'}, 32)
	require.DeepSSZEqual(t, want, got)
}
