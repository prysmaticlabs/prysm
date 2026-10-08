package decoupled

import (
	"testing"

	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	"github.com/OffchainLabs/prysm/v7/testing/require"
)

func setHeight(n *Node, h primitives.Height) { n.FG.Target.Height = h }

func TestViableNeedsWitnessAtMaxHeightMinusOne(t *testing.T) {
	st := newStore(t)
	g := st.Node(root(0))
	a := st.addChild(root(1), g, 1)
	setHeight(a, 10)
	a2 := st.addChild(root(2), a, 2)
	setHeight(a2, 10)
	b := st.addChild(root(3), g, 1)
	setHeight(b, 11)
	st.hMax = 12
	require.Equal(t, false, st.viable(a, everyNode), "highest processed descendant stays at 10")
	require.Equal(t, true, st.viable(b, everyNode))
	require.Equal(t, true, st.viable(g, everyNode), "genesis has b as witness")
	require.Equal(t, false, st.viable(g, func(n *Node) bool { return n != b }), "a witness outside the allowed view does not count")
}

func TestViableTruncatesAtZero(t *testing.T) {
	st := newStore(t)
	a := st.addChild(root(1), st.Node(root(0)), 1)
	setHeight(a, 0)
	st.hMax = 1
	require.Equal(t, true, st.viable(a, everyNode))
	sc := &supportCounts{subtree: map[*Node]int{}}
	require.Equal(t, false, st.eligible(sc, 5)(a), "the height clause is false when h_max <= 1 and there is no majority")
}

func TestFGRootAndFilteredTree(t *testing.T) {
	st := newStore(t)
	g := st.Node(root(0))
	j := st.addChild(root(1), g, 1)
	setHeight(j, 5)
	k := st.addChild(root(2), j, 2)
	setHeight(k, 6)
	other := st.addChild(root(3), g, 1)
	setHeight(other, 6)
	st.justified, st.hJ, st.hMax = j, 5, 6
	require.Equal(t, j, st.fgRoot(), "h_max = h_j + 1 selects J")
	in := st.candidates(everyNode)
	require.Equal(t, true, in(j))
	require.Equal(t, true, in(k))
	require.Equal(t, false, in(other), "viable but not below the FG root")
	setHeight(k, 5)
	st.hMax = 7
	require.Equal(t, g, st.fgRoot(), "otherwise F")
	in = st.candidates(everyNode)
	require.Equal(t, false, in(j), "no descendant of j reaches height 6")
	require.Equal(t, true, in(other))
}

func TestEligibleClauses(t *testing.T) {
	st := newStore(t)
	g := st.Node(root(0))
	setHeight(g, 3)
	low := st.addChild(root(1), g, 1)
	setHeight(low, 3)
	deep := st.addChild(root(2), low, 2)
	setHeight(deep, 5)
	st.hMax = 6
	sc := &supportCounts{participants: 4, subtree: map[*Node]int{low: 1, deep: 1}}
	el := st.eligible(sc, 9)
	require.Equal(t, true, el(low), "parent height 3 < h_max-1 = 5, no majority needed")
	require.Equal(t, true, el(deep), "parent low is at height 3 < 5, no majority needed either")
	setHeight(low, 5)
	require.Equal(t, false, el(deep), "a parent inside the band needs a majority, 4 < 2*1 fails")
	deep.Slot = 9
	require.Equal(t, true, el(deep), "current-slot exception")
}
