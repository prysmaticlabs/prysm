package doublylinkedtree

import (
	"context"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestStore_PruneThreshold(t *testing.T) {
	s := &Store{
		pruneThreshold: defaultPruneThreshold,
	}
	if got := s.PruneThreshold(); got != defaultPruneThreshold {
		t.Errorf("PruneThreshold() = %v, want %v", got, defaultPruneThreshold)
	}
}

func TestStore_JustifiedEpoch(t *testing.T) {
	j := types.Epoch(100)
	f := setup(j, j)
	require.Equal(t, j, f.JustifiedEpoch())
}

func TestStore_FinalizedEpoch(t *testing.T) {
	j := types.Epoch(50)
	f := setup(j, j)
	require.Equal(t, j, f.FinalizedEpoch())
}

func TestStore_NodeCount(t *testing.T) {
	f := setup(0, 0)
	require.NoError(t, f.ProcessBlock(context.Background(), 1, indexToHash(1), params.BeaconConfig().ZeroHash, 0, 0, false))
	require.Equal(t, 2, f.NodeCount())
}

func TestStore_NodeByRoot(t *testing.T) {
	f := setup(0, 0)
	require.NoError(t, f.ProcessBlock(context.Background(), 1, indexToHash(1), params.BeaconConfig().ZeroHash, 0, 0, false))
	require.NoError(t, f.ProcessBlock(context.Background(), 2, indexToHash(2), indexToHash(1), 0, 0, false))
	node0 := f.store.treeRootNode
	node1 := node0.children[0]
	node2 := node1.children[0]

	expectedRoots := map[[32]byte]*Node{
		params.BeaconConfig().ZeroHash: node0,
		indexToHash(1):                 node1,
		indexToHash(2):                 node2,
	}

	require.Equal(t, 3, f.NodeCount())
	for root, node := range f.store.nodeByRoot {
		v, ok := expectedRoots[root]
		require.Equal(t, ok, true)
		require.Equal(t, v, node)
	}
}

func TestForkChoice_HasNode(t *testing.T) {
	f := setup(0, 0)
	require.NoError(t, f.ProcessBlock(context.Background(), 1, indexToHash(1), params.BeaconConfig().ZeroHash, 0, 0, false))
	require.Equal(t, true, f.HasNode(indexToHash(1)))
}

func TestStore_Head_UnknownJustifiedRoot(t *testing.T) {
	f := setup(0, 0)

	_, err := f.store.head(context.Background(), [32]byte{'a'})
	assert.ErrorContains(t, errUnknownJustifiedRoot.Error(), err)
}

func TestStore_Head_Itself(t *testing.T) {
	f := setup(0, 0)
	require.NoError(t, f.ProcessBlock(context.Background(), 1, indexToHash(1), params.BeaconConfig().ZeroHash, 0, 0, false))

	// Since the justified node does not have a best descendant so the best node
	// is itself.
	h, err := f.store.head(context.Background(), indexToHash(1))
	require.NoError(t, err)
	assert.Equal(t, indexToHash(1), h)
}

func TestStore_Head_BestDescendant(t *testing.T) {
	f := setup(0, 0)
	require.NoError(t, f.ProcessBlock(context.Background(), 1, indexToHash(1), params.BeaconConfig().ZeroHash, 0, 0, false))
	require.NoError(t, f.ProcessBlock(context.Background(), 2, indexToHash(2), indexToHash(1), 0, 0, false))
	require.NoError(t, f.ProcessBlock(context.Background(), 3, indexToHash(3), indexToHash(1), 0, 0, false))
	require.NoError(t, f.ProcessBlock(context.Background(), 4, indexToHash(4), indexToHash(2), 0, 0, false))
	h, err := f.store.head(context.Background(), indexToHash(1))
	require.NoError(t, err)
	require.Equal(t, h, indexToHash(4))
}

func TestStore_UpdateBestDescendant_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	f := setup(0, 0)
	require.NoError(t, f.ProcessBlock(ctx, 1, indexToHash(1), params.BeaconConfig().ZeroHash, 0, 0, false))
	cancel()
	err := f.ProcessBlock(ctx, 2, indexToHash(2), indexToHash(1), 0, 0, false)
	require.ErrorContains(t, "context canceled", err)
}

