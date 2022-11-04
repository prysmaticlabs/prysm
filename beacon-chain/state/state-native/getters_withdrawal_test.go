package state_native

import (
	"testing"

	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
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
		s := BeaconState{version: version.Capella, lastWithdrawalValidatorIndex: 123}
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
