package protoarray

import (
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestNode_Getters(t *testing.T) {
	s := uint64(100)
	r := [32]byte{'a'}
	p := uint64(10)
	j := uint64(20)
	f := uint64(30)
	w := uint64(10000)
	bc := uint64(5)
	bd := uint64(4)
	g := [32]byte{'b'}
	n := &Node{
		slot:           s,
		root:           r,
		parent:         p,
		justifiedEpoch: j,
		finalizedEpoch: f,
		weight:         w,
		bestChild:      bc,
		bestDescendant: bd,
		graffiti:       g,
	}

	require.Equal(t, s, n.Slot())
	require.Equal(t, r, n.Root())
	require.Equal(t, p, n.Parent())
	require.Equal(t, j, n.JustifiedEpoch())
	require.Equal(t, f, n.FinalizedEpoch())
	require.Equal(t, w, n.Weight())
	require.Equal(t, bc, n.BestChild())
	require.Equal(t, bd, n.BestDescendant())
	require.Equal(t, g, n.Graffiti())
}
