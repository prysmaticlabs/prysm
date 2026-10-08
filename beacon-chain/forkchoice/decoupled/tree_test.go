package decoupled

import (
	"testing"
	"time"

	"github.com/OffchainLabs/prysm/v7/config/params"
	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	"github.com/OffchainLabs/prysm/v7/testing/require"
)

var genesis = time.Unix(1_000_000, 0)

func root(i byte) [32]byte { return [32]byte{i} }

// clock returns the instant deltas quarter-slots into slot s.
func clock(s primitives.Slot, deltas int) time.Time {
	d := params.BeaconConfig().SlotDuration()
	return genesis.Add(time.Duration(s)*d + time.Duration(deltas)*d/4)
}

func newStore(t *testing.T) *Store {
	st := New(root(0))
	st.SetGenesisTime(genesis)
	st.now = func() time.Time { t.Fatal("test did not set the clock"); return time.Time{} }
	return st
}

func (st *Store) at(tm time.Time) { st.now = func() time.Time { return tm } }

func TestNewStoreHoldsGenesis(t *testing.T) {
	st := newStore(t)
	g := st.Node(root(0))
	require.NotNil(t, g)
	require.Equal(t, primitives.Slot(0), g.Slot)
	require.Equal(t, g, st.finalized)
	require.Equal(t, g, st.justified)
	require.Equal(t, primitives.Height(1), st.hMax)
	require.Equal(t, primitives.Height(0), st.hJ)
	require.Equal(t, true, g.processed.IsZero())
}

func TestIsAncestorReflexiveAndTransitive(t *testing.T) {
	st := newStore(t)
	a := st.addChild(root(1), st.Node(root(0)), 1)
	b := st.addChild(root(2), a, 2)
	c := st.addChild(root(3), a, 3)
	require.Equal(t, true, isAncestor(a, a))
	require.Equal(t, true, isAncestor(st.Node(root(0)), b))
	require.Equal(t, true, st.IsAncestor(root(1), root(2)))
	require.Equal(t, false, isAncestor(b, c))
	require.Equal(t, false, st.IsAncestor(root(2), root(1)))
	require.Equal(t, false, st.IsAncestor(root(9), root(1)))
}
