package state_native

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	enginev1 "github.com/prysmaticlabs/prysm/v4/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/runtime/version"
	"github.com/prysmaticlabs/prysm/v4/testing/assert"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
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

func TestNextWithdrawalValidatorIndex(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		s := BeaconState{version: version.Capella, nextWithdrawalValidatorIndex: 123}
		i, err := s.NextWithdrawalValidatorIndex()
		require.NoError(t, err)
		assert.Equal(t, primitives.ValidatorIndex(123), i)
	})
	t.Run("version before Capella not supported", func(t *testing.T) {
		s := BeaconState{version: version.Bellatrix}
		_, err := s.NextWithdrawalValidatorIndex()
		assert.ErrorContains(t, "NextWithdrawalValidatorIndex is not supported", err)
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

func TestExpectedWithdrawals(t *testing.T) {
	t.Run("no withdrawals", func(t *testing.T) {
		s := &BeaconState{
			version:    version.Capella,
			validators: NewMultiValueValidators(make([]*ethpb.Validator, 100)),
			balances:   NewMultiValueBalances(make([]uint64, 100)),
		}
		for i := range s.validators.Value(s) {
			require.NoError(t, s.balances.UpdateAt(s, uint64(i), params.BeaconConfig().MaxEffectiveBalance))
			val := &ethpb.Validator{
				WithdrawalCredentials: make([]byte, 32),
				EffectiveBalance:      params.BeaconConfig().MaxEffectiveBalance,
				WithdrawableEpoch:     primitives.Epoch(1),
			}
			val.WithdrawalCredentials[0] = params.BeaconConfig().ETH1AddressWithdrawalPrefixByte
			require.NoError(t, s.validators.UpdateAt(s, uint64(i), val))
		}
		expected, err := s.ExpectedWithdrawals()
		require.NoError(t, err)
		require.Equal(t, 0, len(expected))
	})
	t.Run("one fully withdrawable", func(t *testing.T) {
		s := &BeaconState{
			version:                      version.Capella,
			validators:                   NewMultiValueValidators(make([]*ethpb.Validator, 100)),
			balances:                     NewMultiValueBalances(make([]uint64, 100)),
			nextWithdrawalValidatorIndex: 20,
		}
		for i := range s.validators.Value(s) {
			require.NoError(t, s.balances.UpdateAt(s, uint64(i), params.BeaconConfig().MaxEffectiveBalance))
			val := &ethpb.Validator{
				WithdrawalCredentials: make([]byte, 32),
				EffectiveBalance:      params.BeaconConfig().MaxEffectiveBalance,
				WithdrawableEpoch:     primitives.Epoch(1),
			}
			val.WithdrawalCredentials[0] = params.BeaconConfig().ETH1AddressWithdrawalPrefixByte
			require.NoError(t, s.validators.UpdateAt(s, uint64(i), val))
		}
		v, err := s.validators.At(s, 3)
		require.NoError(t, err)
		v.WithdrawableEpoch = primitives.Epoch(0)
		expected, err := s.ExpectedWithdrawals()
		require.NoError(t, err)
		require.Equal(t, 1, len(expected))
		amount, err := s.balances.At(s, 3)
		require.NoError(t, err)
		v, err = s.validators.At(s, 3)
		require.NoError(t, err)
		withdrawal := &enginev1.Withdrawal{
			Index:          0,
			ValidatorIndex: 3,
			Address:        v.WithdrawalCredentials[12:],
			Amount:         amount,
		}
		require.DeepEqual(t, withdrawal, expected[0])
	})
	t.Run("one partially withdrawable", func(t *testing.T) {
		s := &BeaconState{
			version:    version.Capella,
			validators: NewMultiValueValidators(make([]*ethpb.Validator, 100)),
			balances:   NewMultiValueBalances(make([]uint64, 100)),
		}
		for i := range s.validators.Value(s) {
			require.NoError(t, s.balances.UpdateAt(s, uint64(i), params.BeaconConfig().MaxEffectiveBalance))
			val := &ethpb.Validator{
				WithdrawalCredentials: make([]byte, 32),
				EffectiveBalance:      params.BeaconConfig().MaxEffectiveBalance,
				WithdrawableEpoch:     primitives.Epoch(1),
			}
			val.WithdrawalCredentials[0] = params.BeaconConfig().ETH1AddressWithdrawalPrefixByte
			require.NoError(t, s.validators.UpdateAt(s, uint64(i), val))
		}
		bal, err := s.balances.At(s, 3)
		require.NoError(t, err)
		require.NoError(t, s.balances.UpdateAt(s, 3, bal+params.BeaconConfig().MinDepositAmount))
		expected, err := s.ExpectedWithdrawals()
		require.NoError(t, err)
		require.Equal(t, 1, len(expected))
		v, err := s.validators.At(s, 3)
		require.NoError(t, err)
		withdrawal := &enginev1.Withdrawal{
			Index:          0,
			ValidatorIndex: 3,
			Address:        v.WithdrawalCredentials[12:],
			Amount:         params.BeaconConfig().MinDepositAmount,
		}
		require.DeepEqual(t, withdrawal, expected[0])
	})
	t.Run("one partially and one fully withdrawable", func(t *testing.T) {
		s := &BeaconState{
			version:    version.Capella,
			validators: NewMultiValueValidators(make([]*ethpb.Validator, 100)),
			balances:   NewMultiValueBalances(make([]uint64, 100)),
		}
		for i := range s.validators.Value(s) {
			require.NoError(t, s.balances.UpdateAt(s, uint64(i), params.BeaconConfig().MaxEffectiveBalance))
			val := &ethpb.Validator{
				WithdrawalCredentials: make([]byte, 32),
				EffectiveBalance:      params.BeaconConfig().MaxEffectiveBalance,
				WithdrawableEpoch:     primitives.Epoch(1),
			}
			val.WithdrawalCredentials[0] = params.BeaconConfig().ETH1AddressWithdrawalPrefixByte
			val.WithdrawalCredentials[31] = byte(i)
			require.NoError(t, s.validators.UpdateAt(s, uint64(i), val))
		}
		bal, err := s.balances.At(s, 3)
		require.NoError(t, err)
		require.NoError(t, s.balances.UpdateAt(s, 3, bal+params.BeaconConfig().MinDepositAmount))
		v, err := s.validators.At(s, 7)
		require.NoError(t, err)
		v.WithdrawableEpoch = primitives.Epoch(0)
		expected, err := s.ExpectedWithdrawals()
		require.NoError(t, err)
		require.Equal(t, 2, len(expected))

		amount, err := s.balances.At(s, 7)
		require.NoError(t, err)
		v, err = s.validators.At(s, 7)
		require.NoError(t, err)
		withdrawalFull := &enginev1.Withdrawal{
			Index:          1,
			ValidatorIndex: 7,
			Address:        v.WithdrawalCredentials[12:],
			Amount:         amount,
		}
		v, err = s.validators.At(s, 3)
		require.NoError(t, err)
		withdrawalPartial := &enginev1.Withdrawal{
			Index:          0,
			ValidatorIndex: 3,
			Address:        v.WithdrawalCredentials[12:],
			Amount:         params.BeaconConfig().MinDepositAmount,
		}
		require.DeepEqual(t, withdrawalPartial, expected[0])
		require.DeepEqual(t, withdrawalFull, expected[1])
	})
	t.Run("all partially withdrawable", func(t *testing.T) {
		s := &BeaconState{
			version:    version.Capella,
			validators: NewMultiValueValidators(make([]*ethpb.Validator, 100)),
			balances:   NewMultiValueBalances(make([]uint64, 100)),
		}
		for i := range s.validators.Value(s) {
			require.NoError(t, s.balances.UpdateAt(s, uint64(i), params.BeaconConfig().MaxEffectiveBalance+1))
			val := &ethpb.Validator{
				WithdrawalCredentials: make([]byte, 32),
				EffectiveBalance:      params.BeaconConfig().MaxEffectiveBalance,
				WithdrawableEpoch:     primitives.Epoch(1),
			}
			val.WithdrawalCredentials[0] = params.BeaconConfig().ETH1AddressWithdrawalPrefixByte
			require.NoError(t, s.validators.UpdateAt(s, uint64(i), val))
		}
		expected, err := s.ExpectedWithdrawals()
		require.NoError(t, err)
		require.Equal(t, params.BeaconConfig().MaxWithdrawalsPerPayload, uint64(len(expected)))
		v, err := s.validators.At(s, 0)
		require.NoError(t, err)
		withdrawal := &enginev1.Withdrawal{
			Index:          0,
			ValidatorIndex: 0,
			Address:        v.WithdrawalCredentials[12:],
			Amount:         1,
		}
		require.DeepEqual(t, withdrawal, expected[0])
	})
	t.Run("all fully withdrawable", func(t *testing.T) {
		s := &BeaconState{
			version:    version.Capella,
			validators: NewMultiValueValidators(make([]*ethpb.Validator, 100)),
			balances:   NewMultiValueBalances(make([]uint64, 100)),
		}
		for i := range s.validators.Value(s) {
			require.NoError(t, s.balances.UpdateAt(s, uint64(i), params.BeaconConfig().MaxEffectiveBalance))
			val := &ethpb.Validator{
				WithdrawalCredentials: make([]byte, 32),
				EffectiveBalance:      params.BeaconConfig().MaxEffectiveBalance,
				WithdrawableEpoch:     primitives.Epoch(0),
			}
			val.WithdrawalCredentials[0] = params.BeaconConfig().ETH1AddressWithdrawalPrefixByte
			require.NoError(t, s.validators.UpdateAt(s, uint64(i), val))
		}
		expected, err := s.ExpectedWithdrawals()
		require.NoError(t, err)
		require.Equal(t, params.BeaconConfig().MaxWithdrawalsPerPayload, uint64(len(expected)))
		v, err := s.validators.At(s, 0)
		require.NoError(t, err)
		withdrawal := &enginev1.Withdrawal{
			Index:          0,
			ValidatorIndex: 0,
			Address:        v.WithdrawalCredentials[12:],
			Amount:         params.BeaconConfig().MaxEffectiveBalance,
		}
		require.DeepEqual(t, withdrawal, expected[0])
	})
	t.Run("all fully and partially withdrawable", func(t *testing.T) {
		s := &BeaconState{
			version:    version.Capella,
			validators: NewMultiValueValidators(make([]*ethpb.Validator, 100)),
			balances:   NewMultiValueBalances(make([]uint64, 100)),
		}
		for i := range s.validators.Value(s) {
			require.NoError(t, s.balances.UpdateAt(s, uint64(i), params.BeaconConfig().MaxEffectiveBalance+1))
			val := &ethpb.Validator{
				WithdrawalCredentials: make([]byte, 32),
				EffectiveBalance:      params.BeaconConfig().MaxEffectiveBalance,
				WithdrawableEpoch:     primitives.Epoch(0),
			}
			val.WithdrawalCredentials[0] = params.BeaconConfig().ETH1AddressWithdrawalPrefixByte
			require.NoError(t, s.validators.UpdateAt(s, uint64(i), val))
		}
		expected, err := s.ExpectedWithdrawals()
		require.NoError(t, err)
		require.Equal(t, params.BeaconConfig().MaxWithdrawalsPerPayload, uint64(len(expected)))
		v, err := s.validators.At(s, 0)
		require.NoError(t, err)
		withdrawal := &enginev1.Withdrawal{
			Index:          0,
			ValidatorIndex: 0,
			Address:        v.WithdrawalCredentials[12:],
			Amount:         params.BeaconConfig().MaxEffectiveBalance + 1,
		}
		require.DeepEqual(t, withdrawal, expected[0])
	})
	t.Run("one fully withdrawable but zero balance", func(t *testing.T) {
		s := &BeaconState{
			version:                      version.Capella,
			validators:                   NewMultiValueValidators(make([]*ethpb.Validator, 100)),
			balances:                     NewMultiValueBalances(make([]uint64, 100)),
			nextWithdrawalValidatorIndex: 20,
		}
		for i := range s.validators.Value(s) {
			require.NoError(t, s.balances.UpdateAt(s, uint64(i), params.BeaconConfig().MaxEffectiveBalance))
			val := &ethpb.Validator{
				WithdrawalCredentials: make([]byte, 32),
				EffectiveBalance:      params.BeaconConfig().MaxEffectiveBalance,
				WithdrawableEpoch:     primitives.Epoch(1),
			}
			val.WithdrawalCredentials[0] = params.BeaconConfig().ETH1AddressWithdrawalPrefixByte
			require.NoError(t, s.validators.UpdateAt(s, uint64(i), val))
		}
		v, err := s.validators.At(s, 3)
		require.NoError(t, err)
		v.WithdrawableEpoch = primitives.Epoch(0)
		require.NoError(t, s.balances.UpdateAt(s, 3, 0))
		expected, err := s.ExpectedWithdrawals()
		require.NoError(t, err)
		require.Equal(t, 0, len(expected))
	})
	t.Run("one partially withdrawable, one above sweep bound", func(t *testing.T) {
		s := &BeaconState{
			version:    version.Capella,
			validators: NewMultiValueValidators(make([]*ethpb.Validator, 100)),
			balances:   NewMultiValueBalances(make([]uint64, 100)),
		}
		for i := range s.validators.Value(s) {
			require.NoError(t, s.balances.UpdateAt(s, uint64(i), params.BeaconConfig().MaxEffectiveBalance))
			val := &ethpb.Validator{
				WithdrawalCredentials: make([]byte, 32),
				EffectiveBalance:      params.BeaconConfig().MaxEffectiveBalance,
				WithdrawableEpoch:     primitives.Epoch(1),
			}
			val.WithdrawalCredentials[0] = params.BeaconConfig().ETH1AddressWithdrawalPrefixByte
			require.NoError(t, s.validators.UpdateAt(s, uint64(i), val))
		}
		bal, err := s.balances.At(s, 3)
		require.NoError(t, err)
		require.NoError(t, s.balances.UpdateAt(s, 3, bal+params.BeaconConfig().MinDepositAmount))
		bal, err = s.balances.At(s, 10)
		require.NoError(t, err)
		require.NoError(t, s.balances.UpdateAt(s, 10, bal+params.BeaconConfig().MinDepositAmount))
		saved := params.BeaconConfig().MaxValidatorsPerWithdrawalsSweep
		params.BeaconConfig().MaxValidatorsPerWithdrawalsSweep = 10
		expected, err := s.ExpectedWithdrawals()
		require.NoError(t, err)
		require.Equal(t, 1, len(expected))
		v, err := s.validators.At(s, 3)
		require.NoError(t, err)
		withdrawal := &enginev1.Withdrawal{
			Index:          0,
			ValidatorIndex: 3,
			Address:        v.WithdrawalCredentials[12:],
			Amount:         params.BeaconConfig().MinDepositAmount,
		}
		require.DeepEqual(t, withdrawal, expected[0])
		params.BeaconConfig().MaxValidatorsPerWithdrawalsSweep = saved
	})
}
