package v2

import (
	"testing"

	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestBeaconState_LatestBlockHeader(t *testing.T) {
	s, err := InitializeFromProto(&ethpb.BeaconStateAltair{})
	require.NoError(t, err)
	got := s.LatestBlockHeader()
	require.DeepEqual(t, (*ethpb.BeaconBlockHeader)(nil), got)

	want := &ethpb.BeaconBlockHeader{Slot: 100}
	s, err = InitializeFromProto(&ethpb.BeaconStateAltair{LatestBlockHeader: want})
	require.NoError(t, err)
	got = s.LatestBlockHeader()
	require.DeepEqual(t, want, got)

	// Test copy does not mutate.
	got.Slot = 101
	require.DeepNotEqual(t, want, got)
}

func TestBeaconState_BlockRoots(t *testing.T) {
	s, err := InitializeFromProto(&ethpb.BeaconStateAltair{})
	require.NoError(t, err)
	got := s.BlockRoots()
	require.DeepEqual(t, [fieldparams.BlockRootsLength][32]byte{}, *got)

	want := [fieldparams.BlockRootsLength][32]byte{{'a'}}
	bRoots := make([][]byte, len(want))
	for i, r := range want {
		tmp := r
		bRoots[i] = tmp[:]
	}
	s, err = InitializeFromProto(&ethpb.BeaconStateAltair{BlockRoots: bRoots})
	require.NoError(t, err)
	got = s.BlockRoots()
	require.DeepEqual(t, want, *got)

	// Test copy does not mutate.
	got[0][0] = 'b'
	require.DeepNotEqual(t, want, *got)
}

func TestBeaconState_BlockRootAtIndex(t *testing.T) {
	s, err := InitializeFromProto(&ethpb.BeaconStateAltair{})
	require.NoError(t, err)
	got, err := s.BlockRootAtIndex(0)
	require.NoError(t, err)
	require.DeepEqual(t, [32]byte{}, got)

	r := [fieldparams.BlockRootsLength][32]byte{{'a'}}
	bRoots := make([][]byte, len(r))
	for i, root := range r {
		tmp := root
		bRoots[i] = tmp[:]
	}
	s, err = InitializeFromProto(&ethpb.BeaconStateAltair{BlockRoots: bRoots})
	require.NoError(t, err)
	got, err = s.BlockRootAtIndex(0)
	require.NoError(t, err)
	want := [fieldparams.RootLength]byte{'a'}
	require.DeepSSZEqual(t, want, got)
}
