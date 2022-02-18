package testing

import (
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/require"
)

type getStateWithLatestBlockHeader func(*ethpb.BeaconBlockHeader) (state.BeaconState, error)

func VerifyBeaconState_LatestBlockHeader(
	t *testing.T,
	factory getState,
	factoryLBH getStateWithLatestBlockHeader,
) {
	s, err := factory()
	require.NoError(t, err)
	got := s.LatestBlockHeader()
	require.DeepEqual(t, (*ethpb.BeaconBlockHeader)(nil), got)

	want := &ethpb.BeaconBlockHeader{Slot: 100}
	s, err = factoryLBH(want)
	require.NoError(t, err)
	got = s.LatestBlockHeader()
	require.DeepEqual(t, want, got)

	// Test copy does not mutate.
	got.Slot = 101
	require.DeepNotEqual(t, want, got)
}

type getStateWithLBlockRoots func([][]byte) (state.BeaconState, error)

func VerifyBeaconState_BlockRoots(
	t *testing.T,
	factory getState,
	factoryBR getStateWithLBlockRoots,
) {
	s, err := factory()
	require.NoError(t, err)
	got := s.BlockRoots()
	require.DeepEqual(t, ([][]byte)(nil), got)

	want := [][]byte{{'a'}}
	s, err = factoryBR(want)
	require.NoError(t, err)
	got = s.BlockRoots()
	require.DeepEqual(t, want, got)

	// Test copy does not mutate.
	got[0][0] = 'b'
	require.DeepNotEqual(t, want, got)
}

func VerifyBeaconState_BlockRootsNative(
	t *testing.T,
	factory getState,
	factoryBR getStateWithLBlockRoots,
) {
	s, err := factory()
	require.NoError(t, err)
	got := s.BlockRoots()
	want := make([][]byte, fieldparams.BlockRootsLength)
	for i := range want {
		want[i] = make([]byte, 32)
	}
	require.DeepEqual(t, want, got)

	want = make([][]byte, fieldparams.BlockRootsLength)
	for i := range want {
		if i == 0 {
			want[i] = bytesutil.PadTo([]byte{'a'}, 32)
		} else {
			want[i] = make([]byte, 32)
		}

	}
	s, err = factoryBR(want)
	require.NoError(t, err)
	got = s.BlockRoots()
	require.DeepEqual(t, want, got)

	// Test copy does not mutate.
	got[0][0] = 'b'
	require.DeepNotEqual(t, want, got)
}

func VerifyBeaconState_BlockRootAtIndex(
	t *testing.T,
	factory getState,
	factoryBR getStateWithLBlockRoots,
) {
	s, err := factory()
	require.NoError(t, err)
	got, err := s.BlockRootAtIndex(0)
	require.NoError(t, err)
	require.DeepEqual(t, ([]byte)(nil), got)

	r := [][]byte{{'a'}}
	s, err = factoryBR(r)
	require.NoError(t, err)
	got, err = s.BlockRootAtIndex(0)
	require.NoError(t, err)
	want := bytesutil.PadTo([]byte{'a'}, fieldparams.RootLength)
	require.DeepSSZEqual(t, want, got)
}

func VerifyBeaconState_BlockRootAtIndexNative(
	t *testing.T,
	factory getState,
	factoryBR getStateWithLBlockRoots,
) {
	s, err := factory()
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
	s, err = factoryBR(bRoots)
	require.NoError(t, err)
	got, err = s.BlockRootAtIndex(0)
	require.NoError(t, err)
	want := bytesutil.PadTo([]byte{'a'}, 32)
	require.DeepSSZEqual(t, want, got)
}
