package protoarray

import (
	"context"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestNode_Getters(t *testing.T) {
	slot := types.Slot(100)
	root := [32]byte{'a'}
	parent := &Node{}
	jEpoch := types.Epoch(20)
	fEpoch := types.Epoch(30)
	weight := uint64(10000)
	balance := uint64(10)
	n := &Node{
		slot:           slot,
		root:           root,
		parent:         parent,
		justifiedEpoch: jEpoch,
		finalizedEpoch: fEpoch,
		weight:         weight,
		balance:        balance,
	}

	require.Equal(t, slot, n.Slot())
	require.Equal(t, root, n.Root())
	require.Equal(t, parent, n.Parent())
	require.Equal(t, jEpoch, n.JustifiedEpoch())
	require.Equal(t, fEpoch, n.FinalizedEpoch())
	require.Equal(t, weight, n.Weight())
	require.Equal(t, nil, n.BestDescendant())
}

func TestNode_ApplyWeightChanges_PositiveChange(t *testing.T) {
	f := setup(0, 0)
	ctx := context.Background()
	require.NoError(t, f.ProcessBlock(ctx, 1, indexToHash(1), params.BeaconConfig().ZeroHash, 0, 0, false))
	require.NoError(t, f.ProcessBlock(ctx, 2, indexToHash(2), indexToHash(1), 0, 0, false))
	require.NoError(t, f.ProcessBlock(ctx, 3, indexToHash(3), indexToHash(2), 0, 0, false))

	// The updated balances of each node is 100
	s := f.store
	s.nodeByRoot[indexToHash(1)].balance = 100
	s.nodeByRoot[indexToHash(2)].balance = 100
	s.nodeByRoot[indexToHash(3)].balance = 100

	assert.NoError(t, s.treeRoot.applyWeightChanges(ctx))

	assert.Equal(t, uint64(300), s.nodeByRoot[indexToHash(1)].weight)
	assert.Equal(t, uint64(200), s.nodeByRoot[indexToHash(2)].weight)
	assert.Equal(t, uint64(100), s.nodeByRoot[indexToHash(3)].weight)
}

func TestNode_ApplyWeightChanges_NegativeChange(t *testing.T) {
	f := setup(0, 0)
	ctx := context.Background()
	require.NoError(t, f.ProcessBlock(ctx, 1, indexToHash(1), params.BeaconConfig().ZeroHash, 0, 0, false))
	require.NoError(t, f.ProcessBlock(ctx, 2, indexToHash(2), indexToHash(1), 0, 0, false))
	require.NoError(t, f.ProcessBlock(ctx, 3, indexToHash(3), indexToHash(2), 0, 0, false))

	// The updated balances of each node is 100
	s := f.store
	s.nodeByRoot[indexToHash(1)].weight = 400
	s.nodeByRoot[indexToHash(2)].weight = 400
	s.nodeByRoot[indexToHash(3)].weight = 400

	s.nodeByRoot[indexToHash(1)].balance = 100
	s.nodeByRoot[indexToHash(2)].balance = 100
	s.nodeByRoot[indexToHash(3)].balance = 100

	assert.NoError(t, s.treeRoot.applyWeightChanges(ctx))

	assert.Equal(t, uint64(300), s.nodeByRoot[indexToHash(1)].weight)
	assert.Equal(t, uint64(200), s.nodeByRoot[indexToHash(2)].weight)
	assert.Equal(t, uint64(100), s.nodeByRoot[indexToHash(3)].weight)
}

func TestNode_UpdateBestDescendant_NonViableChild(t *testing.T) {
	f := setup(1, 1)
	ctx := context.Background()
	// Input child is not viable.
	require.NoError(t, f.ProcessBlock(ctx, 1, indexToHash(1), params.BeaconConfig().ZeroHash, 0, 1, false))

	// Verify parent's best child and best descendant are `none`.
	s := f.store
	assert.Equal(t, 1, len(s.treeRoot.children))
	assert.Equal(t, nil, s.treeRoot.bestDescendant)
}

func TestNode_UpdateBestDescendant_ViableChild(t *testing.T) {
	f := setup(1, 1)
	ctx := context.Background()
	// Input child is best descendant
	require.NoError(t, f.ProcessBlock(ctx, 1, indexToHash(1), params.BeaconConfig().ZeroHash, 1, 1, false))

	s := f.store
	assert.Equal(t, 1, len(s.treeRoot.children))
	assert.Equal(t, s.treeRoot.children[0], s.treeRoot.bestDescendant)
}

func TestNode_UpdateBestDescendant_HigherWeightChild(t *testing.T) {
	f := setup(1, 1)
	ctx := context.Background()
	// Input child is best descendant
	require.NoError(t, f.ProcessBlock(ctx, 1, indexToHash(1), params.BeaconConfig().ZeroHash, 1, 1, false))
	require.NoError(t, f.ProcessBlock(ctx, 2, indexToHash(2), params.BeaconConfig().ZeroHash, 1, 1, false))

	s := f.store
	s.nodeByRoot[indexToHash(1)].weight = 100
	s.nodeByRoot[indexToHash(2)].weight = 200
	assert.NoError(t, s.treeRoot.updateBestDescendant(ctx, 1, 1))

	assert.Equal(t, 2, len(s.treeRoot.children))
	assert.Equal(t, s.treeRoot.children[1], s.treeRoot.bestDescendant)
}

func TestNode_UpdateBestDescendant_LowerWeightChild(t *testing.T) {
	f := setup(1, 1)
	ctx := context.Background()
	// Input child is best descendant
	require.NoError(t, f.ProcessBlock(ctx, 1, indexToHash(1), params.BeaconConfig().ZeroHash, 1, 1, false))
	require.NoError(t, f.ProcessBlock(ctx, 2, indexToHash(2), params.BeaconConfig().ZeroHash, 1, 1, false))

	s := f.store
	s.nodeByRoot[indexToHash(1)].weight = 200
	s.nodeByRoot[indexToHash(2)].weight = 100
	assert.NoError(t, s.treeRoot.updateBestDescendant(ctx, 1, 1))

	assert.Equal(t, 2, len(s.treeRoot.children))
	assert.Equal(t, s.treeRoot.children[0], s.treeRoot.bestDescendant)
}

func TestNode_TestDepth(t *testing.T) {
	f := setup(1, 1)
	ctx := context.Background()
	// Input child is best descendant
	require.NoError(t, f.ProcessBlock(ctx, 1, indexToHash(1), params.BeaconConfig().ZeroHash, 1, 1, false))
	require.NoError(t, f.ProcessBlock(ctx, 2, indexToHash(2), indexToHash(1), 1, 1, false))
	require.NoError(t, f.ProcessBlock(ctx, 3, indexToHash(3), params.BeaconConfig().ZeroHash, 1, 1, false))

	s := f.store
	require.Equal(t, s.nodeByRoot[indexToHash(2)].depth(), 2)
	require.Equal(t, s.nodeByRoot[indexToHash(3)].depth(), 1)
}

func TestNode_ViableForHead(t *testing.T) {
	tests := []struct {
		n              *Node
		justifiedEpoch types.Epoch
		finalizedEpoch types.Epoch
		want           bool
	}{
		{&Node{}, 0, 0, true},
		{&Node{}, 1, 0, false},
		{&Node{}, 0, 1, false},
		{&Node{finalizedEpoch: 1, justifiedEpoch: 1}, 1, 1, true},
		{&Node{finalizedEpoch: 1, justifiedEpoch: 1}, 2, 2, false},
		{&Node{finalizedEpoch: 3, justifiedEpoch: 4}, 4, 3, true},
	}
	for _, tc := range tests {
		got := tc.n.viableForHead(tc.justifiedEpoch, tc.finalizedEpoch)
		assert.Equal(t, tc.want, got)
	}
}

func TestStore_LeadsToViableHead(t *testing.T) {
	f := setup(4, 3)
	ctx := context.Background()
	require.NoError(t, f.ProcessBlock(ctx, 1, indexToHash(1), params.BeaconConfig().ZeroHash, 1, 1, false))
	require.NoError(t, f.ProcessBlock(ctx, 2, indexToHash(2), params.BeaconConfig().ZeroHash, 1, 1, false))
	require.NoError(t, f.ProcessBlock(ctx, 3, indexToHash(3), indexToHash(1), 1, 1, false))
	require.NoError(t, f.ProcessBlock(ctx, 4, indexToHash(4), indexToHash(2), 1, 1, false))
	require.NoError(t, f.ProcessBlock(ctx, 5, indexToHash(5), indexToHash(3), 4, 3, false))

	require.Equal(t, true, f.store.treeRoot.leadsToViableHead(4, 3))
	require.Equal(t, true, f.store.nodeByRoot[indexToHash(5)].leadsToViableHead(4, 3))
	require.Equal(t, false, f.store.nodeByRoot[indexToHash(2)].leadsToViableHead(4, 3))
	require.Equal(t, false, f.store.nodeByRoot[indexToHash(4)].leadsToViableHead(4, 3))
}
