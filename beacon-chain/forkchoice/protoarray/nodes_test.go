package protoarray

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

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
	indices := make(map[[32]byte]uint64)
	indices[r] = 0

	// Since the justified node does not have a best descendant so the best node
	// is itself.
	s := &Store{nodesIndices: indices, nodes: []*Node{{root: r, bestDescendant: NonExistentNode}}}
	h, err := s.head(context.Background(), r)
	require.NoError(t, err)
	assert.Equal(t, r, h)
}

func TestStore_Head_BestDescendant(t *testing.T) {
	r := [32]byte{'A'}
	best := [32]byte{'B'}
	indices := make(map[[32]byte]uint64)
	indices[r] = 0

	// Since the justified node's best descendent is at index 1 and it's root is `best`,
	// the head should be `best`.
	s := &Store{nodesIndices: indices, nodes: []*Node{{root: r, bestDescendant: 1}, {root: best}}}
	h, err := s.head(context.Background(), r)
	require.NoError(t, err)
	assert.Equal(t, best, h)
}

func TestStore_Insert_UnknownParent(t *testing.T) {
	// The new node does not have a parent.
	s := &Store{nodesIndices: make(map[[32]byte]uint64)}
	require.NoError(t, s.insert(context.Background(), 100, [32]byte{'A'}, [32]byte{'B'}, [32]byte{}, 1, 1))
	assert.Equal(t, 1, len(s.nodes), "Did not insert block")
	assert.Equal(t, 1, len(s.nodesIndices), "Did not insert block")
	assert.Equal(t, NonExistentNode, s.nodes[0].parent, "Incorrect parent")
	assert.Equal(t, uint64(1), s.nodes[0].justifiedEpoch, "Incorrect justification")
	assert.Equal(t, uint64(1), s.nodes[0].finalizedEpoch, "Incorrect finalization")
	assert.Equal(t, [32]byte{'A'}, s.nodes[0].root, "Incorrect root")
}

func TestStore_Insert_KnownParent(t *testing.T) {
	// Similar to UnknownParent test, but this time the new node has a valid parent already in store.
	// The new node builds on top of the parent.
	s := &Store{nodesIndices: make(map[[32]byte]uint64)}
	s.nodes = []*Node{{}}
	p := [32]byte{'B'}
	s.nodesIndices[p] = 0
	require.NoError(t, s.insert(context.Background(), 100, [32]byte{'A'}, p, [32]byte{}, 1, 1))
	assert.Equal(t, 2, len(s.nodes), "Did not insert block")
	assert.Equal(t, 2, len(s.nodesIndices), "Did not insert block")
	assert.Equal(t, uint64(0), s.nodes[1].parent, "Incorrect parent")
	assert.Equal(t, uint64(1), s.nodes[1].justifiedEpoch, "Incorrect justification")
	assert.Equal(t, uint64(1), s.nodes[1].finalizedEpoch, "Incorrect finalization")
	assert.Equal(t, [32]byte{'A'}, s.nodes[1].root, "Incorrect root")
}

func TestStore_ApplyScoreChanges_InvalidDeltaLength(t *testing.T) {
	s := &Store{}

	// This will fail because node indices has length of 0, and delta list has a length of 1.
	err := s.applyWeightChanges(context.Background(), 0, 0, []int{1})
	assert.ErrorContains(t, errInvalidDeltaLength.Error(), err)
}

func TestStore_ApplyScoreChanges_UpdateEpochs(t *testing.T) {
	s := &Store{}

	// The justified and finalized epochs in Store should be updated to 1 and 1 given the following input.
	require.NoError(t, s.applyWeightChanges(context.Background(), 1, 1, []int{}))
	assert.Equal(t, uint64(1), s.justifiedEpoch, "Did not update justified epoch")
	assert.Equal(t, uint64(1), s.finalizedEpoch, "Did not update finalized epoch")
}