func TestStore_Insert(t *testing.T) {
	// The new node does not have a parent.
	treeRootNode := &Node{slot: 0, root: indexToHash(0)}
	nodeByRoot := map[[32]byte]*Node{indexToHash(0): treeRootNode}
	s := &Store{nodeByRoot: nodeByRoot, treeRootNode: treeRootNode}
	require.NoError(t, s.insert(context.Background(), 100, indexToHash(100), indexToHash(0), 1, 1, false))
	assert.Equal(t, 2, len(s.nodeByRoot), "Did not insert block")
	assert.Equal(t, (*Node)(nil), treeRootNode.parent, "Incorrect parent")
	assert.Equal(t, 1, len(treeRootNode.children), "Incorrect children number")
	child := treeRootNode.children[0]
	assert.Equal(t, types.Epoch(1), child.justifiedEpoch, "Incorrect justification")
	assert.Equal(t, types.Epoch(1), child.finalizedEpoch, "Incorrect finalization")
	assert.Equal(t, indexToHash(100), child.root, "Incorrect root")
}

func TestStore_updateCheckpoints(t *testing.T) {
	f := setup(0, 0)
	s := f.store

	s.updateCheckpoints(1, 1)
	assert.Equal(t, types.Epoch(1), s.justifiedEpoch, "Did not update justified epoch")
	assert.Equal(t, types.Epoch(1), s.finalizedEpoch, "Did not update finalized epoch")
}

func TestStore_Prune_LessThanThreshold(t *testing.T) {
	// Define 100 nodes in store.
	numOfNodes := uint64(100)
	f := setup(0, 0)
	ctx := context.Background()
	require.NoError(t, f.ProcessBlock(ctx, 1, indexToHash(1), params.BeaconConfig().ZeroHash, 0, 0, false))
	for i := uint64(2); i < numOfNodes; i++ {
		require.NoError(t, f.ProcessBlock(ctx, types.Slot(i), indexToHash(i), indexToHash(i-1), 0, 0, false))
	}

	s := f.store
	s.pruneThreshold = 100

	// Finalized root has depth 99 so everything before it should be pruned,
	// but PruneThreshold is at 100 so nothing will be pruned.
	require.NoError(t, s.prune(context.Background(), indexToHash(99)))
	assert.Equal(t, 100, len(s.nodeByRoot), "Incorrect nodes count")
}

func TestStore_Prune_MoreThanThreshold(t *testing.T) {
	// Define 100 nodes in store.
	numOfNodes := uint64(100)
	f := setup(0, 0)
	ctx := context.Background()
	require.NoError(t, f.ProcessBlock(ctx, 1, indexToHash(1), params.BeaconConfig().ZeroHash, 0, 0, false))
	for i := uint64(2); i < numOfNodes; i++ {
		require.NoError(t, f.ProcessBlock(ctx, types.Slot(i), indexToHash(i), indexToHash(i-1), 0, 0, false))
	}

	s := f.store
	s.pruneThreshold = 0

	// Finalized root is at index 99 so everything before 99 should be pruned.
	require.NoError(t, s.prune(context.Background(), indexToHash(99)))
	assert.Equal(t, 1, len(s.nodeByRoot), "Incorrect nodes count")
}

func TestStore_Prune_MoreThanOnce(t *testing.T) {
	// Define 100 nodes in store.
	numOfNodes := uint64(100)
	f := setup(0, 0)
	ctx := context.Background()
	require.NoError(t, f.ProcessBlock(ctx, 1, indexToHash(1), params.BeaconConfig().ZeroHash, 0, 0, false))
	for i := uint64(2); i < numOfNodes; i++ {
		require.NoError(t, f.ProcessBlock(ctx, types.Slot(i), indexToHash(i), indexToHash(i-1), 0, 0, false))
	}

	s := f.store
	s.pruneThreshold = 0

	// Finalized root is at index 11 so everything before 11 should be pruned.
	require.NoError(t, s.prune(context.Background(), indexToHash(10)))
	assert.Equal(t, 90, len(s.nodeByRoot), "Incorrect nodes count")

	// One more time.
	require.NoError(t, s.prune(context.Background(), indexToHash(20)))
	assert.Equal(t, 80, len(s.nodeByRoot), "Incorrect nodes count")
}

