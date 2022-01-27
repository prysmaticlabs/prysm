package v3

import (
	"testing"

	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestBeaconState_LatestBlockHeader(t *testing.T) {
	s, err := InitializeFromProto(&ethpb.BeaconStateBellatrix{})
	require.NoError(t, err)
	got := s.LatestBlockHeader()
	require.DeepEqual(t, (*ethpb.BeaconBlockHeader)(nil), got)

	want := &ethpb.BeaconBlockHeader{Slot: 100}
	s, err = InitializeFromProto(&ethpb.BeaconStateBellatrix{LatestBlockHeader: want})
	require.NoError(t, err)
	got = s.LatestBlockHeader()
	require.DeepEqual(t, want, got)

	// Test copy does not mutate.
	got.Slot = 101
	require.DeepNotEqual(t, want, got)
}

func TestBeaconState_BlockRoots(t *testing.T) {
	s, err := InitializeFromProto(&ethpb.BeaconStateBellatrix{})
	require.NoError(t, err)
	got := s.BlockRoots()
	want := make([][]byte, fieldparams.BlockRootsLength)
	for i, _ := range want {
		want[i] = make([]byte, fieldparams.RootLength)
	}
	require.DeepEqual(t, want, got)

	want = make([][]byte, fieldparams.BlockRootsLength)
	for i, _ := range want {
		if i == 0 {
			want[i] = bytesutil.PadTo([]byte{'a'}, fieldparams.RootLength)
		} else {
			want[i] = make([]byte, fieldparams.RootLength)
		}

	}
	s, err = InitializeFromProto(&ethpb.BeaconStateBellatrix{BlockRoots: want})
	require.NoError(t, err)
	got = s.BlockRoots()
	require.DeepEqual(t, want, got)

	// Test copy does not mutate.
	got[0][0] = 'b'
	require.DeepNotEqual(t, want, got)
}

func TestBeaconState_BlockRootAtIndex(t *testing.T) {
	s, err := InitializeFromProto(&ethpb.BeaconStateBellatrix{})
	require.NoError(t, err)
	got, err := s.BlockRootAtIndex(0)
	require.NoError(t, err)
	require.DeepEqual(t, bytesutil.PadTo([]byte{}, fieldparams.RootLength), got)

	r := [fieldparams.BlockRootsLength][fieldparams.RootLength]byte{{'a'}}
	bRoots := make([][]byte, len(r))
	for i, root := range r {
		tmp := root
		bRoots[i] = tmp[:]
	}
	s, err = InitializeFromProto(&ethpb.BeaconStateBellatrix{BlockRoots: bRoots})
	require.NoError(t, err)
	got, err = s.BlockRootAtIndex(0)
	require.NoError(t, err)
	want := bytesutil.PadTo([]byte{'a'}, fieldparams.RootLength)
	require.DeepSSZEqual(t, want, got)
}
