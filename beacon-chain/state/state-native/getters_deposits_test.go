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

func TestPendingDeposits(t *testing.T) {
	s, err := state_native.InitializeFromProtoElectra(&eth.BeaconStateElectra{
		PendingDeposits: []*eth.PendingDeposit{
			{Amount: 2},
			{Amount: 4},
		},
	})
	require.NoError(t, err)
	pbd, err := s.PendingDeposits()
	require.NoError(t, err)
	require.Equal(t, 2, len(pbd))

	require.Equal(t, uint64(2), pbd[0].Amount)

	require.Equal(t, uint64(4), pbd[1].Amount)

	// Fails for older than electra state
	s, err = state_native.InitializeFromProtoDeneb(&eth.BeaconStateDeneb{})
	require.NoError(t, err)
	_, err = s.DepositBalanceToConsume()
	require.ErrorContains(t, "not supported", err)
}
