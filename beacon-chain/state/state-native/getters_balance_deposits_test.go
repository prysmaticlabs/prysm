package state_native_test

import (
	"testing"

	state_native "github.com/prysmaticlabs/prysm/v5/beacon-chain/state/state-native"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	eth "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
)

func TestDepositBalanceToConsume(t *testing.T) {
	s, err := state_native.InitializeFromProtoElectra(&eth.BeaconStateElectra{
		DepositBalanceToConsume: 44,
	})
	require.NoError(t, err)
	dbtc, err := s.DepositBalanceToConsume()
	require.NoError(t, err)
	require.Equal(t, primitives.Gwei(44), dbtc)

	// Fails for older than electra state
	s, err = state_native.InitializeFromProtoDeneb(&eth.BeaconStateDeneb{})
	require.NoError(t, err)
	_, err = s.DepositBalanceToConsume()
	require.ErrorContains(t, "not supported", err)
}

func TestPendingBalanceDeposits(t *testing.T) {
	s, err := state_native.InitializeFromProtoElectra(&eth.BeaconStateElectra{
		PendingBalanceDeposits: []*eth.PendingBalanceDeposit{
			{Index: 1, Amount: 2},
			{Index: 3, Amount: 4},
		},
	})
	require.NoError(t, err)
	pbd, err := s.PendingBalanceDeposits()
	require.NoError(t, err)
	require.Equal(t, 2, len(pbd))
	require.Equal(t, primitives.ValidatorIndex(1), pbd[0].Index)
	require.Equal(t, uint64(2), pbd[0].Amount)
	require.Equal(t, primitives.ValidatorIndex(3), pbd[1].Index)
	require.Equal(t, uint64(4), pbd[1].Amount)

	// Fails for older than electra state
	s, err = state_native.InitializeFromProtoDeneb(&eth.BeaconStateDeneb{})
	require.NoError(t, err)
	_, err = s.DepositBalanceToConsume()
	require.ErrorContains(t, "not supported", err)
}
