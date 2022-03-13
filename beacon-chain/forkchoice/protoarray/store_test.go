package protoarray

import (
	"context"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
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

func TestForkChoice_HasNode(t *testing.T) {
	nodeIndices := map[[32]byte]uint64{
		{'a'}: 1,
		{'b'}: 2,
	}
	s := &Store{
		nodesIndices: nodeIndices,
	}
	f := &ForkChoice{store: s}
	require.Equal(t, true, f.HasNode([32]byte{'a'}))
}

func TestStore_Head_UnknownJustifiedRoot(t *testing.T) {
	s := &Store{nodesIndices: make(map[[32]byte]uint64)}

	_, err := s.head(context.Background(), [32]byte{})
	assert.ErrorContains(t, errUnknownJustifiedRoot.Error(), err)
}

func TestStore_Head_UnknownJustifiedIndex(t *testing.T) {
	r := [32]byte{'A'}
	indices := make(map[[32]byte]uint64)
	indices[r] = 1
	s := &Store{nodesIndices: indices}

	_, err := s.head(context.Background(), r)
	assert.ErrorContains(t, errInvalidJustifiedIndex.Error(), err)
}

func TestStore_Head_Itself(t *testing.T) {
	r := [32]byte{'A'}
	indices := map[[32]byte]uint64{r: 0}

	// Since the justified node does not have a best descendant so the best node
	// is itself.
	s := &Store{nodesIndices: indices, nodes: []*Node{{root: r, parent: NonExistentNode, bestDescendant: NonExistentNode}}, canonicalNodes: make(map[[32]byte]bool)}
	h, err := s.head(context.Background(), r)
	require.NoError(t, err)
	assert.Equal(t, r, h)
}

func TestStore_Head_BestDescendant(t *testing.T) {
	r := [32]byte{'A'}
	best := [32]byte{'B'}
	indices := map[[32]byte]uint64{r: 0, best: 1}

	// Since the justified node's best descendant is at index 1, and its root is `best`,
	// the head should be `best`.
	s := &Store{nodesIndices: indices, nodes: []*Node{{root: r, bestDescendant: 1, parent: NonExistentNode}, {root: best, parent: 0}}, canonicalNodes: make(map[[32]byte]bool)}
	h, err := s.head(context.Background(), r)
	require.NoError(t, err)
	assert.Equal(t, best, h)
}

func TestStore_Head_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	r := [32]byte{'A'}
	best := [32]byte{'B'}
	indices := map[[32]byte]uint64{r: 0, best: 1}

	s := &Store{nodesIndices: indices, nodes: []*Node{{root: r, parent: NonExistentNode, bestDescendant: 1}, {root: best, parent: 0}}, canonicalNodes: make(map[[32]byte]bool)}
	cancel()
	_, err := s.head(ctx, r)
	require.ErrorContains(t, "context canceled", err)
}

func TestStore_Insert_UnknownParent(t *testing.T) {
	// The new node does not have a parent.
	s := &Store{nodesIndices: make(map[[32]byte]uint64)}
	require.NoError(t, s.insert(context.Background(), 100, [32]byte{'A'}, [32]byte{'B'}, 1, 1))
	assert.Equal(t, 1, len(s.nodes), "Did not insert block")
	assert.Equal(t, 1, len(s.nodesIndices), "Did not insert block")
	assert.Equal(t, NonExistentNode, s.nodes[0].parent, "Incorrect parent")
	assert.Equal(t, types.Epoch(1), s.nodes[0].justifiedEpoch, "Incorrect justification")
	assert.Equal(t, types.Epoch(1), s.nodes[0].finalizedEpoch, "Incorrect finalization")
	assert.Equal(t, [32]byte{'A'}, s.nodes[0].root, "Incorrect root")
}

