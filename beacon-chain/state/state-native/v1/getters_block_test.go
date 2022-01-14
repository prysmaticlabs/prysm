package v1

import (
	"testing"

	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestBeaconState_LatestBlockHeader(t *testing.T) {
	s, err := InitializeFromProto(&ethpb.BeaconState{})
	require.NoError(t, err)
	got := s.LatestBlockHeader()
	require.DeepEqual(t, (*ethpb.BeaconBlockHeader)(nil), got)

	want := &ethpb.BeaconBlockHeader{Slot: 100}
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
	want := make([][]byte, fieldparams.BlockRootsLength)
	for i, _ := range want {
		want[i] = make([]byte, 32)
	}
	require.DeepEqual(t, want, got)

	want = make([][]byte, fieldparams.BlockRootsLength)
	for i, _ := range want {
		if i == 0 {
			want[i] = bytesutil.PadTo([]byte{'a'}, 32)
		} else {
			want[i] = make([]byte, 32)
		}

	}
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
	require.DeepEqual(t, bytesutil.PadTo([]byte{}, 32), got)

	r := [fieldparams.BlockRootsLength][32]byte{{'a'}}
	bRoots := make([][]byte, len(r))
	for i, root := range r {
		tmp := root
		bRoots[i] = tmp[:]
	}
	s, err = InitializeFromProto(&ethpb.BeaconState{BlockRoots: bRoots})
	require.NoError(t, err)
	got, err = s.BlockRootAtIndex(0)
	require.NoError(t, err)
	want := bytesutil.PadTo([]byte{'a'}, 32)
	require.DeepSSZEqual(t, want, got)
}
