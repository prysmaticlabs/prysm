package transition_test

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/transition"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
	"github.com/prysmaticlabs/prysm/v4/testing/util"
)

func TestTrailingSlotState_RoundTrip(t *testing.T) {
	ctx := context.Background()
	r := []byte{'a'}
	s := transition.NextSlotState(r)
	require.Equal(t, nil, s)

	s, _ = util.DeterministicGenesisState(t, 1)
	require.NoError(t, transition.UpdateNextSlotCache(ctx, r, s))
	s = transition.NextSlotState(r)
	require.Equal(t, primitives.Slot(1), s.Slot())

	lastRoot, lastState := transition.LastCachedPair()
	require.DeepEqual(t, r, lastRoot)
	require.Equal(t, s.Slot(), lastState.Slot())

	require.NoError(t, transition.UpdateNextSlotCache(ctx, r, s))
	s = transition.NextSlotState(r)
	require.Equal(t, primitives.Slot(2), s.Slot())

	lastRoot, lastState = transition.LastCachedPair()
	require.DeepEqual(t, r, lastRoot)
	require.Equal(t, s.Slot(), lastState.Slot())
}
