package v3_test

import (
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	v3 "github.com/prysmaticlabs/prysm/beacon-chain/state/v3"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestBeaconState_ValidatorAtIndexReadOnly_HandlesNilSlice(t *testing.T) {
	st, err := v3.InitializeFromProtoUnsafe(&ethpb.BeaconStateBellatrix{
		Validators: nil,
	})
	require.NoError(t, err)

	_, err = st.ValidatorAtIndexReadOnly(0)
	assert.Equal(t, state.ErrNilValidatorsInState, err)
}
