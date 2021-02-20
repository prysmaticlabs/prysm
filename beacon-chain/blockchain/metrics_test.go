package blockchain

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestReportEpochMetrics_BadHeadState(t *testing.T) {
	s, err := testutil.NewBeaconState()
	require.NoError(t, err)
	h, err := testutil.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, h.SetValidators(nil))
	err = reportEpochMetrics(context.Background(), s, h)
	require.ErrorContains(t, "failed to initialize precompute: nil validators in state", err)
}
