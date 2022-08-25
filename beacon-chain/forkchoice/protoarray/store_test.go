package protoarray

import (
	"context"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/v3/beacon-chain/forkchoice"
	forkchoicetypes "github.com/prysmaticlabs/prysm/v3/beacon-chain/forkchoice/types"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/blocks"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/testing/util"
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
	require.Equal(t, j, f.JustifiedCheckpoint().Epoch)
}

func TestStore_FinalizedEpoch(t *testing.T) {
	j := types.Epoch(50)
	f := setup(j, j)
	require.Equal(t, j, f.FinalizedCheckpoint().Epoch)
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
	s.justifiedCheckpoint = &forkchoicetypes.Checkpoint{Epoch: 1, Root: [32]byte{'a'}}

	_, err := s.head(context.Background())
	assert.ErrorContains(t, errUnknownJustifiedRoot.Error(), err)
}

func TestStore_Head_UnknownJustifiedIndex(t *testing.T) {
	r := [32]byte{'A'}
	indices := make(map[[32]byte]uint64)
	indices[r] = 1
	s := &Store{nodesIndices: indices}
	s.justifiedCheckpoint = &forkchoicetypes.Checkpoint{Epoch: 0, Root: r}

	_, err := s.head(context.Background())
	assert.ErrorContains(t, errInvalidJustifiedIndex.Error(), err)
}

func TestStore_Head_Itself(t *testing.T) {
	r := [32]byte{'A'}
	indices := map[[32]byte]uint64{r: 0}

	// Since the justified node does not have a best descendant so the best node
	// is itself.
	s := &Store{nodesIndices: indices, nodes: []*Node{{root: r, parent: NonExistentNode, bestDescendant: NonExistentNode}}, canonicalNodes: make(map[[32]byte]bool)}
	s.justifiedCheckpoint = &forkchoicetypes.Checkpoint{Epoch: 0, Root: r}
	s.finalizedCheckpoint = &forkchoicetypes.Checkpoint{Epoch: 0, Root: r}
	h, err := s.head(context.Background())
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
	s.justifiedCheckpoint = &forkchoicetypes.Checkpoint{Epoch: 0, Root: r}
	s.finalizedCheckpoint = &forkchoicetypes.Checkpoint{Epoch: 0, Root: r}
	h, err := s.head(context.Background())
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
	s.justifiedCheckpoint = &forkchoicetypes.Checkpoint{Epoch: 0, Root: r}
	s.finalizedCheckpoint = &forkchoicetypes.Checkpoint{Epoch: 0, Root: r}
	_, err := s.head(ctx)
	require.ErrorContains(t, "context canceled", err)
}

func TestStore_Insert_UnknownParent(t *testing.T) {
	// The new node does not have a parent.
	s := &Store{nodesIndices: make(map[[32]byte]uint64), payloadIndices: make(map[[32]byte]uint64)}
	_, err := s.insert(context.Background(), 100, [32]byte{'A'}, [32]byte{'B'}, params.BeaconConfig().ZeroHash, 1, 1)
	require.NoError(t, err)
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
	s := &Store{nodesIndices: make(map[[32]byte]uint64), payloadIndices: make(map[[32]byte]uint64)}
	s.nodes = []*Node{{}}
	p := [32]byte{'B'}
	s.nodesIndices[p] = 0
	payloadHash := [32]byte{'c'}
	s.justifiedCheckpoint = &forkchoicetypes.Checkpoint{}
	s.finalizedCheckpoint = &forkchoicetypes.Checkpoint{}
	_, err := s.insert(context.Background(), 100, [32]byte{'A'}, p, payloadHash, 1, 1)
	require.NoError(t, err)
	assert.Equal(t, 2, len(s.nodes), "Did not insert block")
	assert.Equal(t, 2, len(s.nodesIndices), "Did not insert block")
	assert.Equal(t, uint64(0), s.nodes[1].parent, "Incorrect parent")
	assert.Equal(t, types.Epoch(1), s.nodes[1].justifiedEpoch, "Incorrect justification")
	assert.Equal(t, types.Epoch(1), s.nodes[1].finalizedEpoch, "Incorrect finalization")
	assert.Equal(t, [32]byte{'A'}, s.nodes[1].root, "Incorrect root")
	assert.Equal(t, payloadHash, s.nodes[1].payloadHash)
}

func TestStore_ApplyScoreChanges_InvalidDeltaLength(t *testing.T) {
	s := &Store{}

	// This will fail because node indices has length of 0, and delta list has a length of 1.
	err := s.applyWeightChanges(context.Background(), []uint64{}, []int{1})
	assert.ErrorContains(t, errInvalidDeltaLength.Error(), err)
}

func TestStore_ApplyScoreChanges_UpdateWeightsPositiveDelta(t *testing.T) {
	// Construct 3 nodes with weight 100 on each node. The 3 nodes linked to each other.
	s := &Store{nodes: []*Node{
		{root: [32]byte{'A'}, weight: 100},
		{root: [32]byte{'A'}, weight: 100},
		{parent: 1, root: [32]byte{'A'}, weight: 100}}}

	// Each node gets one unique vote. The weight should look like 103 <- 102 <- 101 because
	// they get propagated back.
	s.justifiedCheckpoint = &forkchoicetypes.Checkpoint{}
	s.finalizedCheckpoint = &forkchoicetypes.Checkpoint{}
	require.NoError(t, s.applyWeightChanges(context.Background(), []uint64{}, []int{1, 1, 1}))
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
	s.justifiedCheckpoint = &forkchoicetypes.Checkpoint{}
	s.finalizedCheckpoint = &forkchoicetypes.Checkpoint{}
	require.NoError(t, s.applyWeightChanges(context.Background(), []uint64{}, []int{-1, -1, -1}))
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
	s.justifiedCheckpoint = &forkchoicetypes.Checkpoint{}
	s.finalizedCheckpoint = &forkchoicetypes.Checkpoint{}
	require.NoError(t, s.applyWeightChanges(context.Background(), []uint64{}, []int{-100, -50, 150}))
	assert.Equal(t, uint64(100), s.nodes[0].weight)
	assert.Equal(t, uint64(200), s.nodes[1].weight)
	assert.Equal(t, uint64(250), s.nodes[2].weight)
}