func TestStore_ApplyScoreChanges_UpdateWeightsPositiveDelta(t *testing.T) {
	// Construct 3 nodes with weight 100 on each node. The 3 nodes linked to each other.
	s := &Store{nodes: []*Node{
		{root: [32]byte{'A'}, weight: 100},
		{root: [32]byte{'A'}, weight: 100},
		{parent: 1, root: [32]byte{'A'}, weight: 100}}}

	// Each node gets one unique vote. The weight should look like 103 <- 102 <- 101 because
	// they get propagated back.
	require.NoError(t, s.applyWeightChanges(context.Background(), 0, 0, []int{1, 1, 1}))
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
	require.NoError(t, s.applyWeightChanges(context.Background(), 0, 0, []int{-1, -1, -1}))
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
	require.NoError(t, s.applyWeightChanges(context.Background(), 0, 0, []int{-100, -50, 150}))
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
	// parents best child doesnt lead to viable index.
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
	for i := 0; i < numOfNodes; i++ {
		indices[indexToHash(uint64(i))] = uint64(i)
		nodes = append(nodes, &Node{slot: uint64(i)})
	}

	s := &Store{nodes: nodes, nodesIndices: indices, pruneThreshold: 100}

	// Finalized root is at index 99 so everything before 99 should be pruned,
	// but PruneThreshold is at 100 so nothing will be pruned.
	require.NoError(t, s.prune(context.Background(), indexToHash(99)))
	assert.Equal(t, 100, len(s.nodes), "Incorrect nodes count")
	assert.Equal(t, 100, len(s.nodesIndices), "Incorrect node indices count")
}

func TestStore_Prune_MoreThanThreshold(t *testing.T) {
	// Define 100 nodes in store.
	numOfNodes := 100
	indices := make(map[[32]byte]uint64)
	nodes := make([]*Node, 0)
	for i := 0; i < numOfNodes; i++ {
		indices[indexToHash(uint64(i))] = uint64(i)
		nodes = append(nodes, &Node{slot: uint64(i), root: indexToHash(uint64(i)),
			bestDescendant: NonExistentNode, bestChild: NonExistentNode})
	}

	s := &Store{nodes: nodes, nodesIndices: indices}

	// Finalized root is at index 99 so everything before 99 should be pruned.
	require.NoError(t, s.prune(context.Background(), indexToHash(99)))
	assert.Equal(t, 1, len(s.nodes), "Incorrect nodes count")
	assert.Equal(t, 1, len(s.nodesIndices), "Incorrect node indices count")
}

func TestStore_Prune_MoreThanOnce(t *testing.T) {
	// Define 100 nodes in store.
	numOfNodes := 100
	indices := make(map[[32]byte]uint64)
	nodes := make([]*Node, 0)
	for i := 0; i < numOfNodes; i++ {
		indices[indexToHash(uint64(i))] = uint64(i)
		nodes = append(nodes, &Node{slot: uint64(i), root: indexToHash(uint64(i)),
			bestDescendant: NonExistentNode, bestChild: NonExistentNode})
	}

	s := &Store{nodes: nodes, nodesIndices: indices}

	// Finalized root is at index 11 so everything before 11 should be pruned.
	require.NoError(t, s.prune(context.Background(), indexToHash(10)))
	assert.Equal(t, 90, len(s.nodes), "Incorrect nodes count")
	assert.Equal(t, 90, len(s.nodesIndices), "Incorrect node indices count")

	// One more time.
	require.NoError(t, s.prune(context.Background(), indexToHash(20)))
	assert.Equal(t, 80, len(s.nodes), "Incorrect nodes count")
	assert.Equal(t, 80, len(s.nodesIndices), "Incorrect node indices count")
}
func TestStore_LeadsToViableHead(t *testing.T) {
	tests := []struct {
		n              *Node
		justifiedEpoch uint64
		finalizedEpoch uint64
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

func TestStore_ViableForHead(t *testing.T) {
	tests := []struct {
		n              *Node
		justifiedEpoch uint64
		finalizedEpoch uint64
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