func TestStore_Insert_KnownParent(t *testing.T) {
	// Similar to UnknownParent test, but this time the new node has a valid parent already in store.
	// The new node builds on top of the parent.
	s := &Store{nodesIndices: make(map[[32]byte]uint64)}
	s.nodes = []*Node{{}}
	p := [32]byte{'B'}
	s.nodesIndices[p] = 0
	require.NoError(t, s.insert(context.Background(), 100, [32]byte{'A'}, p, 1, 1))
	assert.Equal(t, 2, len(s.nodes), "Did not insert block")
	assert.Equal(t, 2, len(s.nodesIndices), "Did not insert block")
	assert.Equal(t, uint64(0), s.nodes[1].parent, "Incorrect parent")
	assert.Equal(t, types.Epoch(1), s.nodes[1].justifiedEpoch, "Incorrect justification")
	assert.Equal(t, types.Epoch(1), s.nodes[1].finalizedEpoch, "Incorrect finalization")
	assert.Equal(t, [32]byte{'A'}, s.nodes[1].root, "Incorrect root")
}

func TestStore_ApplyScoreChanges_InvalidDeltaLength(t *testing.T) {
	s := &Store{}

	// This will fail because node indices has length of 0, and delta list has a length of 1.
	err := s.applyWeightChanges(context.Background(), 0, 0, []uint64{}, []int{1})
	assert.ErrorContains(t, errInvalidDeltaLength.Error(), err)
}

func TestStore_ApplyScoreChanges_UpdateEpochs(t *testing.T) {
	s := &Store{}

	// The justified and finalized epochs in Store should be updated to 1 and 1 given the following input.
	require.NoError(t, s.applyWeightChanges(context.Background(), 1, 1, []uint64{}, []int{}))
	assert.Equal(t, types.Epoch(1), s.justifiedEpoch, "Did not update justified epoch")
	assert.Equal(t, types.Epoch(1), s.finalizedEpoch, "Did not update finalized epoch")
}

func TestStore_ApplyScoreChanges_UpdateWeightsPositiveDelta(t *testing.T) {
	// Construct 3 nodes with weight 100 on each node. The 3 nodes linked to each other.
	s := &Store{nodes: []*Node{
		{root: [32]byte{'A'}, weight: 100},
		{root: [32]byte{'A'}, weight: 100},
		{parent: 1, root: [32]byte{'A'}, weight: 100}}}

	// Each node gets one unique vote. The weight should look like 103 <- 102 <- 101 because
	// they get propagated back.
	require.NoError(t, s.applyWeightChanges(context.Background(), 0, 0, []uint64{}, []int{1, 1, 1}))
	assert.Equal(t, uint64(103), s.nodes[0].weight)
	assert.Equal(t, uint64(102), s.nodes[1].weight)
	assert.Equal(t, uint64(101), s.nodes[2].weight)
}

func TestStore_ApplyScoreChanges_UpdateWeightsNegativeDelta(t *testing.T) {
	// Construct 3 nodes with weight 100 on each node. The 3 nodes linked to each other.
	s := &Store{nodes: []*Node{
		{root: [32]byte{'A'}, weight: 100},
		{root: [32]byte{'A'}, weight: 100},
		{parent: 1, root: [32]byte{'A'}, weight: 100}}}

	// Each node gets one unique vote which contributes to negative delta.
	// The weight should look like 97 <- 98 <- 99 because they get propagated back.
	require.NoError(t, s.applyWeightChanges(context.Background(), 0, 0, []uint64{}, []int{-1, -1, -1}))
	assert.Equal(t, uint64(97), s.nodes[0].weight)
	assert.Equal(t, uint64(98), s.nodes[1].weight)
	assert.Equal(t, uint64(99), s.nodes[2].weight)
}

func TestStore_ApplyScoreChanges_UpdateWeightsMixedDelta(t *testing.T) {
	// Construct 3 nodes with weight 100 on each node. The 3 nodes linked to each other.
	s := &Store{nodes: []*Node{
		{root: [32]byte{'A'}, weight: 100},
		{root: [32]byte{'A'}, weight: 100},
		{parent: 1, root: [32]byte{'A'}, weight: 100}}}

	// Each node gets one mixed vote. The weight should look like 100 <- 200 <- 250.
	require.NoError(t, s.applyWeightChanges(context.Background(), 0, 0, []uint64{}, []int{-100, -50, 150}))
	assert.Equal(t, uint64(100), s.nodes[0].weight)
	assert.Equal(t, uint64(200), s.nodes[1].weight)
	assert.Equal(t, uint64(250), s.nodes[2].weight)
}