func TestStore_UpdateBestChildAndDescendant_RemoveChild(t *testing.T) {
	// Make parent's best child equal's to input child index and child is not viable.
	jc := &forkchoicetypes.Checkpoint{Epoch: 1}
	fc := &forkchoicetypes.Checkpoint{Epoch: 1}
	s := &Store{nodes: []*Node{{bestChild: 1}, {}}, justifiedCheckpoint: jc, finalizedCheckpoint: fc}
	require.NoError(t, s.updateBestChildAndDescendant(0, 1))

	// Verify parent's best child and best descendant are `none`.
	assert.Equal(t, NonExistentNode, s.nodes[0].bestChild, "Did not get correct best child index")
	assert.Equal(t, NonExistentNode, s.nodes[0].bestDescendant, "Did not get correct best descendant index")
}

func TestStore_UpdateBestChildAndDescendant_UpdateDescendant(t *testing.T) {
	// Make parent's best child equal to child index and child is viable.
	s := &Store{nodes: []*Node{{bestChild: 1}, {bestDescendant: NonExistentNode}}}
	s.justifiedCheckpoint = &forkchoicetypes.Checkpoint{}
	s.finalizedCheckpoint = &forkchoicetypes.Checkpoint{}
	require.NoError(t, s.updateBestChildAndDescendant(0, 1))

	// Verify parent's best child is the same and best descendant is not set to child index.
	assert.Equal(t, uint64(1), s.nodes[0].bestChild, "Did not get correct best child index")
	assert.Equal(t, uint64(1), s.nodes[0].bestDescendant, "Did not get correct best descendant index")
}

