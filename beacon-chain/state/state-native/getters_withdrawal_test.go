package state_native

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v3/config/params"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	enginev1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/runtime/version"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
)

func TestNextWithdrawalIndex(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		s := BeaconState{version: version.Capella, nextWithdrawalIndex: 123}
		i, err := s.NextWithdrawalIndex()
		require.NoError(t, err)
		assert.Equal(t, uint64(123), i)
	})
	t.Run("version before Capella not supported", func(t *testing.T) {
		s := BeaconState{version: version.Bellatrix}
		_, err := s.NextWithdrawalIndex()
		assert.ErrorContains(t, "NextWithdrawalIndex is not supported", err)
	})
}

func TestLastWithdrawalValidatorIndex(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		s := BeaconState{version: version.Capella, nextWithdrawalValidatorIndex: 123}
		i, err := s.LastWithdrawalValidatorIndex()
		require.NoError(t, err)
		assert.Equal(t, types.ValidatorIndex(123), i)
	})
	t.Run("version before Capella not supported", func(t *testing.T) {
		s := BeaconState{version: version.Bellatrix}
		_, err := s.LastWithdrawalValidatorIndex()
		assert.ErrorContains(t, "LastWithdrawalValidatorIndex is not supported", err)
	})
}

func TestHasETH1WithdrawalCredentials(t *testing.T) {
	creds := []byte{0xFA, 0xCC}
	v := &ethpb.Validator{WithdrawalCredentials: creds}
	require.Equal(t, false, hasETH1WithdrawalCredential(v))
	creds = []byte{params.BeaconConfig().ETH1AddressWithdrawalPrefixByte, 0xCC}
	v = &ethpb.Validator{WithdrawalCredentials: creds}
	require.Equal(t, true, hasETH1WithdrawalCredential(v))
	// No Withdrawal cred
	v = &ethpb.Validator{}
	require.Equal(t, false, hasETH1WithdrawalCredential(v))
}

func TestIsFullyWithdrawableValidator(t *testing.T) {
	// No ETH1 prefix
	creds := []byte{0xFA, 0xCC}
	v := &ethpb.Validator{
		WithdrawalCredentials: creds,
		WithdrawableEpoch:     2,
	}
	require.Equal(t, false, isFullyWithdrawableValidator(v, 3))
	// Wrong withdrawable epoch
	creds = []byte{params.BeaconConfig().ETH1AddressWithdrawalPrefixByte, 0xCC}
	v = &ethpb.Validator{
		WithdrawalCredentials: creds,
		WithdrawableEpoch:     2,
	}
	require.Equal(t, false, isFullyWithdrawableValidator(v, 1))
	// Fully withdrawable
	creds = []byte{params.BeaconConfig().ETH1AddressWithdrawalPrefixByte, 0xCC}
	v = &ethpb.Validator{
		WithdrawalCredentials: creds,
		WithdrawableEpoch:     2,
	}
	require.Equal(t, true, isFullyWithdrawableValidator(v, 3))
}

func TestIsPartiallyWithdrawableValidator(t *testing.T) {
	// No ETH1 prefix
	creds := []byte{0xFA, 0xCC}
	v, err := NewValidator(&ethpb.Validator{
		WithdrawalCredentials: creds,
	})
	require.NoError(t, err)
	require.Equal(t, false, v.IsPartiallyWithdrawable(params.BeaconConfig().MaxEffectiveBalance+1))
	// Not the right effective balance
	creds = []byte{params.BeaconConfig().ETH1AddressWithdrawalPrefixByte, 0xCC}
	v, err = NewValidator(&ethpb.Validator{
		WithdrawalCredentials: creds,
		EffectiveBalance:      params.BeaconConfig().MaxEffectiveBalance - 1,
	})
	require.NoError(t, err)
	require.Equal(t, false, v.IsPartiallyWithdrawable(params.BeaconConfig().MaxEffectiveBalance+1))
	// Not enough balance
	creds = []byte{params.BeaconConfig().ETH1AddressWithdrawalPrefixByte, 0xCC}
	v, err = NewValidator(&ethpb.Validator{
		WithdrawalCredentials: creds,
		EffectiveBalance:      params.BeaconConfig().MaxEffectiveBalance,
	})
	require.NoError(t, err)
	require.Equal(t, false, v.IsPartiallyWithdrawable(params.BeaconConfig().MaxEffectiveBalance))
	// Partially Withdrawable
	creds = []byte{params.BeaconConfig().ETH1AddressWithdrawalPrefixByte, 0xCC}
	v, err = NewValidator(&ethpb.Validator{
		WithdrawalCredentials: creds,
		EffectiveBalance:      params.BeaconConfig().MaxEffectiveBalance,
	})
	require.NoError(t, err)
	require.Equal(t, true, v.IsPartiallyWithdrawable(params.BeaconConfig().MaxEffectiveBalance+1))
}

