package decoupled

import (
	"testing"

	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	"github.com/OffchainLabs/prysm/v7/testing/require"
)

func TestScheduleInstants(t *testing.T) {
	st := newStore(t)
	require.Equal(t, clock(2, 0), st.slotStart(2))
	require.Equal(t, clock(2, 1), st.voteTime(2))
	require.Equal(t, clock(2, 2), st.confirmationCutoff(2))
	require.Equal(t, clock(2, 3), st.supportCutoff(2))
	require.Equal(t, clock(2, 6), st.evaluationTime(2))
	require.Equal(t, st.confirmationCutoff(3), st.evaluationTime(2))
	st.at(clock(2, 0))
	require.Equal(t, primitives.Slot(2), st.currentSlot())
	st.at(clock(2, 3))
	require.Equal(t, primitives.Slot(2), st.currentSlot())
}

func TestNewSlotKeepsPreviousSlotPool(t *testing.T) {
	st := newStore(t)
	st.pool[1] = []poolEntry{{vote: Vote{Validator: 1, Slot: 1, Root: root(0)}, at: clock(1, 1)}}
	st.pool[2] = []poolEntry{{vote: Vote{Validator: 1, Slot: 2, Root: root(0)}, at: clock(2, 1)}}
	st.pool[3] = []poolEntry{{vote: Vote{Validator: 1, Slot: 3, Root: root(0)}, at: clock(3, 1)}}
	st.NewSlot(3)
	require.Equal(t, 0, len(st.pool[1]))
	require.Equal(t, 1, len(st.pool[2]))
	require.Equal(t, 1, len(st.pool[3]))
}
