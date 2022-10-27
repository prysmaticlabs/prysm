package capella

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state/stateutil"
	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	enginev1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/runtime/version"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
)

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
	s := state_native.BeaconState{
		version:             version.Capella,
		nextWithdrawalIndex: 2,
		withdrawalQueue:     make([]*enginev1.Withdrawal, 2),
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
	require.Equal(t, 3, len(s.withdrawalQueue))
	withdrawal := s.withdrawalQueue[2]
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
	require.Equal(t, 5, len(s.withdrawalQueue))
	withdrawal = s.withdrawalQueue[4]
	require.Equal(t, params.BeaconConfig().MaxEffectiveBalance, withdrawal.Amount)
}
