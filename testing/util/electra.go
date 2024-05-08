package util

import (
	"math"
	"testing"

	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
)

// HackElectraMaxuint is helpful for tests that need to set up cases where the electra fork has passed.
// We have unit tests that assert our config matches the upstream config, where the next fork is always
// set to MaxUint64 until the fork epoch is formally set. This creates an issue for tests that want to
// work with slots that are defined to be after electra because converting the max epoch to a slot leads
// to multiplication overflow.
// Monkey patching tests with this function is the simplest workaround in these cases.
func HackElectraMaxuint(t *testing.T) func() {
	bc := params.MainnetConfig().Copy()
	bc.ElectraForkEpoch = math.MaxUint32
	undo, err := params.SetActiveWithUndo(bc)
	require.NoError(t, err)
	return func() {
		require.NoError(t, undo())
	}
}
