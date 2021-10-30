package v1_test

import (
	"testing"

	v1 "github.com/prysmaticlabs/prysm/beacon-chain/state/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestBeaconState_ValidatorAtIndexReadOnly_HandlesNilSlice(t *testing.T) {
	st, err := v1.InitializeFromProtoUnsafe(&ethpb.BeaconState{
		Validators: nil,
	})
	require.NoError(t, err)

	_, err = st.ValidatorAtIndexReadOnly(0)
	assert.Equal(t, v1.ErrNilValidatorsInState, err)
}

func TestArraysTreeRoot_OnlyPowerOf2(t *testing.T) {
	_, err := v1.RootsArrayHashTreeRoot([][]byte{}, 1, "testing")
	assert.NoError(t, err)
	_, err = v1.RootsArrayHashTreeRoot([][]byte{}, 4, "testing")
	assert.NoError(t, err)
	_, err = v1.RootsArrayHashTreeRoot([][]byte{}, 8, "testing")
	assert.NoError(t, err)
	_, err = v1.RootsArrayHashTreeRoot([][]byte{}, 10, "testing")
	assert.ErrorContains(t, "hash layer is a non power of 2", err)
}

func TestArraysTreeRoot_ZeroLength(t *testing.T) {
	_, err := v1.RootsArrayHashTreeRoot([][]byte{}, 0, "testing")
	assert.ErrorContains(t, "zero leaves provided", err)
}