func TestStore_UpdateBestChildAndDescendant_RemoveChild(t *testing.T) {
	// Make parent's best child equal's to input child index and child is not viable.
	s := &Store{nodes: []*Node{{bestChild: 1}, {}}, justifiedEpoch: 1, finalizedEpoch: 1}
	require.NoError(t, s.updateBestChildAndDescendant(0, 1))

	// Verify parent's best child and best descendant are `none`.
	assert.Equal(t, NonExistentNode, s.nodes[0].bestChild, "Did not get correct best child index")
	assert.Equal(t, NonExistentNode, s.nodes[0].bestDescendant, "Did not get correct best descendant index")
}

func TestStore_UpdateBestChildAndDescendant_UpdateDescendant(t *testing.T) {
	// Make parent's best child equal to child index and child is viable.
	s := &Store{nodes: []*Node{{bestChild: 1}, {bestDescendant: NonExistentNode}}}
	require.NoError(t, s.updateBestChildAndDescendant(0, 1))

	// Verify parent's best child is the same and best descendant is not set to child index.
	assert.Equal(t, uint64(1), s.nodes[0].bestChild, "Did not get correct best child index")
	assert.Equal(t, uint64(1), s.nodes[0].bestDescendant, "Did not get correct best descendant index")
}

func TestStore_UpdateBestChildAndDescendant_ChangeChildByViability(t *testing.T) {
	// Make parent's best child not equal to child index, child leads to viable index and
	// parent's best child doesn't lead to viable index.
	s := &Store{
		justifiedEpoch: 1,
		finalizedEpoch: 1,
		nodes: []*Node{{bestChild: 1, justifiedEpoch: 1, finalizedEpoch: 1},
			{bestDescendant: NonExistentNode},
			{bestDescendant: NonExistentNode, justifiedEpoch: 1, finalizedEpoch: 1}}}
	require.NoError(t, s.updateBestChildAndDescendant(0, 2))

	// Verify parent's best child and best descendant are set to child index.
	assert.Equal(t, uint64(2), s.nodes[0].bestChild, "Did not get correct best child index")
	assert.Equal(t, uint64(2), s.nodes[0].bestDescendant, "Did not get correct best descendant index")
}

func TestStore_UpdateBestChildAndDescendant_ChangeChildByWeight(t *testing.T) {
	// Make parent's best child not equal to child index, child leads to viable index and
	// parents best child leads to viable index but child has more weight than parent's best child.
	s := &Store{
		justifiedEpoch: 1,
		finalizedEpoch: 1,
		nodes: []*Node{{bestChild: 1, justifiedEpoch: 1, finalizedEpoch: 1},
			{bestDescendant: NonExistentNode, justifiedEpoch: 1, finalizedEpoch: 1},
			{bestDescendant: NonExistentNode, justifiedEpoch: 1, finalizedEpoch: 1, weight: 1}}}
	require.NoError(t, s.updateBestChildAndDescendant(0, 2))

	// Verify parent's best child and best descendant are set to child index.
	assert.Equal(t, uint64(2), s.nodes[0].bestChild, "Did not get correct best child index")
	assert.Equal(t, uint64(2), s.nodes[0].bestDescendant, "Did not get correct best descendant index")
}

func TestStore_UpdateBestChildAndDescendant_ChangeChildAtLeaf(t *testing.T) {
	// Make parent's best child to none and input child leads to viable index.
	s := &Store{
		justifiedEpoch: 1,
		finalizedEpoch: 1,
		nodes: []*Node{{bestChild: NonExistentNode, justifiedEpoch: 1, finalizedEpoch: 1},
			{bestDescendant: NonExistentNode, justifiedEpoch: 1, finalizedEpoch: 1},
			{bestDescendant: NonExistentNode, justifiedEpoch: 1, finalizedEpoch: 1}}}
	require.NoError(t, s.updateBestChildAndDescendant(0, 2))

	// Verify parent's best child and best descendant are set to child index.
	assert.Equal(t, uint64(2), s.nodes[0].bestChild, "Did not get correct best child index")
	assert.Equal(t, uint64(2), s.nodes[0].bestDescendant, "Did not get correct best descendant index")
}

