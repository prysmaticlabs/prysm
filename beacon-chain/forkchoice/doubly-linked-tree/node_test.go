package doublylinkedtree

import (
	"context"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestNode_ApplyWeightChanges_PositiveChange(t *testing.T) {
	f := setup(0, 0)
	ctx := context.Background()
	require.NoError(t, f.InsertOptimisticBlock(ctx, 1, indexToHash(1), params.BeaconConfig().ZeroHash, 0, 0))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 2, indexToHash(2), indexToHash(1), 0, 0))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 3, indexToHash(3), indexToHash(2), 0, 0))

	// The updated balances of each node is 100
	s := f.store

	s.nodesLock.Lock()
	defer s.nodesLock.Unlock()
	s.nodeByRoot[indexToHash(1)].balance = 100
	s.nodeByRoot[indexToHash(2)].balance = 100
	s.nodeByRoot[indexToHash(3)].balance = 100

	assert.NoError(t, s.treeRootNode.applyWeightChanges(ctx))

	assert.Equal(t, uint64(300), s.nodeByRoot[indexToHash(1)].weight)
	assert.Equal(t, uint64(200), s.nodeByRoot[indexToHash(2)].weight)
	assert.Equal(t, uint64(100), s.nodeByRoot[indexToHash(3)].weight)
}

func TestNode_ApplyWeightChanges_NegativeChange(t *testing.T) {
	f := setup(0, 0)
	ctx := context.Background()
	require.NoError(t, f.InsertOptimisticBlock(ctx, 1, indexToHash(1), params.BeaconConfig().ZeroHash, 0, 0))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 2, indexToHash(2), indexToHash(1), 0, 0))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 3, indexToHash(3), indexToHash(2), 0, 0))

	// The updated balances of each node is 100
	s := f.store
	s.nodesLock.Lock()
	defer s.nodesLock.Unlock()
	s.nodeByRoot[indexToHash(1)].weight = 400
	s.nodeByRoot[indexToHash(2)].weight = 400
	s.nodeByRoot[indexToHash(3)].weight = 400

	s.nodeByRoot[indexToHash(1)].balance = 100
	s.nodeByRoot[indexToHash(2)].balance = 100
	s.nodeByRoot[indexToHash(3)].balance = 100

	assert.NoError(t, s.treeRootNode.applyWeightChanges(ctx))

	assert.Equal(t, uint64(300), s.nodeByRoot[indexToHash(1)].weight)
	assert.Equal(t, uint64(200), s.nodeByRoot[indexToHash(2)].weight)
	assert.Equal(t, uint64(100), s.nodeByRoot[indexToHash(3)].weight)
}

func TestNode_UpdateBestDescendant_NonViableChild(t *testing.T) {
	f := setup(1, 1)
	ctx := context.Background()
	// Input child is not viable.
	require.NoError(t, f.InsertOptimisticBlock(ctx, 1, indexToHash(1), params.BeaconConfig().ZeroHash, 2, 3))

	// Verify parent's best child and best descendant are `none`.
	s := f.store
	assert.Equal(t, 1, len(s.treeRootNode.children))
	nilBestDescendant := s.treeRootNode.bestDescendant == nil
	assert.Equal(t, true, nilBestDescendant)
}

func TestNode_UpdateBestDescendant_ViableChild(t *testing.T) {
	f := setup(1, 1)
	ctx := context.Background()
	// Input child is best descendant
	require.NoError(t, f.InsertOptimisticBlock(ctx, 1, indexToHash(1), params.BeaconConfig().ZeroHash, 1, 1))

	s := f.store
	assert.Equal(t, 1, len(s.treeRootNode.children))
	assert.Equal(t, s.treeRootNode.children[0], s.treeRootNode.bestDescendant)
}

func TestNode_UpdateBestDescendant_HigherWeightChild(t *testing.T) {
	f := setup(1, 1)
	ctx := context.Background()
	// Input child is best descendant
	require.NoError(t, f.InsertOptimisticBlock(ctx, 1, indexToHash(1), params.BeaconConfig().ZeroHash, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 2, indexToHash(2), params.BeaconConfig().ZeroHash, 1, 1))

	s := f.store
	s.nodeByRoot[indexToHash(1)].weight = 100
	s.nodeByRoot[indexToHash(2)].weight = 200
	assert.NoError(t, s.treeRootNode.updateBestDescendant(ctx, 1, 1))

	assert.Equal(t, 2, len(s.treeRootNode.children))
	assert.Equal(t, s.treeRootNode.children[1], s.treeRootNode.bestDescendant)
}

