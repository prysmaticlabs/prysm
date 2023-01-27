package state_test

import (
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	testtmpl "github.com/prysmaticlabs/prysm/v3/beacon-chain/state/testing"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/state"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/state/types"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/testing/util"
)

func TestBeaconState_ValidatorAtIndexReadOnly_HandlesNilSlice_Phase0(t *testing.T) {
	testtmpl.VerifyBeaconStateValidatorAtIndexReadOnlyHandlesNilSlice(t, func() (types.BeaconState, error) {
		return state.InitializeFromProtoUnsafePhase0(&ethpb.BeaconState{
			Validators: nil,
		})
	})
}

func TestBeaconState_ValidatorAtIndexReadOnly_HandlesNilSlice_Altair(t *testing.T) {
	testtmpl.VerifyBeaconStateValidatorAtIndexReadOnlyHandlesNilSlice(t, func() (types.BeaconState, error) {
		return state.InitializeFromProtoUnsafeAltair(&ethpb.BeaconStateAltair{
			Validators: nil,
		})
	})
}

func TestBeaconState_ValidatorAtIndexReadOnly_HandlesNilSlice_Bellatrix(t *testing.T) {
	testtmpl.VerifyBeaconStateValidatorAtIndexReadOnlyHandlesNilSlice(t, func() (types.BeaconState, error) {
		return state.InitializeFromProtoUnsafeBellatrix(&ethpb.BeaconStateBellatrix{
			Validators: nil,
		})
	})
}

func TestBeaconState_ValidatorAtIndexReadOnly_HandlesNilSlice_Capella(t *testing.T) {
	testtmpl.VerifyBeaconStateValidatorAtIndexReadOnlyHandlesNilSlice(t, func() (types.BeaconState, error) {
		return state.InitializeFromProtoUnsafeCapella(&ethpb.BeaconStateCapella{
			Validators: nil,
		})
	})
}

func TestValidatorIndexOutOfRangeError(t *testing.T) {
	err := state.NewValidatorIndexOutOfRangeError(1)
	require.Equal(t, err.Error(), "index 1 out of range")
}

func TestValidatorIndexes(t *testing.T) {
	dState, _ := util.DeterministicGenesisState(t, 10)
	byteValue := dState.PubkeyAtIndex(1)
	t.Run("ValidatorIndexByPubkey", func(t *testing.T) {
		require.Equal(t, hexutil.Encode(byteValue[:]), "0xb89bebc699769726a318c8e9971bd3171297c61aea4a6578a7a4f94b547dcba5bac16a89108b6b6a1fe3695d1a874a0b")
	})
	t.Run("ValidatorAtIndexReadOnly", func(t *testing.T) {
		readOnlyState, err := dState.ValidatorAtIndexReadOnly(1)
		require.NoError(t, err)
		readOnlyBytes := readOnlyState.PublicKey()
		require.NotEmpty(t, readOnlyBytes)
		require.Equal(t, hexutil.Encode(readOnlyBytes[:]), hexutil.Encode(byteValue[:]))
	})
}