func TestStore_UpdateBestChildAndDescendant_NoChangeByViability(t *testing.T) {
	// Make parent's best child not equal to child index, child leads to not viable index and
	// parents best child leads to viable index.
	s := &Store{
		justifiedEpoch: 1,
		finalizedEpoch: 1,
		nodes: []*Node{{bestChild: 1, justifiedEpoch: 1, finalizedEpoch: 1},
			{bestDescendant: NonExistentNode, justifiedEpoch: 1, finalizedEpoch: 1},
			{bestDescendant: NonExistentNode}}}
	require.NoError(t, s.updateBestChildAndDescendant(0, 2))

	// Verify parent's best child and best descendant are not changed.
	assert.Equal(t, uint64(1), s.nodes[0].bestChild, "Did not get correct best child index")
	assert.Equal(t, uint64(0), s.nodes[0].bestDescendant, "Did not get correct best descendant index")
}

func TestStore_UpdateBestChildAndDescendant_NoChangeByWeight(t *testing.T) {
	// Make parent's best child not equal to child index, child leads to viable index and
	// parents best child leads to viable index but parent's best child has more weight.
	s := &Store{
		justifiedEpoch: 1,
		finalizedEpoch: 1,
		nodes: []*Node{{bestChild: 1, justifiedEpoch: 1, finalizedEpoch: 1},
			{bestDescendant: NonExistentNode, justifiedEpoch: 1, finalizedEpoch: 1, weight: 1},
			{bestDescendant: NonExistentNode, justifiedEpoch: 1, finalizedEpoch: 1}}}
	require.NoError(t, s.updateBestChildAndDescendant(0, 2))

	// Verify parent's best child and best descendant are not changed.
	assert.Equal(t, uint64(1), s.nodes[0].bestChild, "Did not get correct best child index")
	assert.Equal(t, uint64(0), s.nodes[0].bestDescendant, "Did not get correct best descendant index")
}

func TestStore_UpdateBestChildAndDescendant_NoChangeAtLeaf(t *testing.T) {
	// Make parent's best child to none and input child does not lead to viable index.
	s := &Store{
		justifiedEpoch: 1,
		finalizedEpoch: 1,
		nodes: []*Node{{bestChild: NonExistentNode, justifiedEpoch: 1, finalizedEpoch: 1},
			{bestDescendant: NonExistentNode, justifiedEpoch: 1, finalizedEpoch: 1},
			{bestDescendant: NonExistentNode}}}
	require.NoError(t, s.updateBestChildAndDescendant(0, 2))

	// Verify parent's best child and best descendant are not changed.
	assert.Equal(t, NonExistentNode, s.nodes[0].bestChild, "Did not get correct best child index")
	assert.Equal(t, uint64(0), s.nodes[0].bestDescendant, "Did not get correct best descendant index")
}

func TestStore_Prune_LessThanThreshold(t *testing.T) {
	// Define 100 nodes in store.
	numOfNodes := 100
	indices := make(map[[32]byte]uint64)
	nodes := make([]*Node, 0)
	indices[indexToHash(uint64(0))] = uint64(0)
	nodes = append(nodes, &Node{
		slot:           types.Slot(0),
		root:           indexToHash(uint64(0)),
		bestDescendant: uint64(numOfNodes - 1),
		bestChild:      uint64(1),
		parent:         NonExistentNode,
	})
	for i := 1; i < numOfNodes-1; i++ {
		indices[indexToHash(uint64(i))] = uint64(i)
		nodes = append(nodes, &Node{
			slot:           types.Slot(i),
			root:           indexToHash(uint64(i)),
			bestDescendant: uint64(numOfNodes - 1),
			bestChild:      uint64(i + 1),
			parent:         uint64(i) - 1,
		})
	}
	indices[indexToHash(uint64(numOfNodes-1))] = uint64(numOfNodes - 1)
	nodes = append(nodes, &Node{
		slot:           types.Slot(numOfNodes - 1),
		root:           indexToHash(uint64(numOfNodes - 1)),
		bestDescendant: NonExistentNode,
		bestChild:      NonExistentNode,
		parent:         uint64(numOfNodes - 2),
	})

	s := &Store{nodes: nodes, nodesIndices: indices, pruneThreshold: 100}
	syncedTips := &optimisticStore{}

	// Finalized root is at index 99 so everything before 99 should be pruned,
	// but PruneThreshold is at 100 so nothing will be pruned.
	require.NoError(t, s.prune(context.Background(), indexToHash(99), syncedTips))
	assert.Equal(t, 100, len(s.nodes), "Incorrect nodes count")
	assert.Equal(t, 100, len(s.nodesIndices), "Incorrect node indices count")
}

