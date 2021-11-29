package v1

import (
	"testing"

	customtypes "github.com/prysmaticlabs/prysm/beacon-chain/state/custom-types"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	v1alpha1 "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/require"
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
	require.DeepEqual(t, [customtypes.BlockRootsSize][32]byte{}, *got)

	want := [customtypes.BlockRootsSize][32]byte{{'a'}}
	bRoots := make([][]byte, len(want))
	for i, r := range want {
		tmp := r
		bRoots[i] = tmp[:]
	}
	s, err = InitializeFromProto(&ethpb.BeaconState{BlockRoots: bRoots})
	require.NoError(t, err)
	got = s.BlockRoots()
	require.DeepEqual(t, want, *got)

	// Test copy does not mutate.
	got[0][0] = 'b'
	require.DeepNotEqual(t, want, *got)
}

func TestBeaconState_BlockRootAtIndex(t *testing.T) {
	s, err := InitializeFromProto(&ethpb.BeaconState{})
	require.NoError(t, err)
	got, err := s.BlockRootAtIndex(0)
	require.NoError(t, err)
	require.DeepEqual(t, [32]byte{}, got)

	r := [customtypes.BlockRootsSize][32]byte{{'a'}}
	bRoots := make([][]byte, len(r))
	for i, root := range r {
		tmp := root
		bRoots[i] = tmp[:]
	}
	s, err = InitializeFromProto(&ethpb.BeaconState{BlockRoots: bRoots})
	require.NoError(t, err)
	got, err = s.BlockRootAtIndex(0)
	require.NoError(t, err)
	want := [32]byte{'a'}
	require.DeepSSZEqual(t, want, got)
}
