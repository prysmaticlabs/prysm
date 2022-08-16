package protoarray

import (
	"testing"

	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
)

func TestNode_Getters(t *testing.T) {
	slot := types.Slot(100)
	root := [32]byte{'a'}
	parent := uint64(10)
	jEpoch := types.Epoch(20)
	fEpoch := types.Epoch(30)
	weight := uint64(10000)
	bestChild := uint64(5)
	bestDescendant := uint64(4)
	n := &Node{
		slot:           slot,
		root:           root,
		parent:         parent,
		justifiedEpoch: jEpoch,
		finalizedEpoch: fEpoch,
		weight:         weight,
		bestChild:      bestChild,
		bestDescendant: bestDescendant,
	}

	require.Equal(t, slot, n.Slot())
	require.Equal(t, root, n.Root())
	require.Equal(t, parent, n.Parent())
	require.Equal(t, jEpoch, n.JustifiedEpoch())
	require.Equal(t, fEpoch, n.FinalizedEpoch())
	require.Equal(t, weight, n.Weight())
	require.Equal(t, bestChild, n.BestChild())
	require.Equal(t, bestDescendant, n.BestDescendant())
}