func TestStore_Prune_MoreThanThreshold(t *testing.T) {
	// Define 100 nodes in store.
	numOfNodes := 100
	indices := make(map[[32]byte]uint64)
	nodes := make([]*Node, 0)
	indices[indexToHash(uint64(0))] = uint64(0)
	nodes = append(nodes, &Node{
		slot:           types.Slot(0),
		root:           indexToHash(uint64(0)),
		bestDescendant: uint64(numOfNodes - 1),
		bestChild:      uint64(1),
		parent:         NonExistentNode,
	})
	for i := 1; i < numOfNodes-1; i++ {
		indices[indexToHash(uint64(i))] = uint64(i)
		nodes = append(nodes, &Node{
			slot:           types.Slot(i),
			root:           indexToHash(uint64(i)),
			bestDescendant: uint64(numOfNodes - 1),
			bestChild:      uint64(i + 1),
			parent:         uint64(i) - 1,
		})
	}
	nodes = append(nodes, &Node{
		slot:           types.Slot(numOfNodes - 1),
		root:           indexToHash(uint64(numOfNodes - 1)),
		bestDescendant: NonExistentNode,
		bestChild:      NonExistentNode,
		parent:         uint64(numOfNodes - 2),
	})
	indices[indexToHash(uint64(numOfNodes-1))] = uint64(numOfNodes - 1)
	s := &Store{nodes: nodes, nodesIndices: indices}
	syncedTips := &optimisticStore{}

	// Finalized root is at index 99 so everything before 99 should be pruned.
	require.NoError(t, s.prune(context.Background(), indexToHash(99), syncedTips))
	assert.Equal(t, 1, len(s.nodes), "Incorrect nodes count")
	assert.Equal(t, 1, len(s.nodesIndices), "Incorrect node indices count")
}

func TestStore_Prune_MoreThanOnce(t *testing.T) {
	// Define 100 nodes in store.
	numOfNodes := 100
	indices := make(map[[32]byte]uint64)
	nodes := make([]*Node, 0)
	nodes = append(nodes, &Node{
		slot:           types.Slot(0),
		root:           indexToHash(uint64(0)),
		bestDescendant: uint64(numOfNodes - 1),
		bestChild:      uint64(1),
		parent:         NonExistentNode,
	})
	for i := 1; i < numOfNodes-1; i++ {
		indices[indexToHash(uint64(i))] = uint64(i)
		nodes = append(nodes, &Node{
			slot:           types.Slot(i),
			root:           indexToHash(uint64(i)),
			bestDescendant: uint64(numOfNodes - 1),
			bestChild:      uint64(i + 1),
			parent:         uint64(i) - 1,
		})
	}
	nodes = append(nodes, &Node{
		slot:           types.Slot(numOfNodes - 1),
		root:           indexToHash(uint64(numOfNodes - 1)),
		bestDescendant: NonExistentNode,
		bestChild:      NonExistentNode,
		parent:         uint64(numOfNodes - 2),
	})

	s := &Store{nodes: nodes, nodesIndices: indices}
	syncedTips := &optimisticStore{}

	// Finalized root is at index 11 so everything before 11 should be pruned.
	require.NoError(t, s.prune(context.Background(), indexToHash(10), syncedTips))
	assert.Equal(t, 90, len(s.nodes), "Incorrect nodes count")
	assert.Equal(t, 90, len(s.nodesIndices), "Incorrect node indices count")

	// One more time.
	require.NoError(t, s.prune(context.Background(), indexToHash(20), syncedTips))
	assert.Equal(t, 80, len(s.nodes), "Incorrect nodes count")
	assert.Equal(t, 80, len(s.nodesIndices), "Incorrect node indices count")
}

