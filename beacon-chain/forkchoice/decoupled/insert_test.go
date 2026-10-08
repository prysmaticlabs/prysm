package decoupled

import (
	"testing"

	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	"github.com/OffchainLabs/prysm/v7/testing/require"
)

func fg(h, hj primitives.Height, j [32]byte, hf primitives.Height, f [32]byte) FGSummary {
	return FGSummary{Target: Pair{Height: h}, Justified: Pair{Height: hj, Root: j}, Finalized: Pair{Height: hf, Root: f}}
}

var base = fg(1, 0, root(0), 0, root(0))

func TestInsertNodeDuplicateAndUnknownParent(t *testing.T) {
	st := newStore(t)
	st.at(clock(1, 0))
	require.ErrorIs(t, st.InsertNode(Block{Root: root(1), Parent: root(9), Slot: 1, FG: base}), ErrUnknownParent)
	require.NoError(t, st.InsertNode(Block{Root: root(1), Parent: root(0), Slot: 1, FG: base}))
	st.at(clock(1, 1))
	require.NoError(t, st.InsertNode(Block{Root: root(1), Parent: root(0), Slot: 1, FG: base}), "duplicate is a no-op")
	require.Equal(t, clock(1, 0), st.Node(root(1)).processed, "receipt time is write once")
	require.Equal(t, 1, len(st.Node(root(0)).Children))
}

func TestInsertNodePoolsBlockVotesAtProcessingTime(t *testing.T) {
	st := newStore(t)
	st.at(clock(2, 0))
	v := Vote{Validator: 1, Slot: 1, Root: root(0)}
	require.NoError(t, st.InsertNode(Block{Root: root(1), Parent: root(0), Slot: 2, BlockVotes: []Vote{v}, FG: base}))
	at, ok := st.receivedAt(v)
	require.Equal(t, true, ok)
	require.Equal(t, clock(2, 0), at)
	require.DeepEqual(t, []Vote{v}, st.Node(root(1)).BlockVotes)
}

func TestUpdateFinalityOrder(t *testing.T) {
	st := newStore(t)
	st.at(clock(10, 0))
	require.NoError(t, st.InsertNode(Block{Root: root(1), Parent: root(0), Slot: 1, FG: fg(2, 1, root(1), 0, root(0))}))
	require.Equal(t, primitives.Height(2), st.MaxHeight())
	require.Equal(t, st.Node(root(1)), st.Justified(), "J moves to (1, root1) which beats (0, genesis)")
	require.Equal(t, primitives.Height(1), st.JustifiedHeight())
	require.NoError(t, st.InsertNode(Block{Root: root(2), Parent: root(1), Slot: 2, FG: fg(3, 2, root(2), 1, root(1))}))
	require.Equal(t, st.Node(root(2)), st.Justified())
	require.Equal(t, st.Node(root(1)), st.Finalized(), "root1 is a strict descendant of old F, an ancestor of J, and viable")
	require.NoError(t, st.InsertNode(Block{Root: root(3), Parent: root(2), Slot: 3, FG: fg(3, 2, root(2), 1, root(1))}))
	require.Equal(t, st.Node(root(2)), st.Justified(), "same (h_j, root) does not move J")
}
