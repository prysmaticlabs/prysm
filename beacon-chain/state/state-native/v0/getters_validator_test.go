package v0_test

import (
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	v0 "github.com/prysmaticlabs/prysm/beacon-chain/state/state-native/v0"
	testtmpl "github.com/prysmaticlabs/prysm/beacon-chain/state/testing"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

func TestBeaconState_ValidatorAtIndexReadOnly_HandlesNilSlice(t *testing.T) {
	testtmpl.VerifyBeaconStateValidatorAtIndexReadOnlyHandlesNilSlice(t, func() (state.BeaconState, error) {
		return v0.InitializeFromProtoUnsafe(&ethpb.BeaconState{
			Validators: nil,
		})
	})
}