// This unit tests starts with a simple branch like this
//
//       - 1
//     /
// -- 0 -- 2
//
// And we finalize 1. As a result only 1 should survive
func TestStore_Prune_NoDanglingBranch(t *testing.T) {
	nodes := []*Node{
		{
			slot:           100,
			bestChild:      1,
			bestDescendant: 1,
			root:           indexToHash(uint64(0)),
			parent:         NonExistentNode,
		},
		{
			slot:           101,
			root:           indexToHash(uint64(1)),
			bestChild:      NonExistentNode,
			bestDescendant: NonExistentNode,
			parent:         0,
		},
		{
			slot:           101,
			root:           indexToHash(uint64(2)),
			parent:         0,
			bestChild:      NonExistentNode,
			bestDescendant: NonExistentNode,
		},
	}
	syncedTips := &optimisticStore{}
	s := &Store{
		pruneThreshold: 0,
		nodes:          nodes,
		nodesIndices: map[[32]byte]uint64{
			indexToHash(uint64(0)): 0,
			indexToHash(uint64(1)): 1,
			indexToHash(uint64(2)): 2,
		},
	}
	require.NoError(t, s.prune(context.Background(), indexToHash(uint64(1)), syncedTips))
	require.Equal(t, len(s.nodes), 1)
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
// Synced tips are B, D and E. And we finalize F. All that is left in fork
// choice is F, and the only synced tip left is E which is now away from Fork
// Choice.
func TestStore_PruneSyncedTips(t *testing.T) {
	ctx := context.Background()
	f := setup(1, 1)

	require.NoError(t, f.InsertOptimisticBlock(ctx, 100, [32]byte{'a'}, params.BeaconConfig().ZeroHash, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 101, [32]byte{'b'}, [32]byte{'a'}, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 102, [32]byte{'c'}, [32]byte{'b'}, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 102, [32]byte{'j'}, [32]byte{'b'}, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 103, [32]byte{'d'}, [32]byte{'c'}, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 104, [32]byte{'e'}, [32]byte{'d'}, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 104, [32]byte{'g'}, [32]byte{'d'}, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 105, [32]byte{'f'}, [32]byte{'e'}, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 105, [32]byte{'h'}, [32]byte{'g'}, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 105, [32]byte{'k'}, [32]byte{'g'}, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 106, [32]byte{'i'}, [32]byte{'h'}, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 106, [32]byte{'l'}, [32]byte{'k'}, 1, 1))
	syncedTips := &optimisticStore{
		validatedTips: map[[32]byte]types.Slot{
			[32]byte{'b'}: 101,
			[32]byte{'d'}: 103,
			[32]byte{'e'}: 104,
		},
	}
	f.syncedTips = syncedTips
	f.store.pruneThreshold = 0
	require.NoError(t, f.Prune(ctx, [32]byte{'f'}))
	require.Equal(t, 1, len(f.syncedTips.validatedTips))
	_, ok := f.syncedTips.validatedTips[[32]byte{'e'}]
	require.Equal(t, true, ok)
}

func TestStore_LeadsToViableHead(t *testing.T) {
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
		s := &Store{
			justifiedEpoch: tc.justifiedEpoch,
			finalizedEpoch: tc.finalizedEpoch,
			nodes:          []*Node{tc.n},
		}
		got, err := s.leadsToViableHead(tc.n)
		require.NoError(t, err)
		assert.Equal(t, tc.want, got)
	}
}

func TestStore_SetSyncedTips(t *testing.T) {
	f := setup(1, 1)
	tips := make(map[[32]byte]types.Slot)
	require.ErrorIs(t, errInvalidSyncedTips, f.SetSyncedTips(tips))
	tips[bytesutil.ToBytes32([]byte{'a'})] = 1
	require.NoError(t, f.SetSyncedTips(tips))
	f.syncedTips.RLock()
	defer f.syncedTips.RUnlock()
	require.Equal(t, 1, len(f.syncedTips.validatedTips))
	slot, ok := f.syncedTips.validatedTips[bytesutil.ToBytes32([]byte{'a'})]
	require.Equal(t, true, ok)
	require.Equal(t, types.Slot(1), slot)
}

