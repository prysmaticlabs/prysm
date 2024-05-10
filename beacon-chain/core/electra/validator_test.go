package electra_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/electra"
	state_native "github.com/prysmaticlabs/prysm/v5/beacon-chain/state/state-native"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	eth "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
)

func TestSwitchToCompoundingValidator(t *testing.T) {
	s, err := state_native.InitializeFromProtoElectra(&eth.BeaconStateElectra{
		Validators: []*eth.Validator{
			{
				WithdrawalCredentials: []byte{}, // No withdrawal credentials
			},
			{
				WithdrawalCredentials: []byte{0x01, 0xFF}, // Has withdrawal credentials
			},
			{
				WithdrawalCredentials: []byte{0x01, 0xFF}, // Has withdrawal credentials
			},
		},
		Balances: []uint64{
			params.BeaconConfig().MinActivationBalance,
			params.BeaconConfig().MinActivationBalance,
			params.BeaconConfig().MinActivationBalance + 100_000, // Has excess balance
		},
	})
	// Test that a validator with no withdrawal credentials cannot be switched to compounding.
	require.NoError(t, err)
	require.ErrorContains(t, "validator has no withdrawal credentials", electra.SwitchToCompoundingValidator(context.TODO(), s, 0))

	// Test that a validator with withdrawal credentials can be switched to compounding.
	require.NoError(t, electra.SwitchToCompoundingValidator(context.TODO(), s, 1))
	v, err := s.ValidatorAtIndex(1)
	require.NoError(t, err)
	require.Equal(t, true, bytes.HasPrefix(v.WithdrawalCredentials, []byte{params.BeaconConfig().CompoundingWithdrawalPrefixByte}), "withdrawal credentials were not updated")
	// val_1 Balance is not changed
	b, err := s.BalanceAtIndex(1)
	require.NoError(t, err)
	require.Equal(t, params.BeaconConfig().MinActivationBalance, b, "balance was changed")
	pbd, err := s.PendingBalanceDeposits()
	require.NoError(t, err)
	require.Equal(t, 0, len(pbd), "pending balance deposits should be empty")

	// Test that a validator with excess balance can be switched to compounding, excess balance is queued.
	require.NoError(t, electra.SwitchToCompoundingValidator(context.TODO(), s, 2))
	b, err = s.BalanceAtIndex(2)
	require.NoError(t, err)
	require.Equal(t, params.BeaconConfig().MinActivationBalance, b, "balance was not changed")
	pbd, err = s.PendingBalanceDeposits()
	require.NoError(t, err)
	require.Equal(t, 1, len(pbd), "pending balance deposits should have one element")
	require.Equal(t, uint64(100_000), pbd[0].Amount, "pending balance deposit amount is incorrect")
	require.Equal(t, primitives.ValidatorIndex(2), pbd[0].Index, "pending balance deposit index is incorrect")
}

func TestQueueEntireBalanceAndResetValidator(t *testing.T) {
	s, err := state_native.InitializeFromProtoElectra(&eth.BeaconStateElectra{
		Validators: []*eth.Validator{
			{
				EffectiveBalance:           params.BeaconConfig().MinActivationBalance + 100_000,
				ActivationEligibilityEpoch: primitives.Epoch(100),
			},
		},
		Balances: []uint64{
			params.BeaconConfig().MinActivationBalance + 100_000,
		},
	})
	require.NoError(t, err)
	require.NoError(t, electra.QueueEntireBalanceAndResetValidator(context.TODO(), s, 0))
	b, err := s.BalanceAtIndex(0)
	require.NoError(t, err)
	require.Equal(t, uint64(0), b, "balance was not changed")
	v, err := s.ValidatorAtIndex(0)
	require.NoError(t, err)
	require.Equal(t, uint64(0), v.EffectiveBalance, "effective balance was not reset")
	require.Equal(t, params.BeaconConfig().FarFutureEpoch, v.ActivationEligibilityEpoch, "activation eligibility epoch was not reset")
	pbd, err := s.PendingBalanceDeposits()
	require.NoError(t, err)
	require.Equal(t, 1, len(pbd), "pending balance deposits should have one element")
	require.Equal(t, params.BeaconConfig().MinActivationBalance+100_000, pbd[0].Amount, "pending balance deposit amount is incorrect")
	require.Equal(t, primitives.ValidatorIndex(0), pbd[0].Index, "pending balance deposit index is incorrect")
}
