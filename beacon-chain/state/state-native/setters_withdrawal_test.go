package state_native

import (
	"testing"

	nativetypes "github.com/prysmaticlabs/prysm/v3/beacon-chain/state/state-native/types"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state/stateutil"
	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	enginev1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/runtime/version"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
)

func TestSetWithdrawalQueue(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		oldQ := []*enginev1.Withdrawal{
			{
				WithdrawalIndex:  0,
				ExecutionAddress: []byte("address1"),
				Amount:           1,
				ValidatorIndex:   2,
			},
			{
				WithdrawalIndex:  1,
				ExecutionAddress: []byte("address2"),
				Amount:           2,
				ValidatorIndex:   3,
			},
		}
		newQ := []*enginev1.Withdrawal{
			{
				WithdrawalIndex:  2,
				ExecutionAddress: []byte("address3"),
				Amount:           3,
				ValidatorIndex:   4,
			},
			{
				WithdrawalIndex:  3,
				ExecutionAddress: []byte("address4"),
				Amount:           4,
				ValidatorIndex:   5,
			},
		}
		s := BeaconState{
			version:               version.Capella,
			withdrawalQueue:       oldQ,
			sharedFieldReferences: map[nativetypes.FieldIndex]*stateutil.Reference{nativetypes.WithdrawalQueue: stateutil.NewRef(1)},
			dirtyFields:           map[nativetypes.FieldIndex]bool{},
			rebuildTrie:           map[nativetypes.FieldIndex]bool{},
		}
		err := s.SetWithdrawalQueue(newQ)
		require.NoError(t, err)
		assert.DeepEqual(t, newQ, s.withdrawalQueue)
	})
	t.Run("version before Capella not supported", func(t *testing.T) {
		s := BeaconState{version: version.Bellatrix}
		err := s.SetWithdrawalQueue([]*enginev1.Withdrawal{})
		assert.ErrorContains(t, "SetWithdrawalQueue is not supported", err)
	})
}

func TestAppendWithdrawal(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		oldWithdrawal1 := &enginev1.Withdrawal{
			WithdrawalIndex:  0,
			ExecutionAddress: []byte("address1"),
			Amount:           1,
			ValidatorIndex:   2,
		}
		oldWithdrawal2 := &enginev1.Withdrawal{
			WithdrawalIndex:  1,
			ExecutionAddress: []byte("address2"),
			Amount:           2,
			ValidatorIndex:   3,
		}
		q := []*enginev1.Withdrawal{oldWithdrawal1, oldWithdrawal2}
		s := BeaconState{
			version:               version.Capella,
			withdrawalQueue:       q,
			sharedFieldReferences: map[nativetypes.FieldIndex]*stateutil.Reference{nativetypes.WithdrawalQueue: stateutil.NewRef(1)},
			dirtyFields:           map[nativetypes.FieldIndex]bool{},
			dirtyIndices:          map[nativetypes.FieldIndex][]uint64{},
			rebuildTrie:           map[nativetypes.FieldIndex]bool{},
		}
		newWithdrawal := &enginev1.Withdrawal{
			WithdrawalIndex:  2,
			ExecutionAddress: []byte("address3"),
			Amount:           3,
			ValidatorIndex:   4,
		}
		err := s.AppendWithdrawal(newWithdrawal)
		require.NoError(t, err)
		expectedQ := []*enginev1.Withdrawal{oldWithdrawal1, oldWithdrawal2, newWithdrawal}
		assert.DeepEqual(t, expectedQ, s.withdrawalQueue)
	})
	t.Run("version before Capella not supported", func(t *testing.T) {
		s := BeaconState{version: version.Bellatrix}
		err := s.AppendWithdrawal(&enginev1.Withdrawal{})
		assert.ErrorContains(t, "AppendWithdrawal is not supported", err)
	})
}

func TestIncreaseNextWithdrawalIndex(t *testing.T) {
	s := BeaconState{
		version:             version.Capella,
		nextWithdrawalIndex: 2,
	}
	require.NoError(t, s.IncreaseNextWithdrawalIndex())
	require.Equal(t, uint64(3), s.nextWithdrawalIndex)
}

func TestWithdrawBalance(t *testing.T) {
	creds := make([]byte, fieldparams.RootLength)
	creds[0] = params.BeaconConfig().ETH1AddressWithdrawalPrefixByte
	val := &ethpb.Validator{
		WithdrawalCredentials: creds,
	}

	creds2 := make([]byte, fieldparams.RootLength)
	val2 := &ethpb.Validator{
		WithdrawalCredentials: creds2,
	}

	vals := []*ethpb.Validator{val, val2}
	s := BeaconState{
		version:             version.Capella,
		nextWithdrawalIndex: 2,
		validators:          vals,
		balances: []uint64{
			params.BeaconConfig().MaxEffectiveBalance + params.BeaconConfig().MinDepositAmount,
			params.BeaconConfig().MaxEffectiveBalance,
		},
		sharedFieldReferences: map[nativetypes.FieldIndex]*stateutil.Reference{
			nativetypes.WithdrawalQueue: stateutil.NewRef(1),
			nativetypes.Balances:        stateutil.NewRef(1),
		},
		dirtyFields:  map[nativetypes.FieldIndex]bool{},
		dirtyIndices: map[nativetypes.FieldIndex][]uint64{},
		rebuildTrie:  map[nativetypes.FieldIndex]bool{},
	}
	require.NoError(t, s.WithdrawBalance(0, params.BeaconConfig().MinDepositAmount))
	require.Equal(t, params.BeaconConfig().MaxEffectiveBalance, s.balances[0])
	require.Equal(t, uint64(3), s.nextWithdrawalIndex)
	require.Equal(t, 1, len(s.withdrawalQueue))
	withdrawal := s.withdrawalQueue[0]
	require.Equal(t, uint64(2), withdrawal.WithdrawalIndex)
	require.Equal(t, params.BeaconConfig().MinDepositAmount, withdrawal.Amount)
	require.Equal(t, types.ValidatorIndex(0), withdrawal.ValidatorIndex)

	// BLS validator
	err := s.WithdrawBalance(1, params.BeaconConfig().MinDepositAmount)
	require.ErrorContains(t, "invalid withdrawal credentials", err)

	// Sucessive withdrawals is fine:
	err = s.WithdrawBalance(0, params.BeaconConfig().MinDepositAmount)
	require.NoError(t, err)

	// Underflow produces wrong amount (Spec Repo #3054)
	err = s.WithdrawBalance(0, params.BeaconConfig().MaxEffectiveBalance)
	require.NoError(t, err)
	require.Equal(t, 3, len(s.withdrawalQueue))
	withdrawal = s.withdrawalQueue[2]
	require.Equal(t, params.BeaconConfig().MaxEffectiveBalance, withdrawal.Amount)
}