func TestStore_ViableForHead(t *testing.T) {
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
		s := &Store{
			justifiedEpoch: tc.justifiedEpoch,
			finalizedEpoch: tc.finalizedEpoch,
		}
		assert.Equal(t, tc.want, s.viableForHead(tc.n))
	}
}

func TestStore_HasParent(t *testing.T) {
	tests := []struct {
		m    map[[32]byte]uint64
		n    []*Node
		r    [32]byte
		want bool
	}{
		{r: [32]byte{'a'}, want: false},
		{m: map[[32]byte]uint64{{'a'}: 0}, r: [32]byte{'a'}, want: false},
		{m: map[[32]byte]uint64{{'a'}: 0}, r: [32]byte{'a'},
			n: []*Node{{parent: NonExistentNode}}, want: false},
		{m: map[[32]byte]uint64{{'a'}: 0},
			n: []*Node{{parent: 0}}, r: [32]byte{'a'},
			want: true},
	}
	for _, tc := range tests {
		f := &ForkChoice{store: &Store{
			nodesIndices: tc.m,
			nodes:        tc.n,
		}}
		assert.Equal(t, tc.want, f.HasParent(tc.r))
	}
}

func TestStore_AncestorRoot(t *testing.T) {
	ctx := context.Background()
	f := &ForkChoice{store: &Store{}}
	f.store.nodesIndices = map[[32]byte]uint64{}
	_, err := f.AncestorRoot(ctx, [32]byte{'a'}, 0)
	assert.ErrorContains(t, "node does not exist", err)
	f.store.nodesIndices[[32]byte{'a'}] = 0
	_, err = f.AncestorRoot(ctx, [32]byte{'a'}, 0)
	assert.ErrorContains(t, "node index out of range", err)
	f.store.nodesIndices[[32]byte{'b'}] = 1
	f.store.nodesIndices[[32]byte{'c'}] = 2
	f.store.nodes = []*Node{
		{slot: 1, root: [32]byte{'a'}, parent: NonExistentNode},
		{slot: 2, root: [32]byte{'b'}, parent: 0},
		{slot: 3, root: [32]byte{'c'}, parent: 1},
	}

	r, err := f.AncestorRoot(ctx, [32]byte{'c'}, 1)
	require.NoError(t, err)
	assert.Equal(t, bytesutil.ToBytes32(r), [32]byte{'a'})
	r, err = f.AncestorRoot(ctx, [32]byte{'c'}, 2)
	require.NoError(t, err)
	assert.Equal(t, bytesutil.ToBytes32(r), [32]byte{'b'})
}

func TestStore_AncestorRootOutOfBound(t *testing.T) {
	ctx := context.Background()
	f := &ForkChoice{store: &Store{}}
	f.store.nodesIndices = map[[32]byte]uint64{}
	_, err := f.AncestorRoot(ctx, [32]byte{'a'}, 0)
	assert.ErrorContains(t, "node does not exist", err)
	f.store.nodesIndices[[32]byte{'a'}] = 0
	_, err = f.AncestorRoot(ctx, [32]byte{'a'}, 0)
	assert.ErrorContains(t, "node index out of range", err)
	f.store.nodesIndices[[32]byte{'b'}] = 1
	f.store.nodesIndices[[32]byte{'c'}] = 2
	f.store.nodes = []*Node{
		{slot: 1, root: [32]byte{'a'}, parent: NonExistentNode},
		{slot: 2, root: [32]byte{'b'}, parent: 100}, // Out of bound parent.
		{slot: 3, root: [32]byte{'c'}, parent: 1},
	}

	_, err = f.AncestorRoot(ctx, [32]byte{'c'}, 1)
	require.ErrorContains(t, "node index out of range", err)
}

