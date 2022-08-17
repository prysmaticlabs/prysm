package v3_test

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	testtmpl "github.com/prysmaticlabs/prysm/v3/beacon-chain/state/testing"
	v3 "github.com/prysmaticlabs/prysm/v3/beacon-chain/state/v3"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
)

func TestBeaconState_ValidatorAtIndexReadOnly_HandlesNilSlice(t *testing.T) {
	testtmpl.VerifyBeaconStateValidatorAtIndexReadOnlyHandlesNilSlice(t, func() (state.BeaconState, error) {
		return v3.InitializeFromProtoUnsafe(&ethpb.BeaconStateBellatrix{
			Validators: nil,
		})
	})
}