// This unit tests starts with a simple branch like this
//
//       - 1
//     /
// -- 0 -- 2
//
// And we finalize 1. As a result only 1 should survive
func TestStore_Prune_NoDanglingBranch(t *testing.T) {
	f := setup(0, 0)
	ctx := context.Background()
	require.NoError(t, f.ProcessBlock(ctx, 1, indexToHash(1), params.BeaconConfig().ZeroHash, 0, 0, false))
	require.NoError(t, f.ProcessBlock(ctx, 2, indexToHash(2), params.BeaconConfig().ZeroHash, 0, 0, false))
	f.store.pruneThreshold = 0

	s := f.store
	require.NoError(t, s.prune(context.Background(), indexToHash(1)))
	require.Equal(t, len(s.nodeByRoot), 1)
}

// This test starts with the following branching diagram
/// We start with the following diagram
//
//                E -- F
//               /
//         C -- D
//        /      \
//  A -- B        G -- H -- I
//        \        \
//         J        -- K -- L
//
//
func TestStore_tips(t *testing.T) {
	ctx := context.Background()
	f := setup(1, 1)

	require.NoError(t, f.ProcessBlock(ctx, 100, [32]byte{'a'}, params.BeaconConfig().ZeroHash, 1, 1, true))
	require.NoError(t, f.ProcessBlock(ctx, 101, [32]byte{'b'}, [32]byte{'a'}, 1, 1, true))
	require.NoError(t, f.ProcessBlock(ctx, 102, [32]byte{'c'}, [32]byte{'b'}, 1, 1, true))
	require.NoError(t, f.ProcessBlock(ctx, 102, [32]byte{'j'}, [32]byte{'b'}, 1, 1, true))
	require.NoError(t, f.ProcessBlock(ctx, 103, [32]byte{'d'}, [32]byte{'c'}, 1, 1, true))
	require.NoError(t, f.ProcessBlock(ctx, 104, [32]byte{'e'}, [32]byte{'d'}, 1, 1, true))
	require.NoError(t, f.ProcessBlock(ctx, 104, [32]byte{'g'}, [32]byte{'d'}, 1, 1, true))
	require.NoError(t, f.ProcessBlock(ctx, 105, [32]byte{'f'}, [32]byte{'e'}, 1, 1, true))
	require.NoError(t, f.ProcessBlock(ctx, 105, [32]byte{'h'}, [32]byte{'g'}, 1, 1, true))
	require.NoError(t, f.ProcessBlock(ctx, 105, [32]byte{'k'}, [32]byte{'g'}, 1, 1, true))
	require.NoError(t, f.ProcessBlock(ctx, 106, [32]byte{'i'}, [32]byte{'h'}, 1, 1, true))
	require.NoError(t, f.ProcessBlock(ctx, 106, [32]byte{'l'}, [32]byte{'k'}, 1, 1, true))
	expectedMap := map[[32]byte]types.Slot{
		[32]byte{'f'}: 105,
		[32]byte{'i'}: 106,
		[32]byte{'l'}: 106,
		[32]byte{'j'}: 102,
	}
	roots, slots := f.store.tips()
	for i, r := range roots {
		expectedSlot, ok := expectedMap[r]
		require.Equal(t, true, ok)
		require.Equal(t, slots[i], expectedSlot)
	}
}

func TestStore_PruneMapsNodes(t *testing.T) {
	f := setup(0, 0)
	ctx := context.Background()
	require.NoError(t, f.ProcessBlock(ctx, 1, indexToHash(1), params.BeaconConfig().ZeroHash, 0, 0, false))
	require.NoError(t, f.ProcessBlock(ctx, 2, indexToHash(2), params.BeaconConfig().ZeroHash, 0, 0, false))

	s := f.store
	s.pruneThreshold = 0
	require.NoError(t, s.prune(context.Background(), indexToHash(uint64(1))))
	require.Equal(t, len(s.nodeByRoot), 1)

}

func TestStore_HasParent(t *testing.T) {
	f := setup(1, 1)
	ctx := context.Background()
	require.NoError(t, f.ProcessBlock(ctx, 1, indexToHash(1), params.BeaconConfig().ZeroHash, 1, 1, false))
	require.NoError(t, f.ProcessBlock(ctx, 2, indexToHash(2), indexToHash(1), 1, 1, false))
	require.NoError(t, f.ProcessBlock(ctx, 3, indexToHash(3), indexToHash(2), 1, 1, false))
	require.Equal(t, false, f.HasParent(params.BeaconConfig().ZeroHash))
	require.Equal(t, true, f.HasParent(indexToHash(1)))
	require.Equal(t, true, f.HasParent(indexToHash(2)))
	require.Equal(t, true, f.HasParent(indexToHash(3)))
	require.Equal(t, false, f.HasParent(indexToHash(4)))
}
