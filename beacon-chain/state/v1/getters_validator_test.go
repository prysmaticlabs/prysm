package v1_test

import (
	"testing"

	v1 "github.com/prysmaticlabs/prysm/beacon-chain/state/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestBeaconState_ValidatorAtIndexReadOnly_HandlesNilSlice(t *testing.T) {
	st, err := v1.InitializeFromProtoUnsafe(&pb.BeaconState{
		Validators: nil,
	})
	require.NoError(t, err)

	_, err = st.ValidatorAtIndexReadOnly(0)
	assert.Equal(t, v1.ErrNilValidatorsInState, err)
}