func TestNode_UpdateBestDescendant_LowerWeightChild(t *testing.T) {
	f := setup(1, 1)
	ctx := context.Background()
	// Input child is best descendant
	require.NoError(t, f.InsertOptimisticBlock(ctx, 1, indexToHash(1), params.BeaconConfig().ZeroHash, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 2, indexToHash(2), params.BeaconConfig().ZeroHash, 1, 1))

	s := f.store
	s.nodeByRoot[indexToHash(1)].weight = 200
	s.nodeByRoot[indexToHash(2)].weight = 100
	assert.NoError(t, s.treeRootNode.updateBestDescendant(ctx, 1, 1))

	assert.Equal(t, 2, len(s.treeRootNode.children))
	assert.Equal(t, s.treeRootNode.children[0], s.treeRootNode.bestDescendant)
}

func TestNode_TestDepth(t *testing.T) {
	f := setup(1, 1)
	ctx := context.Background()
	// Input child is best descendant
	require.NoError(t, f.InsertOptimisticBlock(ctx, 1, indexToHash(1), params.BeaconConfig().ZeroHash, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 2, indexToHash(2), indexToHash(1), 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 3, indexToHash(3), params.BeaconConfig().ZeroHash, 1, 1))

	s := f.store
	require.Equal(t, s.nodeByRoot[indexToHash(2)].depth(), uint64(2))
	require.Equal(t, s.nodeByRoot[indexToHash(3)].depth(), uint64(1))
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

func TestNode_LeadsToViableHead(t *testing.T) {
	f := setup(4, 3)
	ctx := context.Background()
	require.NoError(t, f.InsertOptimisticBlock(ctx, 1, indexToHash(1), params.BeaconConfig().ZeroHash, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 2, indexToHash(2), params.BeaconConfig().ZeroHash, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 3, indexToHash(3), indexToHash(1), 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 4, indexToHash(4), indexToHash(2), 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 5, indexToHash(5), indexToHash(3), 4, 3))

	require.Equal(t, true, f.store.treeRootNode.leadsToViableHead(4, 3))
	require.Equal(t, true, f.store.nodeByRoot[indexToHash(5)].leadsToViableHead(4, 3))
	require.Equal(t, false, f.store.nodeByRoot[indexToHash(2)].leadsToViableHead(4, 3))
	require.Equal(t, false, f.store.nodeByRoot[indexToHash(4)].leadsToViableHead(4, 3))
}

func TestNode_SetFullyValidated(t *testing.T) {
	f := setup(1, 1)
	ctx := context.Background()
	// insert blocks in the fork pattern (optimistic status in parenthesis)
	//
	// 0 (false) -- 1 (false) -- 2 (false) -- 3 (true) -- 4 (true)
	//               \
	//                 -- 5 (true)
	//
	require.NoError(t, f.InsertOptimisticBlock(ctx, 1, indexToHash(1), params.BeaconConfig().ZeroHash, 1, 1))
	require.NoError(t, f.SetOptimisticToValid(ctx, params.BeaconConfig().ZeroHash))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 2, indexToHash(2), indexToHash(1), 1, 1))
	require.NoError(t, f.SetOptimisticToValid(ctx, indexToHash(1)))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 3, indexToHash(3), indexToHash(2), 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 4, indexToHash(4), indexToHash(3), 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 5, indexToHash(5), indexToHash(1), 1, 1))

	opt, err := f.IsOptimistic(ctx, indexToHash(5))
	require.NoError(t, err)
	require.Equal(t, true, opt)

	opt, err = f.IsOptimistic(ctx, indexToHash(4))
	require.NoError(t, err)
	require.Equal(t, true, opt)

	require.NoError(t, f.store.nodeByRoot[indexToHash(4)].setNodeAndParentValidated(ctx))

	// block 5 should still be optimistic
	opt, err = f.IsOptimistic(ctx, indexToHash(5))
	require.NoError(t, err)
	require.Equal(t, true, opt)

	// block 4 and 3 should now be valid
	opt, err = f.IsOptimistic(ctx, indexToHash(4))
	require.NoError(t, err)
	require.Equal(t, false, opt)

	opt, err = f.IsOptimistic(ctx, indexToHash(3))
	require.NoError(t, err)
	require.Equal(t, false, opt)
}
