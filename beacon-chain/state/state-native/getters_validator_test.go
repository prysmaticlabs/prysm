package state_native_test

import (
	"math"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	statenative "github.com/prysmaticlabs/prysm/v5/beacon-chain/state/state-native"
	testtmpl "github.com/prysmaticlabs/prysm/v5/beacon-chain/state/testing"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	consensus_types "github.com/prysmaticlabs/prysm/v5/consensus-types"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	"github.com/prysmaticlabs/prysm/v5/testing/util"
)

func TestBeaconState_ValidatorAtIndexReadOnly_HandlesNilSlice_Phase0(t *testing.T) {
	testtmpl.VerifyBeaconStateValidatorAtIndexReadOnlyHandlesNilSlice(t, func() (state.BeaconState, error) {
		return statenative.InitializeFromProtoUnsafePhase0(&ethpb.BeaconState{
			Validators: nil,
		})
	})
}

func TestBeaconState_ValidatorAtIndexReadOnly_HandlesNilSlice_Altair(t *testing.T) {
	testtmpl.VerifyBeaconStateValidatorAtIndexReadOnlyHandlesNilSlice(t, func() (state.BeaconState, error) {
		return statenative.InitializeFromProtoUnsafeAltair(&ethpb.BeaconStateAltair{
			Validators: nil,
		})
	})
}

func TestBeaconState_ValidatorAtIndexReadOnly_HandlesNilSlice_Bellatrix(t *testing.T) {
	testtmpl.VerifyBeaconStateValidatorAtIndexReadOnlyHandlesNilSlice(t, func() (state.BeaconState, error) {
		return statenative.InitializeFromProtoUnsafeBellatrix(&ethpb.BeaconStateBellatrix{
			Validators: nil,
		})
	})
}

func TestBeaconState_ValidatorAtIndexReadOnly_HandlesNilSlice_Capella(t *testing.T) {
	testtmpl.VerifyBeaconStateValidatorAtIndexReadOnlyHandlesNilSlice(t, func() (state.BeaconState, error) {
		return statenative.InitializeFromProtoUnsafeCapella(&ethpb.BeaconStateCapella{
			Validators: nil,
		})
	})
}

func TestBeaconState_ValidatorAtIndexReadOnly_HandlesNilSlice_Deneb(t *testing.T) {
	testtmpl.VerifyBeaconStateValidatorAtIndexReadOnlyHandlesNilSlice(t, func() (state.BeaconState, error) {
		return statenative.InitializeFromProtoUnsafeDeneb(&ethpb.BeaconStateDeneb{
			Validators: nil,
		})
	})
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

func TestActiveBalanceAtIndex(t *testing.T) {
	// Test setup with a state with 4 validators.
	// Validators 0 & 1 have compounding withdrawal credentials while validators 2 & 3 have BLS withdrawal credentials.
	pb := &ethpb.BeaconStateElectra{
		Validators: []*ethpb.Validator{
			{
				WithdrawalCredentials: []byte{params.BeaconConfig().CompoundingWithdrawalPrefixByte},
			},
			{
				WithdrawalCredentials: []byte{params.BeaconConfig().CompoundingWithdrawalPrefixByte},
			},
			{
				WithdrawalCredentials: []byte{params.BeaconConfig().BLSWithdrawalPrefixByte},
			},
			{
				WithdrawalCredentials: []byte{params.BeaconConfig().BLSWithdrawalPrefixByte},
			},
		},
		Balances: []uint64{
			55,
			math.MaxUint64,
			55,
			math.MaxUint64,
		},
	}
	state, err := statenative.InitializeFromProtoUnsafeElectra(pb)
	require.NoError(t, err)

	ab, err := state.ActiveBalanceAtIndex(0)
	require.NoError(t, err)
	require.Equal(t, uint64(55), ab)

	ab, err = state.ActiveBalanceAtIndex(1)
	require.NoError(t, err)
	require.Equal(t, params.BeaconConfig().MaxEffectiveBalanceElectra, ab)

	ab, err = state.ActiveBalanceAtIndex(2)
	require.NoError(t, err)
	require.Equal(t, uint64(55), ab)

	ab, err = state.ActiveBalanceAtIndex(3)
	require.NoError(t, err)
	require.Equal(t, params.BeaconConfig().MinActivationBalance, ab)

	// Accessing a validator index out of bounds should error.
	_, err = state.ActiveBalanceAtIndex(4)
	require.ErrorIs(t, err, consensus_types.ErrOutOfBounds)

	// Accessing a validator wwhere balance slice is out of bounds for some reason.
	require.NoError(t, state.SetBalances([]uint64{}))
	_, err = state.ActiveBalanceAtIndex(0)
	require.ErrorIs(t, err, consensus_types.ErrOutOfBounds)
}

func TestPendingBalanceToWithdraw(t *testing.T) {
	pb := &ethpb.BeaconStateElectra{
		PendingPartialWithdrawals: []*ethpb.PendingPartialWithdrawal{
			{
				Amount: 100,
			},
			{
				Amount: 200,
			},
			{
				Amount: 300,
			},
		},
	}
	state, err := statenative.InitializeFromProtoUnsafeElectra(pb)
	require.NoError(t, err)

	ab, err := state.PendingBalanceToWithdraw(0)
	require.NoError(t, err)
	require.Equal(t, uint64(600), ab)
}
