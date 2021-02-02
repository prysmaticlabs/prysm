package state_test

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	st "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestTrailingSlotState_RoundTrip(t *testing.T) {
	ctx := context.Background()
	r := []byte{'a'}
	s, err := state.NextSlotState(ctx, r)
	require.NoError(t, err)
	require.Equal(t, (*st.BeaconState)(nil), s)

	s, _ = testutil.DeterministicGenesisState(t, 1)
	require.NoError(t, state.UpdateNextSlotCache(ctx, r, s))
	s, err = state.NextSlotState(ctx, r)
	require.NoError(t, err)
	require.Equal(t, uint64(1), s.Slot())

	require.NoError(t, state.UpdateNextSlotCache(ctx, r, s))
	s, err = state.NextSlotState(ctx, r)
	require.NoError(t, err)
	require.Equal(t, uint64(2), s.Slot())
}
