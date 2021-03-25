package stateutil

import (
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
)

func Test_handleValidatorSlice_OutOfRange(t *testing.T) {
	vals := make([]*ethpb.Validator, 1)
	indices := []uint64{3}
	_, err := HandleValidatorSlice(vals, indices, false)
	assert.ErrorContains(t, "index 3 greater than number of validators 1", err)
}
