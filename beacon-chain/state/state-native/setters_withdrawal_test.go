package state_native

import (
	"testing"

	nativetypes "github.com/prysmaticlabs/prysm/v3/beacon-chain/state/state-native/types"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state/stateutil"
	enginev1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
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
