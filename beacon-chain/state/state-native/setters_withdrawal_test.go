package state_native

import (
	"testing"

	nativetypes "github.com/prysmaticlabs/prysm/v3/beacon-chain/state/state-native/types"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/runtime/version"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
)

func TestSetNextWithdrawalIndex(t *testing.T) {
	s := BeaconState{
		version:             version.Capella,
		nextWithdrawalIndex: 3,
		dirtyFields:         make(map[nativetypes.FieldIndex]bool),
	}
	require.NoError(t, s.SetNextWithdrawalIndex(5))
	require.Equal(t, uint64(5), s.nextWithdrawalIndex)
	require.Equal(t, true, s.dirtyFields[nativetypes.NextWithdrawalIndex])
}

func TestSetNextWithdrawalValidatorIndex(t *testing.T) {
	s := BeaconState{
		version:                      version.Capella,
		nextWithdrawalValidatorIndex: 3,
		dirtyFields:                  make(map[nativetypes.FieldIndex]bool),
	}
	require.NoError(t, s.SetNextWithdrawalValidatorIndex(5))
	require.Equal(t, types.ValidatorIndex(5), s.nextWithdrawalValidatorIndex)
	require.Equal(t, true, s.dirtyFields[nativetypes.NextWithdrawalValidatorIndex])
}
