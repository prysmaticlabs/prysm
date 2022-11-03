package state_native

import (
	"testing"

	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/runtime/version"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
)

func TestSetNextWithdrawalIndex(t *testing.T) {
	s := BeaconState{
		version:             version.Capella,
		nextWithdrawalIndex: 3,
	}
	require.NoError(t, s.SetNextWithdrawalIndex(5))
	require.Equal(t, uint64(5), s.nextWithdrawalIndex)
}
func TestSetLastWithdrawalValidatorIndex(t *testing.T) {
	s := BeaconState{
		version:                      version.Capella,
		lastWithdrawalValidatorIndex: 3,
	}
	require.NoError(t, s.SetLastWithdrawalValidatorIndex(5))
	require.Equal(t, types.ValidatorIndex(5), s.lastWithdrawalValidatorIndex)
}