func TestExpectedWithdrawals(t *testing.T) {
	t.Run("no withdrawals", func(t *testing.T) {
		s := BeaconState{
			version:    version.Capella,
			validators: make([]*ethpb.Validator, 100),
			balances:   make([]uint64, 100),
		}
		for i := range s.validators {
			s.balances[i] = params.BeaconConfig().MaxEffectiveBalance
			val := &ethpb.Validator{
				WithdrawalCredentials: make([]byte, 32),
				EffectiveBalance:      params.BeaconConfig().MaxEffectiveBalance,
				WithdrawableEpoch:     types.Epoch(1),
			}
			val.WithdrawalCredentials[0] = params.BeaconConfig().ETH1AddressWithdrawalPrefixByte
			s.validators[i] = val
		}
		expected, err := s.ExpectedWithdrawals()
		require.NoError(t, err)
		require.Equal(t, 0, len(expected))
	})
	t.Run("one fully withdrawable", func(t *testing.T) {
		s := BeaconState{
			version:                      version.Capella,
			validators:                   make([]*ethpb.Validator, 100),
			balances:                     make([]uint64, 100),
			nextWithdrawalValidatorIndex: 20,
		}
		for i := range s.validators {
			s.balances[i] = params.BeaconConfig().MaxEffectiveBalance
			val := &ethpb.Validator{
				WithdrawalCredentials: make([]byte, 32),
				EffectiveBalance:      params.BeaconConfig().MaxEffectiveBalance,
				WithdrawableEpoch:     types.Epoch(1),
			}
			val.WithdrawalCredentials[0] = params.BeaconConfig().ETH1AddressWithdrawalPrefixByte
			s.validators[i] = val
		}
		s.validators[3].WithdrawableEpoch = types.Epoch(0)
		expected, err := s.ExpectedWithdrawals()
		require.NoError(t, err)
		require.Equal(t, 1, len(expected))
		withdrawal := &enginev1.Withdrawal{
			WithdrawalIndex:  0,
			ValidatorIndex:   3,
			ExecutionAddress: s.validators[3].WithdrawalCredentials[12:],
			Amount:           s.balances[3],
		}
		require.DeepEqual(t, withdrawal, expected[0])
	})
	t.Run("one partially withdrawable", func(t *testing.T) {
		s := BeaconState{
			version:    version.Capella,
			validators: make([]*ethpb.Validator, 100),
			balances:   make([]uint64, 100),
		}
		for i := range s.validators {
			s.balances[i] = params.BeaconConfig().MaxEffectiveBalance
			val := &ethpb.Validator{
				WithdrawalCredentials: make([]byte, 32),
				EffectiveBalance:      params.BeaconConfig().MaxEffectiveBalance,
				WithdrawableEpoch:     types.Epoch(1),
			}
			val.WithdrawalCredentials[0] = params.BeaconConfig().ETH1AddressWithdrawalPrefixByte
			s.validators[i] = val
		}
		s.balances[3] += params.BeaconConfig().MinDepositAmount
		expected, err := s.ExpectedWithdrawals()
		require.NoError(t, err)
		require.Equal(t, 1, len(expected))
		withdrawal := &enginev1.Withdrawal{
			WithdrawalIndex:  0,
			ValidatorIndex:   3,
			ExecutionAddress: s.validators[3].WithdrawalCredentials[12:],
			Amount:           params.BeaconConfig().MinDepositAmount,
		}
		require.DeepEqual(t, withdrawal, expected[0])
	})
	t.Run("one partially and one fully withdrawable", func(t *testing.T) {
		s := BeaconState{
			version:    version.Capella,
			validators: make([]*ethpb.Validator, 100),
			balances:   make([]uint64, 100),
		}
		for i := range s.validators {
			s.balances[i] = params.BeaconConfig().MaxEffectiveBalance
			val := &ethpb.Validator{
				WithdrawalCredentials: make([]byte, 32),
				EffectiveBalance:      params.BeaconConfig().MaxEffectiveBalance,
				WithdrawableEpoch:     types.Epoch(1),
			}
			val.WithdrawalCredentials[0] = params.BeaconConfig().ETH1AddressWithdrawalPrefixByte
			s.validators[i] = val
		}
		s.balances[3] += params.BeaconConfig().MinDepositAmount
		s.validators[7].WithdrawableEpoch = types.Epoch(0)
		expected, err := s.ExpectedWithdrawals()
		require.NoError(t, err)
		require.Equal(t, 2, len(expected))

		withdrawalFull := &enginev1.Withdrawal{
			WithdrawalIndex:  1,
			ValidatorIndex:   7,
			ExecutionAddress: s.validators[7].WithdrawalCredentials[12:],
			Amount:           s.balances[7],
		}
		withdrawalPartial := &enginev1.Withdrawal{
			WithdrawalIndex:  0,
			ValidatorIndex:   3,
			ExecutionAddress: s.validators[3].WithdrawalCredentials[12:],
			Amount:           params.BeaconConfig().MinDepositAmount,
		}
		require.DeepEqual(t, withdrawalPartial, expected[0])
		require.DeepEqual(t, withdrawalFull, expected[1])
	})
	t.Run("all partially withdrawable", func(t *testing.T) {
		s := BeaconState{
			version:    version.Capella,
			validators: make([]*ethpb.Validator, 100),
			balances:   make([]uint64, 100),
		}
		for i := range s.validators {
			s.balances[i] = params.BeaconConfig().MaxEffectiveBalance + 1
			val := &ethpb.Validator{
				WithdrawalCredentials: make([]byte, 32),
				EffectiveBalance:      params.BeaconConfig().MaxEffectiveBalance,
				WithdrawableEpoch:     types.Epoch(1),
			}
			val.WithdrawalCredentials[0] = params.BeaconConfig().ETH1AddressWithdrawalPrefixByte
			s.validators[i] = val
		}
		expected, err := s.ExpectedWithdrawals()
		require.NoError(t, err)
		require.Equal(t, params.BeaconConfig().MaxWithdrawalsPerPayload, uint64(len(expected)))
		withdrawal := &enginev1.Withdrawal{
			WithdrawalIndex:  0,
			ValidatorIndex:   1,
			ExecutionAddress: s.validators[0].WithdrawalCredentials[12:],
			Amount:           1,
		}
		require.DeepEqual(t, withdrawal, expected[0])
	})
	t.Run("all fully withdrawable", func(t *testing.T) {
		s := BeaconState{
			version:    version.Capella,
			validators: make([]*ethpb.Validator, 100),
			balances:   make([]uint64, 100),
		}
		for i := range s.validators {
			s.balances[i] = params.BeaconConfig().MaxEffectiveBalance
			val := &ethpb.Validator{
				WithdrawalCredentials: make([]byte, 32),
				EffectiveBalance:      params.BeaconConfig().MaxEffectiveBalance,
				WithdrawableEpoch:     types.Epoch(0),
			}
			val.WithdrawalCredentials[0] = params.BeaconConfig().ETH1AddressWithdrawalPrefixByte
			s.validators[i] = val
		}
		expected, err := s.ExpectedWithdrawals()
		require.NoError(t, err)
		require.Equal(t, params.BeaconConfig().MaxWithdrawalsPerPayload, uint64(len(expected)))
		withdrawal := &enginev1.Withdrawal{
			WithdrawalIndex:  0,
			ValidatorIndex:   1,
			ExecutionAddress: s.validators[0].WithdrawalCredentials[12:],
			Amount:           params.BeaconConfig().MaxEffectiveBalance,
		}
		require.DeepEqual(t, withdrawal, expected[0])
	})
	t.Run("all fully and partially withdrawable", func(t *testing.T) {
		s := BeaconState{
			version:    version.Capella,
			validators: make([]*ethpb.Validator, 100),
			balances:   make([]uint64, 100),
		}
		for i := range s.validators {
			s.balances[i] = params.BeaconConfig().MaxEffectiveBalance + 1
			val := &ethpb.Validator{
				WithdrawalCredentials: make([]byte, 32),
				EffectiveBalance:      params.BeaconConfig().MaxEffectiveBalance,
				WithdrawableEpoch:     types.Epoch(0),
			}
			val.WithdrawalCredentials[0] = params.BeaconConfig().ETH1AddressWithdrawalPrefixByte
			s.validators[i] = val
		}
		expected, err := s.ExpectedWithdrawals()
		require.NoError(t, err)
		require.Equal(t, params.BeaconConfig().MaxWithdrawalsPerPayload, uint64(len(expected)))
		withdrawal := &enginev1.Withdrawal{
			WithdrawalIndex:  0,
			ValidatorIndex:   1,
			ExecutionAddress: s.validators[0].WithdrawalCredentials[12:],
			Amount:           params.BeaconConfig().MaxEffectiveBalance + 1,
		}
		require.DeepEqual(t, withdrawal, expected[0])
	})
}