func TestStore_UpdateCanonicalNodes_WholeList(t *testing.T) {
	ctx := context.Background()
	f := &ForkChoice{store: &Store{}}
	f.store.canonicalNodes = map[[32]byte]bool{}
	f.store.nodesIndices = map[[32]byte]uint64{}
	f.store.nodes = []*Node{
		{slot: 1, root: [32]byte{'a'}, parent: NonExistentNode},
		{slot: 2, root: [32]byte{'b'}, parent: 0},
		{slot: 3, root: [32]byte{'c'}, parent: 1},
	}
	f.store.nodesIndices[[32]byte{'c'}] = 2
	require.NoError(t, f.store.updateCanonicalNodes(ctx, [32]byte{'c'}))
	require.Equal(t, len(f.store.nodes), len(f.store.canonicalNodes))
	require.Equal(t, true, f.IsCanonical([32]byte{'a'}))
	require.Equal(t, true, f.IsCanonical([32]byte{'b'}))
	require.Equal(t, true, f.IsCanonical([32]byte{'c'}))
	idxc := f.store.nodesIndices[[32]byte{'c'}]
	_, ok := f.store.nodesIndices[[32]byte{'d'}]
	require.Equal(t, idxc, uint64(2))
	require.Equal(t, false, ok)
}

func TestStore_UpdateCanonicalNodes_ParentAlreadyIn(t *testing.T) {
	ctx := context.Background()
	f := &ForkChoice{store: &Store{}}
	f.store.canonicalNodes = map[[32]byte]bool{}
	f.store.nodesIndices = map[[32]byte]uint64{}
	f.store.nodes = []*Node{
		{},
		{slot: 2, root: [32]byte{'b'}, parent: 0},
		{slot: 3, root: [32]byte{'c'}, parent: 1},
	}
	f.store.nodesIndices[[32]byte{'c'}] = 2
	f.store.canonicalNodes[[32]byte{'b'}] = true
	require.NoError(t, f.store.updateCanonicalNodes(ctx, [32]byte{'c'}))
	require.Equal(t, len(f.store.nodes)-1, len(f.store.canonicalNodes))

	require.Equal(t, true, f.IsCanonical([32]byte{'c'}))
	require.Equal(t, true, f.IsCanonical([32]byte{'b'}))
}

func TestStore_UpdateCanonicalNodes_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	f := &ForkChoice{store: &Store{}}
	f.store.canonicalNodes = map[[32]byte]bool{}
	f.store.nodesIndices = map[[32]byte]uint64{}
	f.store.nodes = []*Node{
		{slot: 1, root: [32]byte{'a'}, parent: NonExistentNode},
		{slot: 2, root: [32]byte{'b'}, parent: 0},
		{slot: 3, root: [32]byte{'c'}, parent: 1},
	}
	f.store.nodesIndices[[32]byte{'c'}] = 2
	cancel()
	require.ErrorContains(t, "context canceled", f.store.updateCanonicalNodes(ctx, [32]byte{'c'}))
}

func TestStore_UpdateCanonicalNodes_RemoveOldCanonical(t *testing.T) {
	ctx := context.Background()
	f := &ForkChoice{store: &Store{}}
	f.store.canonicalNodes = map[[32]byte]bool{}
	f.store.nodesIndices = map[[32]byte]uint64{
		[32]byte{'a'}: 0,
		[32]byte{'b'}: 1,
		[32]byte{'c'}: 2,
		[32]byte{'d'}: 3,
		[32]byte{'e'}: 4,
	}

	f.store.nodes = []*Node{
		{slot: 1, root: [32]byte{'a'}, parent: NonExistentNode},
		{slot: 2, root: [32]byte{'b'}, parent: 0},
		{slot: 3, root: [32]byte{'c'}, parent: 1},
		{slot: 4, root: [32]byte{'d'}, parent: 1},
		{slot: 5, root: [32]byte{'e'}, parent: 3},
	}
	require.NoError(t, f.store.updateCanonicalNodes(ctx, [32]byte{'c'}))
	require.Equal(t, 3, len(f.store.canonicalNodes))
	require.NoError(t, f.store.updateCanonicalNodes(ctx, [32]byte{'e'}))
	require.Equal(t, 4, len(f.store.canonicalNodes))
	require.Equal(t, true, f.IsCanonical([32]byte{'a'}))
	require.Equal(t, true, f.IsCanonical([32]byte{'b'}))
	require.Equal(t, true, f.IsCanonical([32]byte{'d'}))
	require.Equal(t, true, f.IsCanonical([32]byte{'e'}))
	_, ok := f.store.canonicalNodes[[32]byte{'c'}]
	require.Equal(t, false, ok)
}
