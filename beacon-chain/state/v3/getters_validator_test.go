package v3_test

import (
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	testtmpl "github.com/prysmaticlabs/prysm/beacon-chain/state/testing"
	v3 "github.com/prysmaticlabs/prysm/beacon-chain/state/v3"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

func TestBeaconState_ValidatorAtIndexReadOnly_HandlesNilSlice(t *testing.T) {
	testtmpl.VerifyBeaconState_ValidatorAtIndexReadOnly_HandlesNilSlice(t, func() (state.BeaconState, error) {
		return v3.InitializeFromProtoUnsafe(&ethpb.BeaconStateBellatrix{
			Validators: nil,
		})
	})
}