func TestStore_UpdateBestChildAndDescendant_ChangeChildByViability(t *testing.T) {
	// Make parent's best child not equal to child index, child leads to viable index and
	// parent's best child doesn't lead to viable index.
	jc := &forkchoicetypes.Checkpoint{Epoch: 1}
	fc := &forkchoicetypes.Checkpoint{Epoch: 1}
	s := &Store{
		justifiedCheckpoint: jc,
		finalizedCheckpoint: fc,
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
	jc := &forkchoicetypes.Checkpoint{Epoch: 1}
	fc := &forkchoicetypes.Checkpoint{Epoch: 1}
	s := &Store{
		justifiedCheckpoint: jc,
		finalizedCheckpoint: fc,
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
	jc := &forkchoicetypes.Checkpoint{Epoch: 1}
	fc := &forkchoicetypes.Checkpoint{Epoch: 1}
	s := &Store{
		justifiedCheckpoint: jc,
		finalizedCheckpoint: fc,
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
	jc := &forkchoicetypes.Checkpoint{Epoch: 1}
	fc := &forkchoicetypes.Checkpoint{Epoch: 1}
	s := &Store{
		justifiedCheckpoint: jc,
		finalizedCheckpoint: fc,
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
	jc := &forkchoicetypes.Checkpoint{Epoch: 1}
	fc := &forkchoicetypes.Checkpoint{Epoch: 1}
	s := &Store{
		justifiedCheckpoint: jc,
		finalizedCheckpoint: fc,
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
	jc := &forkchoicetypes.Checkpoint{Epoch: 1}
	fc := &forkchoicetypes.Checkpoint{Epoch: 1}
	s := &Store{
		justifiedCheckpoint: jc,
		finalizedCheckpoint: fc,
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

	// Finalized root is at index 99 so everything before 99 should be pruned,
	// but PruneThreshold is at 100 so nothing will be pruned.
	fc := &forkchoicetypes.Checkpoint{Epoch: 3, Root: indexToHash(99)}
	s.finalizedCheckpoint = fc
	require.NoError(t, s.prune(context.Background()))
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
	s := &Store{nodes: nodes, nodesIndices: indices, canonicalNodes: map[[32]byte]bool{}, payloadIndices: map[[32]byte]uint64{}}

	// Finalized root is at index 99 so everything before 99 should be pruned.
	fc := &forkchoicetypes.Checkpoint{Epoch: 3, Root: indexToHash(99)}
	s.finalizedCheckpoint = fc
	require.NoError(t, s.prune(context.Background()))
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

	s := &Store{nodes: nodes, nodesIndices: indices, canonicalNodes: map[[32]byte]bool{}, payloadIndices: map[[32]byte]uint64{}}

	// Finalized root is at index 11 so everything before 11 should be pruned.
	fc := &forkchoicetypes.Checkpoint{Epoch: 1, Root: indexToHash(10)}
	s.finalizedCheckpoint = fc
	require.NoError(t, s.prune(context.Background()))
	assert.Equal(t, 90, len(s.nodes), "Incorrect nodes count")
	assert.Equal(t, 90, len(s.nodesIndices), "Incorrect node indices count")

	// One more time.
	s.finalizedCheckpoint.Root = indexToHash(20)
	require.NoError(t, s.prune(context.Background()))
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
			payloadHash:    [32]byte{'A'},
		},
		{
			slot:           101,
			root:           indexToHash(uint64(1)),
			bestChild:      NonExistentNode,
			bestDescendant: NonExistentNode,
			parent:         0,
			payloadHash:    [32]byte{'B'},
		},
		{
			slot:           101,
			root:           indexToHash(uint64(2)),
			parent:         0,
			bestChild:      NonExistentNode,
			bestDescendant: NonExistentNode,
			payloadHash:    [32]byte{'C'},
		},
	}
	s := &Store{
		pruneThreshold: 0,
		nodes:          nodes,
		nodesIndices: map[[32]byte]uint64{
			indexToHash(uint64(0)): 0,
			indexToHash(uint64(1)): 1,
			indexToHash(uint64(2)): 2,
		},
		canonicalNodes: map[[32]byte]bool{
			indexToHash(uint64(0)): true,
			indexToHash(uint64(1)): true,
			indexToHash(uint64(2)): true,
		},
		payloadIndices: map[[32]byte]uint64{
			{'A'}: 0,
			{'B'}: 1,
			{'C'}: 2,
		},
	}
	fc := &forkchoicetypes.Checkpoint{Epoch: 1, Root: indexToHash(1)}
	s.finalizedCheckpoint = fc
	require.NoError(t, s.prune(context.Background()))
	require.Equal(t, 1, len(s.nodes))
	require.Equal(t, 1, len(s.nodesIndices))
	require.Equal(t, 1, len(s.canonicalNodes))
	require.Equal(t, 1, len(s.payloadIndices))
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
func TestStore_PruneBranched(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		finalizedRoot      [32]byte
		wantedCanonical    [32]byte
		wantedNonCanonical [32]byte
		canonicalCount     int
		payloadHash        [32]byte
		payloadIndex       uint64
		nonExistentPayload [32]byte
	}{
		{
			[32]byte{'f'},
			[32]byte{'f'},
			[32]byte{'a'},
			1,
			[32]byte{'F'},
			0,
			[32]byte{'H'},
		},
		{
			[32]byte{'d'},
			[32]byte{'e'},
			[32]byte{'i'},
			3,
			[32]byte{'E'},
			1,
			[32]byte{'C'},
		},
		{
			[32]byte{'b'},
			[32]byte{'f'},
			[32]byte{'h'},
			5,
			[32]byte{'D'},
			3,
			[32]byte{'A'},
		},
	}

	for _, tc := range tests {
		f := setup(1, 1)
		state, blkRoot, err := prepareForkchoiceState(ctx, 100, [32]byte{'a'}, params.BeaconConfig().ZeroHash, [32]byte{'A'}, 1, 1)
		require.NoError(t, err)
		require.NoError(t, f.InsertNode(ctx, state, blkRoot))
		state, blkRoot, err = prepareForkchoiceState(ctx, 101, [32]byte{'b'}, [32]byte{'a'}, [32]byte{'B'}, 1, 1)
		require.NoError(t, err)
		require.NoError(t, f.InsertNode(ctx, state, blkRoot))
		state, blkRoot, err = prepareForkchoiceState(ctx, 102, [32]byte{'c'}, [32]byte{'b'}, [32]byte{'C'}, 1, 1)
		require.NoError(t, err)
		require.NoError(t, f.InsertNode(ctx, state, blkRoot))
		state, blkRoot, err = prepareForkchoiceState(ctx, 102, [32]byte{'j'}, [32]byte{'b'}, [32]byte{'J'}, 1, 1)
		require.NoError(t, err)
		require.NoError(t, f.InsertNode(ctx, state, blkRoot))
		state, blkRoot, err = prepareForkchoiceState(ctx, 103, [32]byte{'d'}, [32]byte{'c'}, [32]byte{'D'}, 1, 1)
		require.NoError(t, err)
		require.NoError(t, f.InsertNode(ctx, state, blkRoot))
		state, blkRoot, err = prepareForkchoiceState(ctx, 104, [32]byte{'e'}, [32]byte{'d'}, [32]byte{'E'}, 1, 1)
		require.NoError(t, err)
		require.NoError(t, f.InsertNode(ctx, state, blkRoot))
		state, blkRoot, err = prepareForkchoiceState(ctx, 104, [32]byte{'g'}, [32]byte{'d'}, [32]byte{'G'}, 1, 1)
		require.NoError(t, err)
		require.NoError(t, f.InsertNode(ctx, state, blkRoot))
		state, blkRoot, err = prepareForkchoiceState(ctx, 105, [32]byte{'f'}, [32]byte{'e'}, [32]byte{'F'}, 1, 1)
		require.NoError(t, err)
		require.NoError(t, f.InsertNode(ctx, state, blkRoot))
		state, blkRoot, err = prepareForkchoiceState(ctx, 105, [32]byte{'h'}, [32]byte{'g'}, [32]byte{'H'}, 1, 1)
		require.NoError(t, err)
		require.NoError(t, f.InsertNode(ctx, state, blkRoot))
		state, blkRoot, err = prepareForkchoiceState(ctx, 105, [32]byte{'k'}, [32]byte{'g'}, [32]byte{'K'}, 1, 1)
		require.NoError(t, err)
		require.NoError(t, f.InsertNode(ctx, state, blkRoot))
		state, blkRoot, err = prepareForkchoiceState(ctx, 106, [32]byte{'i'}, [32]byte{'h'}, [32]byte{'I'}, 1, 1)
		require.NoError(t, err)
		require.NoError(t, f.InsertNode(ctx, state, blkRoot))
		state, blkRoot, err = prepareForkchoiceState(ctx, 106, [32]byte{'l'}, [32]byte{'k'}, [32]byte{'L'}, 1, 1)
		require.NoError(t, err)
		require.NoError(t, f.InsertNode(ctx, state, blkRoot))
		f.store.pruneThreshold = 0
		require.NoError(t, f.store.updateCanonicalNodes(ctx, [32]byte{'f'}))
		require.Equal(t, true, f.IsCanonical([32]byte{'a'}))
		require.Equal(t, true, f.IsCanonical([32]byte{'f'}))

		f.store.finalizedCheckpoint.Root = tc.finalizedRoot
		require.NoError(t, f.store.prune(ctx))
		require.Equal(t, tc.canonicalCount, len(f.store.canonicalNodes))
		require.Equal(t, true, f.IsCanonical(tc.wantedCanonical))
		require.Equal(t, false, f.IsCanonical(tc.wantedNonCanonical))
		require.Equal(t, tc.payloadIndex, f.store.payloadIndices[tc.payloadHash])
		_, ok := f.store.payloadIndices[tc.nonExistentPayload]
		require.Equal(t, false, ok)
	}
}

func TestStore_CommonAncestor(t *testing.T) {
	ctx := context.Background()
	f := setup(0, 0)

	//  /-- b -- d -- e
	// a
	//  \-- c -- f
	//        \-- g
	//        \ -- h -- i -- j
	state, blkRoot, err := prepareForkchoiceState(ctx, 0, [32]byte{'a'}, params.BeaconConfig().ZeroHash, [32]byte{'A'}, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	state, blkRoot, err = prepareForkchoiceState(ctx, 1, [32]byte{'b'}, [32]byte{'a'}, [32]byte{'B'}, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	state, blkRoot, err = prepareForkchoiceState(ctx, 2, [32]byte{'c'}, [32]byte{'a'}, [32]byte{'C'}, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	state, blkRoot, err = prepareForkchoiceState(ctx, 3, [32]byte{'d'}, [32]byte{'b'}, [32]byte{}, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	state, blkRoot, err = prepareForkchoiceState(ctx, 4, [32]byte{'e'}, [32]byte{'d'}, [32]byte{}, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	state, blkRoot, err = prepareForkchoiceState(ctx, 5, [32]byte{'f'}, [32]byte{'c'}, [32]byte{}, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	state, blkRoot, err = prepareForkchoiceState(ctx, 6, [32]byte{'g'}, [32]byte{'c'}, [32]byte{}, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	state, blkRoot, err = prepareForkchoiceState(ctx, 7, [32]byte{'h'}, [32]byte{'c'}, [32]byte{}, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	state, blkRoot, err = prepareForkchoiceState(ctx, 8, [32]byte{'i'}, [32]byte{'h'}, [32]byte{}, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	state, blkRoot, err = prepareForkchoiceState(ctx, 9, [32]byte{'j'}, [32]byte{'i'}, [32]byte{}, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))

	tests := []struct {
		name     string
		r1       [32]byte
		r2       [32]byte
		wantRoot [32]byte
		wantSlot types.Slot
	}{
		{
			name:     "Common ancestor between c and b is a",
			r1:       [32]byte{'c'},
			r2:       [32]byte{'b'},
			wantRoot: [32]byte{'a'},
			wantSlot: 0,
		},
		{
			name:     "Common ancestor between c and d is a",
			r1:       [32]byte{'c'},
			r2:       [32]byte{'d'},
			wantRoot: [32]byte{'a'},
			wantSlot: 0,
		},
		{
			name:     "Common ancestor between c and e is a",
			r1:       [32]byte{'c'},
			r2:       [32]byte{'e'},
			wantRoot: [32]byte{'a'},
			wantSlot: 0,
		},
		{
			name:     "Common ancestor between g and f is c",
			r1:       [32]byte{'g'},
			r2:       [32]byte{'f'},
			wantRoot: [32]byte{'c'},
			wantSlot: 2,
		},
		{
			name:     "Common ancestor between f and h is c",
			r1:       [32]byte{'f'},
			r2:       [32]byte{'h'},
			wantRoot: [32]byte{'c'},
			wantSlot: 2,
		},
		{
			name:     "Common ancestor between g and h is c",
			r1:       [32]byte{'g'},
			r2:       [32]byte{'h'},
			wantRoot: [32]byte{'c'},
			wantSlot: 2,
		},
		{
			name:     "Common ancestor between b and h is a",
			r1:       [32]byte{'b'},
			r2:       [32]byte{'h'},
			wantRoot: [32]byte{'a'},
			wantSlot: 0,
		},
		{
			name:     "Common ancestor between e and h is a",
			r1:       [32]byte{'e'},
			r2:       [32]byte{'h'},
			wantRoot: [32]byte{'a'},
			wantSlot: 0,
		},
		{
			name:     "Common ancestor between i and f is c",
			r1:       [32]byte{'i'},
			r2:       [32]byte{'f'},
			wantRoot: [32]byte{'c'},
			wantSlot: 2,
		},
		{
			name:     "Common ancestor between e and h is a",
			r1:       [32]byte{'j'},
			r2:       [32]byte{'g'},
			wantRoot: [32]byte{'c'},
			wantSlot: 2,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotRoot, gotSlot, err := f.CommonAncestor(ctx, tc.r1, tc.r2)
			require.NoError(t, err)
			require.Equal(t, tc.wantRoot, gotRoot)
			require.Equal(t, tc.wantSlot, gotSlot)
		})
	}

	// a -- b -- c -- d
	f = setup(0, 0)
	state, blkRoot, err = prepareForkchoiceState(ctx, 0, [32]byte{'a'}, params.BeaconConfig().ZeroHash, [32]byte{'A'}, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	state, blkRoot, err = prepareForkchoiceState(ctx, 1, [32]byte{'b'}, [32]byte{'a'}, [32]byte{'B'}, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	state, blkRoot, err = prepareForkchoiceState(ctx, 2, [32]byte{'c'}, [32]byte{'b'}, [32]byte{'C'}, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	state, blkRoot, err = prepareForkchoiceState(ctx, 3, [32]byte{'d'}, [32]byte{'c'}, [32]byte{}, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	tests = []struct {
		name     string
		r1       [32]byte
		r2       [32]byte
		wantRoot [32]byte
		wantSlot types.Slot
	}{
		{
			name:     "Common ancestor between a and b is a",
			r1:       [32]byte{'a'},
			r2:       [32]byte{'b'},
			wantRoot: [32]byte{'a'},
			wantSlot: 0,
		},
		{
			name:     "Common ancestor between b and d is b",
			r1:       [32]byte{'d'},
			r2:       [32]byte{'b'},
			wantRoot: [32]byte{'b'},
			wantSlot: 1,
		},
		{
			name:     "Common ancestor between d and a is a",
			r1:       [32]byte{'d'},
			r2:       [32]byte{'a'},
			wantRoot: [32]byte{'a'},
			wantSlot: 0,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotRoot, gotSlot, err := f.CommonAncestor(ctx, tc.r1, tc.r2)
			require.NoError(t, err)
			require.Equal(t, tc.wantRoot, gotRoot)
			require.Equal(t, tc.wantSlot, gotSlot)
		})
	}

	// Equal inputs should return the same root.
	r, s, err := f.CommonAncestor(ctx, [32]byte{'b'}, [32]byte{'b'})
	require.NoError(t, err)
	require.Equal(t, [32]byte{'b'}, r)
	require.Equal(t, types.Slot(1), s)
	// Requesting finalized root (last node) should return the same root.
	r, s, err = f.CommonAncestor(ctx, [32]byte{'a'}, [32]byte{'a'})
	require.NoError(t, err)
	require.Equal(t, [32]byte{'a'}, r)
	require.Equal(t, types.Slot(0), s)
	// Requesting unknown root
	_, _, err = f.CommonAncestor(ctx, [32]byte{'a'}, [32]byte{'z'})
	require.ErrorIs(t, err, forkchoice.ErrUnknownCommonAncestor)
	_, _, err = f.CommonAncestor(ctx, [32]byte{'z'}, [32]byte{'a'})
	require.ErrorIs(t, err, forkchoice.ErrUnknownCommonAncestor)
	state, blkRoot, err = prepareForkchoiceState(ctx, 100, [32]byte{'y'}, [32]byte{'z'}, [32]byte{}, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	// broken link
	_, _, err = f.CommonAncestor(ctx, [32]byte{'y'}, [32]byte{'a'})
	require.ErrorIs(t, err, forkchoice.ErrUnknownCommonAncestor)
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
		jc := &forkchoicetypes.Checkpoint{Epoch: tc.justifiedEpoch}
		fc := &forkchoicetypes.Checkpoint{Epoch: tc.finalizedEpoch}
		s := &Store{
			justifiedCheckpoint: jc,
			finalizedCheckpoint: fc,
			nodes:               []*Node{tc.n},
		}
		got, err := s.leadsToViableHead(tc.n)
		require.NoError(t, err)
		assert.Equal(t, tc.want, got)
	}
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
		jc := &forkchoicetypes.Checkpoint{Epoch: tc.justifiedEpoch}
		fc := &forkchoicetypes.Checkpoint{Epoch: tc.finalizedEpoch}
		s := &Store{
			justifiedCheckpoint: jc,
			finalizedCheckpoint: fc,
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
	assert.Equal(t, r, [32]byte{'a'})
	r, err = f.AncestorRoot(ctx, [32]byte{'c'}, 2)
	require.NoError(t, err)
	assert.Equal(t, r, [32]byte{'b'})
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
		{'a'}: 0,
		{'b'}: 1,
		{'c'}: 2,
		{'d'}: 3,
		{'e'}: 4,
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

func TestStore_RemoveEquivocating(t *testing.T) {
	ctx := context.Background()
	f := setup(1, 1)
	// Insert a block it will be head
	state, blkRoot, err := prepareForkchoiceState(ctx, 1, [32]byte{'a'}, params.BeaconConfig().ZeroHash, [32]byte{'A'}, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	head, err := f.Head(ctx, []uint64{})
	require.NoError(t, err)
	require.Equal(t, [32]byte{'a'}, head)

	// Insert two extra blocks
	state, blkRoot, err = prepareForkchoiceState(ctx, 2, [32]byte{'b'}, [32]byte{'a'}, [32]byte{'B'}, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	state, blkRoot, err = prepareForkchoiceState(ctx, 3, [32]byte{'c'}, [32]byte{'a'}, [32]byte{'C'}, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	head, err = f.Head(ctx, []uint64{})
	require.NoError(t, err)
	require.Equal(t, [32]byte{'c'}, head)

	// Insert two attestations for block b, it becomes head
	f.ProcessAttestation(ctx, []uint64{1, 2}, [32]byte{'b'}, 1)
	f.ProcessAttestation(ctx, []uint64{3}, [32]byte{'c'}, 1)
	head, err = f.Head(ctx, []uint64{100, 200, 200, 300})
	require.NoError(t, err)
	require.Equal(t, [32]byte{'b'}, head)

	// Process b's slashing, c is now head
	f.InsertSlashedIndex(ctx, 1)
	head, err = f.Head(ctx, []uint64{100, 200, 200, 300})
	require.NoError(t, err)
	require.Equal(t, [32]byte{'c'}, head)
	require.Equal(t, uint64(200), f.store.nodes[2].weight)
	require.Equal(t, uint64(300), f.store.nodes[3].weight)

	// Process the same slashing again, should be a noop
	f.InsertSlashedIndex(ctx, 1)
	head, err = f.Head(ctx, []uint64{100, 200, 200, 300})
	require.NoError(t, err)
	require.Equal(t, [32]byte{'c'}, head)
	require.Equal(t, uint64(200), f.store.nodes[2].weight)
	require.Equal(t, uint64(300), f.store.nodes[3].weight)

	// Process index where index == vote length. Should not panic.
	f.InsertSlashedIndex(ctx, types.ValidatorIndex(len(f.balances)))
	f.InsertSlashedIndex(ctx, types.ValidatorIndex(len(f.votes)))
	require.Equal(t, true, len(f.store.slashedIndices) > 0)
}

func TestStore_UpdateCheckpoints(t *testing.T) {
	f := setup(1, 1)
	jr := [32]byte{'j'}
	fr := [32]byte{'f'}
	jc := &forkchoicetypes.Checkpoint{Root: jr, Epoch: 3}
	fc := &forkchoicetypes.Checkpoint{Root: fr, Epoch: 2}
	require.NoError(t, f.UpdateJustifiedCheckpoint(jc))
	require.NoError(t, f.UpdateFinalizedCheckpoint(fc))
	require.Equal(t, f.store.justifiedCheckpoint, jc)
	require.Equal(t, f.store.finalizedCheckpoint, fc)
}

func TestStore_InsertOptimisticChain(t *testing.T) {
	f := setup(1, 1)
	blks := make([]*forkchoicetypes.BlockAndCheckpoints, 0)
	blk := util.NewBeaconBlock()
	blk.Block.Slot = 1
	pr := [32]byte{}
	blk.Block.ParentRoot = pr[:]
	root, err := blk.Block.HashTreeRoot()
	require.NoError(t, err)
	wsb, err := blocks.NewSignedBeaconBlock(blk)
	require.NoError(t, err)
	blks = append(blks, &forkchoicetypes.BlockAndCheckpoints{Block: wsb.Block(),
		JustifiedCheckpoint: &ethpb.Checkpoint{Epoch: 1, Root: params.BeaconConfig().ZeroHash[:]},
		FinalizedCheckpoint: &ethpb.Checkpoint{Epoch: 1, Root: params.BeaconConfig().ZeroHash[:]},
	})
	for i := uint64(2); i < 11; i++ {
		blk := util.NewBeaconBlock()
		blk.Block.Slot = types.Slot(i)
		copiedRoot := root
		blk.Block.ParentRoot = copiedRoot[:]
		wsb, err = blocks.NewSignedBeaconBlock(blk)
		require.NoError(t, err)
		blks = append(blks, &forkchoicetypes.BlockAndCheckpoints{Block: wsb.Block(),
			JustifiedCheckpoint: &ethpb.Checkpoint{Epoch: 1, Root: params.BeaconConfig().ZeroHash[:]},
			FinalizedCheckpoint: &ethpb.Checkpoint{Epoch: 1, Root: params.BeaconConfig().ZeroHash[:]},
		})
		root, err = blk.Block.HashTreeRoot()
		require.NoError(t, err)
	}
	args := make([]*forkchoicetypes.BlockAndCheckpoints, 10)
	for i := 0; i < len(blks); i++ {
		args[i] = blks[10-i-1]
	}
	require.NoError(t, f.InsertOptimisticChain(context.Background(), args))

	f = setup(1, 1)
	require.NoError(t, f.InsertOptimisticChain(context.Background(), args[2:]))
}

func TestForkChoice_UpdateCheckpoints(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name                string
		justified           *forkchoicetypes.Checkpoint
		bestJustified       *forkchoicetypes.Checkpoint
		finalized           *forkchoicetypes.Checkpoint
		newJustified        *forkchoicetypes.Checkpoint
		newFinalized        *forkchoicetypes.Checkpoint
		wantedJustified     *forkchoicetypes.Checkpoint
		wantedBestJustified *forkchoicetypes.Checkpoint
		wantedFinalized     *forkchoicetypes.Checkpoint
		currentSlot         types.Slot
		wantedErr           string
	}{
		{
			name:                "lower than store justified and finalized",
			justified:           &forkchoicetypes.Checkpoint{Epoch: 2, Root: [32]byte{'j'}},
			finalized:           &forkchoicetypes.Checkpoint{Epoch: 1, Root: [32]byte{'f'}},
			bestJustified:       &forkchoicetypes.Checkpoint{Epoch: 2, Root: [32]byte{'j'}},
			newJustified:        &forkchoicetypes.Checkpoint{Epoch: 1},
			newFinalized:        &forkchoicetypes.Checkpoint{Epoch: 0},
			wantedJustified:     &forkchoicetypes.Checkpoint{Epoch: 2, Root: [32]byte{'j'}},
			wantedBestJustified: &forkchoicetypes.Checkpoint{Epoch: 2, Root: [32]byte{'j'}},
			wantedFinalized:     &forkchoicetypes.Checkpoint{Epoch: 1, Root: [32]byte{'f'}},
		},
		{
			name:                "higher than store justified, early slot, direct descendant",
			justified:           &forkchoicetypes.Checkpoint{Epoch: 2, Root: [32]byte{'j'}},
			bestJustified:       &forkchoicetypes.Checkpoint{Epoch: 2, Root: [32]byte{'j'}},
			finalized:           &forkchoicetypes.Checkpoint{Epoch: 1, Root: [32]byte{'f'}},
			newJustified:        &forkchoicetypes.Checkpoint{Epoch: 3, Root: [32]byte{'b'}},
			newFinalized:        &forkchoicetypes.Checkpoint{Epoch: 1, Root: [32]byte{'g'}},
			wantedJustified:     &forkchoicetypes.Checkpoint{Epoch: 3, Root: [32]byte{'b'}},
			wantedBestJustified: &forkchoicetypes.Checkpoint{Epoch: 3, Root: [32]byte{'b'}},
			wantedFinalized:     &forkchoicetypes.Checkpoint{Epoch: 1, Root: [32]byte{'f'}},
		},
		{
			name:                "higher than store justified, early slot, not a descendant",
			justified:           &forkchoicetypes.Checkpoint{Epoch: 2, Root: [32]byte{'j'}},
			bestJustified:       &forkchoicetypes.Checkpoint{Epoch: 2, Root: [32]byte{'j'}},
			finalized:           &forkchoicetypes.Checkpoint{Epoch: 1, Root: [32]byte{'f'}},
			newJustified:        &forkchoicetypes.Checkpoint{Epoch: 3, Root: [32]byte{'c'}},
			newFinalized:        &forkchoicetypes.Checkpoint{Epoch: 1, Root: [32]byte{'g'}},
			wantedJustified:     &forkchoicetypes.Checkpoint{Epoch: 3, Root: [32]byte{'c'}},
			wantedBestJustified: &forkchoicetypes.Checkpoint{Epoch: 3, Root: [32]byte{'c'}},
			wantedFinalized:     &forkchoicetypes.Checkpoint{Epoch: 1, Root: [32]byte{'f'}},
		},
		{
			name:                "higher than store justified, late slot, descendant",
			justified:           &forkchoicetypes.Checkpoint{Epoch: 2, Root: [32]byte{'j'}},
			bestJustified:       &forkchoicetypes.Checkpoint{Epoch: 2, Root: [32]byte{'j'}},
			finalized:           &forkchoicetypes.Checkpoint{Epoch: 1, Root: [32]byte{'f'}},
			newJustified:        &forkchoicetypes.Checkpoint{Epoch: 3, Root: [32]byte{'b'}},
			newFinalized:        &forkchoicetypes.Checkpoint{Epoch: 1, Root: [32]byte{'g'}},
			wantedJustified:     &forkchoicetypes.Checkpoint{Epoch: 3, Root: [32]byte{'b'}},
			wantedFinalized:     &forkchoicetypes.Checkpoint{Epoch: 1, Root: [32]byte{'f'}},
			wantedBestJustified: &forkchoicetypes.Checkpoint{Epoch: 3, Root: [32]byte{'b'}},
			currentSlot:         params.BeaconConfig().SafeSlotsToUpdateJustified.Add(1),
		},
		{
			name:                "higher than store justified, late slot, not descendant",
			justified:           &forkchoicetypes.Checkpoint{Epoch: 2, Root: [32]byte{'j'}},
			bestJustified:       &forkchoicetypes.Checkpoint{Epoch: 2, Root: [32]byte{'j'}},
			finalized:           &forkchoicetypes.Checkpoint{Epoch: 1, Root: [32]byte{'f'}},
			newJustified:        &forkchoicetypes.Checkpoint{Epoch: 3, Root: [32]byte{'c'}},
			newFinalized:        &forkchoicetypes.Checkpoint{Epoch: 1, Root: [32]byte{'g'}},
			wantedJustified:     &forkchoicetypes.Checkpoint{Epoch: 2, Root: [32]byte{'j'}},
			wantedFinalized:     &forkchoicetypes.Checkpoint{Epoch: 1, Root: [32]byte{'f'}},
			wantedBestJustified: &forkchoicetypes.Checkpoint{Epoch: 3, Root: [32]byte{'c'}},
			currentSlot:         params.BeaconConfig().SafeSlotsToUpdateJustified.Add(1),
		},
		{
			name:                "higher than store finalized, late slot, not descendant",
			justified:           &forkchoicetypes.Checkpoint{Epoch: 2, Root: [32]byte{'j'}},
			bestJustified:       &forkchoicetypes.Checkpoint{Epoch: 2, Root: [32]byte{'j'}},
			finalized:           &forkchoicetypes.Checkpoint{Epoch: 1, Root: [32]byte{'f'}},
			newJustified:        &forkchoicetypes.Checkpoint{Epoch: 3, Root: [32]byte{'c'}},
			newFinalized:        &forkchoicetypes.Checkpoint{Epoch: 2, Root: [32]byte{'h'}},
			wantedJustified:     &forkchoicetypes.Checkpoint{Epoch: 3, Root: [32]byte{'c'}},
			wantedFinalized:     &forkchoicetypes.Checkpoint{Epoch: 2, Root: [32]byte{'h'}},
			wantedBestJustified: &forkchoicetypes.Checkpoint{Epoch: 3, Root: [32]byte{'c'}},
			currentSlot:         params.BeaconConfig().SafeSlotsToUpdateJustified.Add(1),
		},
		{
			name:          "Unknown checkpoint root, late slot",
			justified:     &forkchoicetypes.Checkpoint{Epoch: 2, Root: [32]byte{'j'}},
			bestJustified: &forkchoicetypes.Checkpoint{Epoch: 2, Root: [32]byte{'j'}},
			finalized:     &forkchoicetypes.Checkpoint{Epoch: 1, Root: [32]byte{'f'}},
			newJustified:  &forkchoicetypes.Checkpoint{Epoch: 3, Root: [32]byte{'d'}},
			newFinalized:  &forkchoicetypes.Checkpoint{Epoch: 1, Root: [32]byte{'h'}},
			currentSlot:   params.BeaconConfig().SafeSlotsToUpdateJustified.Add(1),
			wantedErr:     "node does not exist",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fcs := setup(tt.justified.Epoch, tt.finalized.Epoch)
			fcs.store.justifiedCheckpoint = tt.justified
			fcs.store.finalizedCheckpoint = tt.finalized
			fcs.store.bestJustifiedCheckpoint = tt.bestJustified
			fcs.store.genesisTime = uint64(time.Now().Unix()) - uint64(tt.currentSlot)*params.BeaconConfig().SecondsPerSlot

			state, blkRoot, err := prepareForkchoiceState(ctx, 32, [32]byte{'f'},
				[32]byte{}, [32]byte{}, tt.finalized.Epoch, tt.finalized.Epoch)
			require.NoError(t, err)
			require.NoError(t, fcs.InsertNode(ctx, state, blkRoot))
			state, blkRoot, err = prepareForkchoiceState(ctx, 64, [32]byte{'j'},
				[32]byte{'f'}, [32]byte{}, tt.justified.Epoch, tt.finalized.Epoch)
			require.NoError(t, err)
			require.NoError(t, fcs.InsertNode(ctx, state, blkRoot))
			state, blkRoot, err = prepareForkchoiceState(ctx, 96, [32]byte{'b'},
				[32]byte{'j'}, [32]byte{}, tt.newJustified.Epoch, tt.newFinalized.Epoch)
			require.NoError(t, err)
			require.NoError(t, fcs.InsertNode(ctx, state, blkRoot))
			state, blkRoot, err = prepareForkchoiceState(ctx, 96, [32]byte{'c'},
				[32]byte{'f'}, [32]byte{}, tt.newJustified.Epoch, tt.newFinalized.Epoch)
			require.NoError(t, err)
			require.NoError(t, fcs.InsertNode(ctx, state, blkRoot))
			state, blkRoot, err = prepareForkchoiceState(ctx, 65, [32]byte{'h'},
				[32]byte{'f'}, [32]byte{}, tt.newFinalized.Epoch, tt.newFinalized.Epoch)
			require.NoError(t, err)
			require.NoError(t, fcs.InsertNode(ctx, state, blkRoot))
			// restart justifications cause insertion messed it up
			// restart justifications cause insertion messed it up
			fcs.store.justifiedCheckpoint = tt.justified
			fcs.store.finalizedCheckpoint = tt.finalized
			fcs.store.bestJustifiedCheckpoint = tt.bestJustified

			jc := &ethpb.Checkpoint{Epoch: tt.newJustified.Epoch, Root: tt.newJustified.Root[:]}
			fc := &ethpb.Checkpoint{Epoch: tt.newFinalized.Epoch, Root: tt.newFinalized.Root[:]}
			err = fcs.updateCheckpoints(ctx, jc, fc)
			if len(tt.wantedErr) > 0 {
				require.ErrorContains(t, tt.wantedErr, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.wantedJustified.Epoch, fcs.store.justifiedCheckpoint.Epoch)
				require.Equal(t, tt.wantedFinalized.Epoch, fcs.store.finalizedCheckpoint.Epoch)
				require.Equal(t, tt.wantedJustified.Root, fcs.store.justifiedCheckpoint.Root)
				require.Equal(t, tt.wantedFinalized.Root, fcs.store.finalizedCheckpoint.Root)
				require.Equal(t, tt.wantedBestJustified.Epoch, fcs.store.bestJustifiedCheckpoint.Epoch)
				require.Equal(t, tt.wantedBestJustified.Root, fcs.store.bestJustifiedCheckpoint.Root)
			}
		})
	}
}

func TestForkChoice_HighestReceivedBlockSlot(t *testing.T) {
	f := setup(1, 1)
	s := f.store
	_, err := s.insert(context.Background(), 100, [32]byte{'A'}, [32]byte{'a'}, params.BeaconConfig().ZeroHash, 1, 1)
	require.NoError(t, err)
	require.Equal(t, types.Slot(100), s.highestReceivedSlot)
	require.Equal(t, types.Slot(100), f.HighestReceivedBlockSlot())
	_, err = s.insert(context.Background(), 1000, [32]byte{'B'}, [32]byte{'A'}, params.BeaconConfig().ZeroHash, 1, 1)
	require.NoError(t, err)
	require.Equal(t, types.Slot(1000), s.highestReceivedSlot)
	require.Equal(t, types.Slot(1000), f.HighestReceivedBlockSlot())
	_, err = s.insert(context.Background(), 500, [32]byte{'C'}, [32]byte{'A'}, params.BeaconConfig().ZeroHash, 1, 1)
	require.NoError(t, err)
	require.Equal(t, types.Slot(1000), s.highestReceivedSlot)
	require.Equal(t, types.Slot(1000), f.HighestReceivedBlockSlot())
}

func TestForkChoice_ReceivedBlocksLastEpoch(t *testing.T) {
	f := setup(1, 1)
	s := f.store
	b := [32]byte{}

	// Make sure it doesn't underflow
	s.genesisTime = uint64(time.Now().Add(time.Duration(-1*int64(params.BeaconConfig().SecondsPerSlot)) * time.Second).Unix())
	_, err := s.insert(context.Background(), 1, [32]byte{'a'}, b, b, 1, 1)
	require.NoError(t, err)
	count, err := f.ReceivedBlocksLastEpoch()
	require.NoError(t, err)
	require.Equal(t, uint64(1), count)
	require.Equal(t, types.Slot(1), f.HighestReceivedBlockSlot())

	// 64
	// Received block last epoch is 1
	_, err = s.insert(context.Background(), 64, [32]byte{'A'}, b, b, 1, 1)
	require.NoError(t, err)
	s.genesisTime = uint64(time.Now().Add(time.Duration(-64*int64(params.BeaconConfig().SecondsPerSlot)) * time.Second).Unix())
	count, err = f.ReceivedBlocksLastEpoch()
	require.NoError(t, err)
	require.Equal(t, uint64(1), count)
	require.Equal(t, types.Slot(64), f.HighestReceivedBlockSlot())

	// 64 65
	// Received block last epoch is 2
	_, err = s.insert(context.Background(), 65, [32]byte{'B'}, b, b, 1, 1)
	require.NoError(t, err)
	s.genesisTime = uint64(time.Now().Add(time.Duration(-65*int64(params.BeaconConfig().SecondsPerSlot)) * time.Second).Unix())
	count, err = f.ReceivedBlocksLastEpoch()
	require.NoError(t, err)
	require.Equal(t, uint64(2), count)
	require.Equal(t, types.Slot(65), f.HighestReceivedBlockSlot())

	// 64 65 66
	// Received block last epoch is 3
	_, err = s.insert(context.Background(), 66, [32]byte{'C'}, b, b, 1, 1)
	require.NoError(t, err)
	s.genesisTime = uint64(time.Now().Add(time.Duration(-66*int64(params.BeaconConfig().SecondsPerSlot)) * time.Second).Unix())
	count, err = f.ReceivedBlocksLastEpoch()
	require.NoError(t, err)
	require.Equal(t, uint64(3), count)
	require.Equal(t, types.Slot(66), f.HighestReceivedBlockSlot())

	// 64 65 66
	//       98
	// Received block last epoch is 1
	_, err = s.insert(context.Background(), 98, [32]byte{'D'}, b, b, 1, 1)
	require.NoError(t, err)
	s.genesisTime = uint64(time.Now().Add(time.Duration(-98*int64(params.BeaconConfig().SecondsPerSlot)) * time.Second).Unix())
	count, err = f.ReceivedBlocksLastEpoch()
	require.NoError(t, err)
	require.Equal(t, uint64(1), count)
	require.Equal(t, types.Slot(98), f.HighestReceivedBlockSlot())

	// 64 65 66
	//       98
	//              132
	// Received block last epoch is 1
	_, err = s.insert(context.Background(), 132, [32]byte{'E'}, b, b, 1, 1)
	require.NoError(t, err)
	s.genesisTime = uint64(time.Now().Add(time.Duration(-132*int64(params.BeaconConfig().SecondsPerSlot)) * time.Second).Unix())
	count, err = f.ReceivedBlocksLastEpoch()
	require.NoError(t, err)
	require.Equal(t, uint64(1), count)
	require.Equal(t, types.Slot(132), f.HighestReceivedBlockSlot())

	// 64 65 66
	//       98
	//              132
	//       99
	// Received block last epoch is still 1. 99 is outside the window
	_, err = s.insert(context.Background(), 99, [32]byte{'F'}, b, b, 1, 1)
	require.NoError(t, err)
	s.genesisTime = uint64(time.Now().Add(time.Duration(-132*int64(params.BeaconConfig().SecondsPerSlot)) * time.Second).Unix())
	count, err = f.ReceivedBlocksLastEpoch()
	require.NoError(t, err)
	require.Equal(t, uint64(1), count)
	require.Equal(t, types.Slot(132), f.HighestReceivedBlockSlot())

	// 64 65 66
	//       98
	//              132
	//       99 100
	// Received block last epoch is still 1. 100 is at the same position as 132
	_, err = s.insert(context.Background(), 100, [32]byte{'G'}, b, b, 1, 1)
	require.NoError(t, err)
	s.genesisTime = uint64(time.Now().Add(time.Duration(-132*int64(params.BeaconConfig().SecondsPerSlot)) * time.Second).Unix())
	count, err = f.ReceivedBlocksLastEpoch()
	require.NoError(t, err)
	require.Equal(t, uint64(1), count)
	require.Equal(t, types.Slot(132), f.HighestReceivedBlockSlot())

	// 64 65 66
	//       98
	//              132
	//       99 100 101
	// Received block last epoch is 2. 101 is within the window
	_, err = s.insert(context.Background(), 101, [32]byte{'H'}, b, b, 1, 1)
	require.NoError(t, err)
	s.genesisTime = uint64(time.Now().Add(time.Duration(-132*int64(params.BeaconConfig().SecondsPerSlot)) * time.Second).Unix())
	count, err = f.ReceivedBlocksLastEpoch()
	require.NoError(t, err)
	require.Equal(t, uint64(2), count)
	require.Equal(t, types.Slot(132), f.HighestReceivedBlockSlot())

	s.genesisTime = uint64(time.Now().Add(time.Duration(-134*int64(params.BeaconConfig().SecondsPerSlot)) * time.Second).Unix())
	count, err = f.ReceivedBlocksLastEpoch()
	require.NoError(t, err)
	require.Equal(t, uint64(1), count)
	s.genesisTime = uint64(time.Now().Add(time.Duration(-165*int64(params.BeaconConfig().SecondsPerSlot)) * time.Second).Unix())
	count, err = f.ReceivedBlocksLastEpoch()
	require.NoError(t, err)
	require.Equal(t, uint64(0), count)
}
