package v2_test

import (
	"context"
	"testing"

	v2 "github.com/prysmaticlabs/prysm/v3/beacon-chain/state/v2"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/testing/util"
)

func TestBeaconState_ValidatorMutation_Altair(t *testing.T) {
	testState, _ := util.DeterministicGenesisStateAltair(t, 400)
	pbState, err := v2.ProtobufBeaconState(testState.InnerStateUnsafe())
	require.NoError(t, err)
	testState, err = v2.InitializeFromProto(pbState)
	require.NoError(t, err)

	_, err = testState.HashTreeRoot(context.Background())
	require.NoError(t, err)

	// Reset tries
	require.NoError(t, testState.UpdateValidatorAtIndex(200, new(ethpb.Validator)))
	_, err = testState.HashTreeRoot(context.Background())
	require.NoError(t, err)

	newState1 := testState.Copy()
	_ = testState.Copy()

	require.NoError(t, testState.UpdateValidatorAtIndex(15, &ethpb.Validator{
		PublicKey:                  make([]byte, 48),
		WithdrawalCredentials:      make([]byte, 32),
		EffectiveBalance:           1111,
		Slashed:                    false,
		ActivationEligibilityEpoch: 1112,
		ActivationEpoch:            1114,
		ExitEpoch:                  1116,
		WithdrawableEpoch:          1117,
	}))

	rt, err := testState.HashTreeRoot(context.Background())
	require.NoError(t, err)
	pbState, err = v2.ProtobufBeaconState(testState.InnerStateUnsafe())
	require.NoError(t, err)

	copiedTestState, err := v2.InitializeFromProto(pbState)
	require.NoError(t, err)

	rt2, err := copiedTestState.HashTreeRoot(context.Background())
	require.NoError(t, err)

	assert.Equal(t, rt, rt2)

	require.NoError(t, newState1.UpdateValidatorAtIndex(150, &ethpb.Validator{
		PublicKey:                  make([]byte, 48),
		WithdrawalCredentials:      make([]byte, 32),
		EffectiveBalance:           2111,
		Slashed:                    false,
		ActivationEligibilityEpoch: 2112,
		ActivationEpoch:            2114,
		ExitEpoch:                  2116,
		WithdrawableEpoch:          2117,
	}))

	rt, err = newState1.HashTreeRoot(context.Background())
	require.NoError(t, err)
	pbState, err = v2.ProtobufBeaconState(newState1.InnerStateUnsafe())
	require.NoError(t, err)

	copiedTestState, err = v2.InitializeFromProto(pbState)
	require.NoError(t, err)

	rt2, err = copiedTestState.HashTreeRoot(context.Background())
	require.NoError(t, err)

	assert.Equal(t, rt, rt2)
}
