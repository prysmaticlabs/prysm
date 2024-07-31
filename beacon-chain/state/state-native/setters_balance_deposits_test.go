package state_native_test

import (
	"testing"

	state_native "github.com/prysmaticlabs/prysm/v5/beacon-chain/state/state-native"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	eth "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
)

func TestAppendPendingBalanceDeposit(t *testing.T) {
	s, err := state_native.InitializeFromProtoElectra(&eth.BeaconStateElectra{})
	require.NoError(t, err)
	pbd, err := s.PendingBalanceDeposits()
	require.NoError(t, err)
	require.Equal(t, 0, len(pbd))
	require.NoError(t, s.AppendPendingBalanceDeposit(1, 10))
	pbd, err = s.PendingBalanceDeposits()
	require.NoError(t, err)
	require.Equal(t, 1, len(pbd))
	require.Equal(t, primitives.ValidatorIndex(1), pbd[0].Index)
	require.Equal(t, uint64(10), pbd[0].Amount)

	// Fails for versions older than electra
	s, err = state_native.InitializeFromProtoDeneb(&eth.BeaconStateDeneb{})
	require.NoError(t, err)
	require.ErrorContains(t, "not supported", s.AppendPendingBalanceDeposit(1, 1))
}

func TestSetPendingBalanceDeposits(t *testing.T) {
	s, err := state_native.InitializeFromProtoElectra(&eth.BeaconStateElectra{})
	require.NoError(t, err)
	pbd, err := s.PendingBalanceDeposits()
	require.NoError(t, err)
	require.Equal(t, 0, len(pbd))
	require.NoError(t, s.SetPendingBalanceDeposits([]*eth.PendingBalanceDeposit{{}, {}, {}}))
	pbd, err = s.PendingBalanceDeposits()
	require.NoError(t, err)
	require.Equal(t, 3, len(pbd))

	// Fails for versions older than electra
	s, err = state_native.InitializeFromProtoDeneb(&eth.BeaconStateDeneb{})
	require.NoError(t, err)
	require.ErrorContains(t, "not supported", s.SetPendingBalanceDeposits([]*eth.PendingBalanceDeposit{{}, {}, {}}))
}

func TestSetDepositBalanceToConsume(t *testing.T) {
	s, err := state_native.InitializeFromProtoElectra(&eth.BeaconStateElectra{})
	require.NoError(t, err)
	require.NoError(t, s.SetDepositBalanceToConsume(10))
	dbtc, err := s.DepositBalanceToConsume()
	require.NoError(t, err)
	require.Equal(t, primitives.Gwei(10), dbtc)

	// Fails for versions older than electra
	s, err = state_native.InitializeFromProtoDeneb(&eth.BeaconStateDeneb{})
	require.NoError(t, err)
	require.ErrorContains(t, "not supported", s.SetDepositBalanceToConsume(10))
}
