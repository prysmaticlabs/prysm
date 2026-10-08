package decoupled

import (
	"testing"

	"github.com/OffchainLabs/prysm/v7/testing/require"
)

func TestScoreCountsOncePerValidatorAndEquivocatorsEverywhere(t *testing.T) {
	st := newStore(t)
	g := st.Node(root(0))
	a := st.addChild(root(1), g, 1)
	b := st.addChild(root(2), g, 1)
	c := st.addChild(root(3), a, 2)
	v := view{participants: []Vote{
		vote(1, 2, root(3)),
		vote(2, 2, root(1)),
		vote(3, 2, root(2)),
		vote(4, 2, root(1)),
		vote(4, 2, root(2)),
		vote(5, 2, root(9)),
	}}
	v.supported = v.participants
	sc := st.countSupport(v, 2)
	require.Equal(t, 1, sc.equivocators)
	require.Equal(t, 5, sc.participants)
	require.Equal(t, 1+2, sc.score(a), "validators 1 and 2 under a, plus the equivocator")
	require.Equal(t, 1+1, sc.score(b))
	require.Equal(t, 1+1, sc.score(c))
	require.Equal(t, 1+0, sc.score(st.addChild(root(4), b, 2)))
	require.Equal(t, true, sc.hasMajority(a), "5 < 2*3")
	require.Equal(t, false, sc.hasMajority(b), "5 < 2*2 fails")
}

func TestScoreIgnoresOtherSlotAndLaterRoot(t *testing.T) {
	st := newStore(t)
	a := st.addChild(root(1), st.Node(root(0)), 3)
	v := view{participants: []Vote{vote(1, 2, root(1)), vote(2, 1, root(1))}}
	v.supported = v.participants
	sc := st.countSupport(v, 2)
	require.Equal(t, 0, sc.score(a), "block slot 3 > vote slot 2, and the slot-1 vote is not slot 2")
	require.Equal(t, 2, sc.participants)
}

func TestGhostArgmaxTiesByGreatestRoot(t *testing.T) {
	st := newStore(t)
	g := st.Node(root(0))
	a := st.addChild(root(1), g, 1)
	b := st.addChild(root(2), g, 1)
	c := st.addChild(root(3), b, 2)
	score := map[*Node]int{a: 2, b: 2, c: 1}
	byScore := func(n *Node) int { return score[n] }
	require.Equal(t, c, ghost(g, everyNode, byScore, everyNode), "b wins the tie by root, then descends to c")
	require.Equal(t, a, ghost(g, everyNode, byScore, func(n *Node) bool { return n != b }))
	require.Equal(t, a, ghost(g, func(n *Node) bool { return n == a }, byScore, everyNode), "membership is tested on the child only")
	require.Equal(t, c, ghost(c, everyNode, func(*Node) int { return 0 }, everyNode))
}
