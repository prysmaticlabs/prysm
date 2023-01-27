package state

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/state/types"
	"github.com/prysmaticlabs/prysm/v3/runtime/version"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
)

func TestSetNextWithdrawalIndex(t *testing.T) {
	s := State{
		version:             version.Capella,
		nextWithdrawalIndex: 3,
		dirtyFields:         make(map[types.FieldIndex]bool),
	}
	require.NoError(t, s.SetNextWithdrawalIndex(5))
	require.Equal(t, uint64(5), s.nextWithdrawalIndex)
	require.Equal(t, true, s.dirtyFields[types.NextWithdrawalIndex])
}

func TestSetLastWithdrawalValidatorIndex(t *testing.T) {
	s := State{
		version:                      version.Capella,
		nextWithdrawalValidatorIndex: 3,
		dirtyFields:                  make(map[types.FieldIndex]bool),
	}
	require.NoError(t, s.SetNextWithdrawalValidatorIndex(5))
	require.Equal(t, primitives.ValidatorIndex(5), s.nextWithdrawalValidatorIndex)
	require.Equal(t, true, s.dirtyFields[types.NextWithdrawalValidatorIndex])
}
