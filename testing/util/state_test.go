package util

import (
	"context"
	"reflect"
	"testing"

	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestNewBeaconState(t *testing.T) {
	st, err := NewBeaconState()
	require.NoError(t, err)
	b, err := st.MarshalSSZ()
	require.NoError(t, err)
	got := &ethpb.BeaconState{}
	require.NoError(t, got.UnmarshalSSZ(b))
	if !reflect.DeepEqual(st.ToProtoUnsafe(), got) {
		t.Fatal("State did not match after round trip marshal")
	}
}

func TestNewBeaconState_HashTreeRoot(t *testing.T) {
	st, err := NewBeaconState()
	require.NoError(t, err)
	_, err = st.HTR(context.Background())
	require.NoError(t, err)
	st, err = NewBeaconState()
	require.NoError(t, err)
	_, err = st.HTR(context.Background())
	require.NoError(t, err)
}
