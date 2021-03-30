package testutil

import (
	"context"
	"reflect"
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestNewBeaconState(t *testing.T) {
	st, err := NewBeaconState()
	require.NoError(t, err)
	b, err := st.MarshalSSZ()
	require.NoError(t, err)
	got := &pb.BeaconState{}
	require.NoError(t, got.UnmarshalSSZ(b))
	if !reflect.DeepEqual(st.InnerStateUnsafe(), got) {
		t.Fatal("State did not match after round trip marshal")
	}
}

func TestNewBeaconState_HashTreeRoot(t *testing.T) {
	st, err := NewBeaconState()
	require.NoError(t, err)
	_, err = st.HashTreeRoot(context.Background())
	require.NoError(t, err)
	st, err = NewBeaconState()
	require.NoError(t, err)
	_, err = st.HashTreeRoot(context.Background())
	require.NoError(t, err)
}
