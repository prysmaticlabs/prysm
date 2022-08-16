package state_native_test

import (
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	statenative "github.com/prysmaticlabs/prysm/v3/beacon-chain/state/state-native"
	testtmpl "github.com/prysmaticlabs/prysm/v3/beacon-chain/state/testing"
	"github.com/prysmaticlabs/prysm/v3/config/features"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/testing/util"
)

func TestBeaconState_ValidatorAtIndexReadOnly_HandlesNilSlice_Phase0(t *testing.T) {
	features.Init(&features.Flags{EnableNativeState: true})
	testtmpl.VerifyBeaconStateValidatorAtIndexReadOnlyHandlesNilSlice(t, func() (state.BeaconState, error) {
		return statenative.InitializeFromProtoUnsafePhase0(&ethpb.BeaconState{
			Validators: nil,
		})
	})
	features.Init(&features.Flags{EnableNativeState: false})
}

func TestBeaconState_ValidatorAtIndexReadOnly_HandlesNilSlice_Altair(t *testing.T) {
	features.Init(&features.Flags{EnableNativeState: true})
	testtmpl.VerifyBeaconStateValidatorAtIndexReadOnlyHandlesNilSlice(t, func() (state.BeaconState, error) {
		return statenative.InitializeFromProtoUnsafeAltair(&ethpb.BeaconStateAltair{
			Validators: nil,
		})
	})
	features.Init(&features.Flags{EnableNativeState: false})
}

func TestBeaconState_ValidatorAtIndexReadOnly_HandlesNilSlice_Bellatrix(t *testing.T) {
	features.Init(&features.Flags{EnableNativeState: true})
	testtmpl.VerifyBeaconStateValidatorAtIndexReadOnlyHandlesNilSlice(t, func() (state.BeaconState, error) {
		return statenative.InitializeFromProtoUnsafeBellatrix(&ethpb.BeaconStateBellatrix{
			Validators: nil,
		})
	})
	features.Init(&features.Flags{EnableNativeState: false})
}

func TestValidatorIndexOutOfRangeError(t *testing.T) {
	features.Init(&features.Flags{EnableNativeState: true})
	err := statenative.NewValidatorIndexOutOfRangeError(1)
	require.Equal(t, err.Error(), "index 1 out of range")
	features.Init(&features.Flags{EnableNativeState: false})
}

func TestValidatorIndexes(t *testing.T) {
	features.Init(&features.Flags{EnableNativeState: true})
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
	features.Init(&features.Flags{EnableNativeState: false})
}
