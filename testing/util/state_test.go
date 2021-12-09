package util

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/testing/require"
)

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
