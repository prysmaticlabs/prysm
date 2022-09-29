package state_native

import (
	"testing"

	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	enginev1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	"github.com/prysmaticlabs/prysm/v3/runtime/version"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
)

func TestWithdrawalQueue(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		ws := []*enginev1.Withdrawal{
			{
				WithdrawalIndex:  0,
				ExecutionAddress: []byte("address1"),
				Amount:           1,
			},
			{
				WithdrawalIndex:  1,
				ExecutionAddress: []byte("address2"),
				Amount:           2,
			},
		}
		s := BeaconState{version: version.Capella, withdrawalQueue: ws}
		q, err := s.WithdrawalQueue()
		require.NoError(t, err)
		assert.DeepEqual(t, ws, q)
	})
	t.Run("version before Capella not supported", func(t *testing.T) {
		s := BeaconState{version: version.Bellatrix}
		_, err := s.WithdrawalQueue()
		assert.ErrorContains(t, "WithdrawalQueue is not supported", err)
	})
}

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

func TestNextPartialWithdrawalValidatorIndex(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		s := BeaconState{version: version.Capella, nextPartialWithdrawalValidatorIndex: 123}
		i, err := s.NextPartialWithdrawalValidatorIndex()
		require.NoError(t, err)
		assert.Equal(t, types.ValidatorIndex(123), i)
	})
	t.Run("version before Capella not supported", func(t *testing.T) {
		s := BeaconState{version: version.Bellatrix}
		_, err := s.NextPartialWithdrawalValidatorIndex()
		assert.ErrorContains(t, "NextPartialWithdrawalValidatorIndex is not supported", err)
	})
}
