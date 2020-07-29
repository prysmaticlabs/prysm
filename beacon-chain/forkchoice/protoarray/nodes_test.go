package protoarray

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestStore_Head_UnknownJustifiedRoot(t *testing.T) {
	s := &Store{NodeIndices: make(map[[32]byte]uint64)}

	_, err := s.head(context.Background(), [32]byte{})
	assert.ErrorContains(t, errUnknownJustifiedRoot.Error(), err)
}

func TestStore_Head_UnknownJustifiedIndex(t *testing.T) {
	r := [32]byte{'A'}
	indices := make(map[[32]byte]uint64)
	indices[r] = 1
	s := &Store{NodeIndices: indices}

	_, err := s.head(context.Background(), r)
	assert.ErrorContains(t, errInvalidJustifiedIndex.Error(), err)
}

func TestStore_Head_Itself(t *testing.T) {
	r := [32]byte{'A'}
	indices := make(map[[32]byte]uint64)
	indices[r] = 0

	// Since the justified node does not have a best descendant so the best node
	// is itself.
	s := &Store{NodeIndices: indices, Nodes: []*Node{{Root: r, BestDescendant: NonExistentNode}}}
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
	s := &Store{NodeIndices: indices, Nodes: []*Node{{Root: r, BestDescendant: 1}, {Root: best}}}
	h, err := s.head(context.Background(), r)
	require.NoError(t, err)
	assert.Equal(t, best, h)
}

func TestStore_Insert_UnknownParent(t *testing.T) {
	// The new node does not have a parent.
	s := &Store{NodeIndices: make(map[[32]byte]uint64)}
	require.NoError(t, s.insert(context.Background(), 100, [32]byte{'A'}, [32]byte{'B'}, [32]byte{}, 1, 1))
	assert.Equal(t, 1, len(s.Nodes), "Did not insert block")
	assert.Equal(t, 1, len(s.NodeIndices), "Did not insert block")
	assert.Equal(t, NonExistentNode, s.Nodes[0].Parent, "Incorrect parent")
	assert.Equal(t, uint64(1), s.Nodes[0].JustifiedEpoch, "Incorrect justification")
	assert.Equal(t, uint64(1), s.Nodes[0].FinalizedEpoch, "Incorrect finalization")
	assert.Equal(t, [32]byte{'A'}, s.Nodes[0].Root, "Incorrect root")
}

func TestStore_Insert_KnownParent(t *testing.T) {
	// Similar to UnknownParent test, but this time the new node has a valid parent already in store.
	// The new node builds on top of the parent.
	s := &Store{NodeIndices: make(map[[32]byte]uint64)}
	s.Nodes = []*Node{{}}
	p := [32]byte{'B'}
	s.NodeIndices[p] = 0
	require.NoError(t, s.insert(context.Background(), 100, [32]byte{'A'}, p, [32]byte{}, 1, 1))
	assert.Equal(t, 2, len(s.Nodes), "Did not insert block")
	assert.Equal(t, 2, len(s.NodeIndices), "Did not insert block")
	assert.Equal(t, uint64(0), s.Nodes[1].Parent, "Incorrect parent")
	assert.Equal(t, uint64(1), s.Nodes[1].JustifiedEpoch, "Incorrect justification")
	assert.Equal(t, uint64(1), s.Nodes[1].FinalizedEpoch, "Incorrect finalization")
	assert.Equal(t, [32]byte{'A'}, s.Nodes[1].Root, "Incorrect root")
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
	assert.Equal(t, uint64(1), s.JustifiedEpoch, "Did not update justified epoch")
	assert.Equal(t, uint64(1), s.FinalizedEpoch, "Did not update finalized epoch")
}

func TestStore_ApplyScoreChanges_UpdateWeightsPositiveDelta(t *testing.T) {
	// Construct 3 Nodes with weight 100 on each node. The 3 Nodes linked to each other.
	s := &Store{Nodes: []*Node{
		{Root: [32]byte{'A'}, Weight: 100},
		{Root: [32]byte{'A'}, Weight: 100},
		{Parent: 1, Root: [32]byte{'A'}, Weight: 100}}}

	// Each node gets one unique vote. The weight should look like 103 <- 102 <- 101 because
	// they get propagated back.
	require.NoError(t, s.applyWeightChanges(context.Background(), 0, 0, []int{1, 1, 1}))
	assert.Equal(t, uint64(103), s.Nodes[0].Weight)
	assert.Equal(t, uint64(102), s.Nodes[1].Weight)
	assert.Equal(t, uint64(101), s.Nodes[2].Weight)
}

func TestStore_ApplyScoreChanges_UpdateWeightsNegativeDelta(t *testing.T) {
	// Construct 3 Nodes with weight 100 on each node. The 3 Nodes linked to each other.
	s := &Store{Nodes: []*Node{
		{Root: [32]byte{'A'}, Weight: 100},
		{Root: [32]byte{'A'}, Weight: 100},
		{Parent: 1, Root: [32]byte{'A'}, Weight: 100}}}

	// Each node gets one unique vote which contributes to negative delta.
	// The weight should look like 97 <- 98 <- 99 because they get propagated back.
	require.NoError(t, s.applyWeightChanges(context.Background(), 0, 0, []int{-1, -1, -1}))
	assert.Equal(t, uint64(97), s.Nodes[0].Weight)
	assert.Equal(t, uint64(98), s.Nodes[1].Weight)
	assert.Equal(t, uint64(99), s.Nodes[2].Weight)
}

func TestStore_ApplyScoreChanges_UpdateWeightsMixedDelta(t *testing.T) {
	// Construct 3 Nodes with weight 100 on each node. The 3 Nodes linked to each other.
	s := &Store{Nodes: []*Node{
		{Root: [32]byte{'A'}, Weight: 100},
		{Root: [32]byte{'A'}, Weight: 100},
		{Parent: 1, Root: [32]byte{'A'}, Weight: 100}}}

	// Each node gets one mixed vote. The weight should look like 100 <- 200 <- 250.
	require.NoError(t, s.applyWeightChanges(context.Background(), 0, 0, []int{-100, -50, 150}))
	assert.Equal(t, uint64(100), s.Nodes[0].Weight)
	assert.Equal(t, uint64(200), s.Nodes[1].Weight)
	assert.Equal(t, uint64(250), s.Nodes[2].Weight)
}

func TestStore_UpdateBestChildAndDescendant_RemoveChild(t *testing.T) {
	// Make parent's best child equal's to input child index and child is not viable.
	s := &Store{Nodes: []*Node{{BestChild: 1}, {}}, JustifiedEpoch: 1, FinalizedEpoch: 1}
	require.NoError(t, s.updateBestChildAndDescendant(0, 1))

	// Verify parent's best child and best descendant are `none`.
	assert.Equal(t, NonExistentNode, s.Nodes[0].BestChild, "Did not get correct best child index")
	assert.Equal(t, NonExistentNode, s.Nodes[0].BestDescendant, "Did not get correct best descendant index")
}

func TestStore_UpdateBestChildAndDescendant_UpdateDescendant(t *testing.T) {
	// Make parent's best child equal to child index and child is viable.
	s := &Store{Nodes: []*Node{{BestChild: 1}, {BestDescendant: NonExistentNode}}}
	require.NoError(t, s.updateBestChildAndDescendant(0, 1))

	// Verify parent's best child is the same and best descendant is not set to child index.
	assert.Equal(t, uint64(1), s.Nodes[0].BestChild, "Did not get correct best child index")
	assert.Equal(t, uint64(1), s.Nodes[0].BestDescendant, "Did not get correct best descendant index")
}

func TestStore_UpdateBestChildAndDescendant_ChangeChildByViability(t *testing.T) {
	// Make parent's best child not equal to child index, child leads to viable index and
	// parents best child doesnt lead to viable index.
	s := &Store{
		JustifiedEpoch: 1,
		FinalizedEpoch: 1,
		Nodes: []*Node{{BestChild: 1, JustifiedEpoch: 1, FinalizedEpoch: 1},
			{BestDescendant: NonExistentNode},
			{BestDescendant: NonExistentNode, JustifiedEpoch: 1, FinalizedEpoch: 1}}}
	require.NoError(t, s.updateBestChildAndDescendant(0, 2))

	// Verify parent's best child and best descendant are set to child index.
	assert.Equal(t, uint64(2), s.Nodes[0].BestChild, "Did not get correct best child index")
	assert.Equal(t, uint64(2), s.Nodes[0].BestDescendant, "Did not get correct best descendant index")
}

func TestStore_UpdateBestChildAndDescendant_ChangeChildByWeight(t *testing.T) {
	// Make parent's best child not equal to child index, child leads to viable index and
	// parents best child leads to viable index but child has more weight than parent's best child.
	s := &Store{
		JustifiedEpoch: 1,
		FinalizedEpoch: 1,
		Nodes: []*Node{{BestChild: 1, JustifiedEpoch: 1, FinalizedEpoch: 1},
			{BestDescendant: NonExistentNode, JustifiedEpoch: 1, FinalizedEpoch: 1},
			{BestDescendant: NonExistentNode, JustifiedEpoch: 1, FinalizedEpoch: 1, Weight: 1}}}
	require.NoError(t, s.updateBestChildAndDescendant(0, 2))

	// Verify parent's best child and best descendant are set to child index.
	assert.Equal(t, uint64(2), s.Nodes[0].BestChild, "Did not get correct best child index")
	assert.Equal(t, uint64(2), s.Nodes[0].BestDescendant, "Did not get correct best descendant index")
}

func TestStore_UpdateBestChildAndDescendant_ChangeChildAtLeaf(t *testing.T) {
	// Make parent's best child to none and input child leads to viable index.
	s := &Store{
		JustifiedEpoch: 1,
		FinalizedEpoch: 1,
		Nodes: []*Node{{BestChild: NonExistentNode, JustifiedEpoch: 1, FinalizedEpoch: 1},
			{BestDescendant: NonExistentNode, JustifiedEpoch: 1, FinalizedEpoch: 1},
			{BestDescendant: NonExistentNode, JustifiedEpoch: 1, FinalizedEpoch: 1}}}
	require.NoError(t, s.updateBestChildAndDescendant(0, 2))

	// Verify parent's best child and best descendant are set to child index.
	assert.Equal(t, uint64(2), s.Nodes[0].BestChild, "Did not get correct best child index")
	assert.Equal(t, uint64(2), s.Nodes[0].BestDescendant, "Did not get correct best descendant index")
}

func TestStore_UpdateBestChildAndDescendant_NoChangeByViability(t *testing.T) {
	// Make parent's best child not equal to child index, child leads to not viable index and
	// parents best child leads to viable index.
	s := &Store{
		JustifiedEpoch: 1,
		FinalizedEpoch: 1,
		Nodes: []*Node{{BestChild: 1, JustifiedEpoch: 1, FinalizedEpoch: 1},
			{BestDescendant: NonExistentNode, JustifiedEpoch: 1, FinalizedEpoch: 1},
			{BestDescendant: NonExistentNode}}}
	require.NoError(t, s.updateBestChildAndDescendant(0, 2))

	// Verify parent's best child and best descendant are not changed.
	assert.Equal(t, uint64(1), s.Nodes[0].BestChild, "Did not get correct best child index")
	assert.Equal(t, uint64(0), s.Nodes[0].BestDescendant, "Did not get correct best descendant index")
}

func TestStore_UpdateBestChildAndDescendant_NoChangeByWeight(t *testing.T) {
	// Make parent's best child not equal to child index, child leads to viable index and
	// parents best child leads to viable index but parent's best child has more weight.
	s := &Store{
		JustifiedEpoch: 1,
		FinalizedEpoch: 1,
		Nodes: []*Node{{BestChild: 1, JustifiedEpoch: 1, FinalizedEpoch: 1},
			{BestDescendant: NonExistentNode, JustifiedEpoch: 1, FinalizedEpoch: 1, Weight: 1},
			{BestDescendant: NonExistentNode, JustifiedEpoch: 1, FinalizedEpoch: 1}}}
	require.NoError(t, s.updateBestChildAndDescendant(0, 2))

	// Verify parent's best child and best descendant are not changed.
	assert.Equal(t, uint64(1), s.Nodes[0].BestChild, "Did not get correct best child index")
	assert.Equal(t, uint64(0), s.Nodes[0].BestDescendant, "Did not get correct best descendant index")
}

func TestStore_UpdateBestChildAndDescendant_NoChangeAtLeaf(t *testing.T) {
	// Make parent's best child to none and input child does not lead to viable index.
	s := &Store{
		JustifiedEpoch: 1,
		FinalizedEpoch: 1,
		Nodes: []*Node{{BestChild: NonExistentNode, JustifiedEpoch: 1, FinalizedEpoch: 1},
			{BestDescendant: NonExistentNode, JustifiedEpoch: 1, FinalizedEpoch: 1},
			{BestDescendant: NonExistentNode}}}
	require.NoError(t, s.updateBestChildAndDescendant(0, 2))

	// Verify parent's best child and best descendant are not changed.
	assert.Equal(t, NonExistentNode, s.Nodes[0].BestChild, "Did not get correct best child index")
	assert.Equal(t, uint64(0), s.Nodes[0].BestDescendant, "Did not get correct best descendant index")
}

func TestStore_Prune_LessThanThreshold(t *testing.T) {
	// Define 100 Nodes in store.
	numOfNodes := 100
	indices := make(map[[32]byte]uint64)
	nodes := make([]*Node, 0)
	for i := 0; i < numOfNodes; i++ {
		indices[indexToHash(uint64(i))] = uint64(i)
		nodes = append(nodes, &Node{Slot: uint64(i)})
	}

	s := &Store{Nodes: nodes, NodeIndices: indices, PruneThreshold: 100}

	// Finalized root is at index 99 so everything before 99 should be pruned,
	// but PruneThreshold is at 100 so nothing will be pruned.
	require.NoError(t, s.prune(context.Background(), indexToHash(99)))
	assert.Equal(t, 100, len(s.Nodes), "Incorrect nodes count")
	assert.Equal(t, 100, len(s.NodeIndices), "Incorrect node indices count")
}

func TestStore_Prune_MoreThanThreshold(t *testing.T) {
	// Define 100 Nodes in store.
	numOfNodes := 100
	indices := make(map[[32]byte]uint64)
	nodes := make([]*Node, 0)
	for i := 0; i < numOfNodes; i++ {
		indices[indexToHash(uint64(i))] = uint64(i)
		nodes = append(nodes, &Node{Slot: uint64(i), Root: indexToHash(uint64(i)),
			BestDescendant: NonExistentNode, BestChild: NonExistentNode})
	}

	s := &Store{Nodes: nodes, NodeIndices: indices}

	// Finalized root is at index 99 so everything before 99 should be pruned.
	require.NoError(t, s.prune(context.Background(), indexToHash(99)))
	assert.Equal(t, 1, len(s.Nodes), "Incorrect nodes count")
	assert.Equal(t, 1, len(s.NodeIndices), "Incorrect node indices count")
}

func TestStore_Prune_MoreThanOnce(t *testing.T) {
	// Define 100 Nodes in store.
	numOfNodes := 100
	indices := make(map[[32]byte]uint64)
	nodes := make([]*Node, 0)
	for i := 0; i < numOfNodes; i++ {
		indices[indexToHash(uint64(i))] = uint64(i)
		nodes = append(nodes, &Node{Slot: uint64(i), Root: indexToHash(uint64(i)),
			BestDescendant: NonExistentNode, BestChild: NonExistentNode})
	}

	s := &Store{Nodes: nodes, NodeIndices: indices}

	// Finalized root is at index 11 so everything before 11 should be pruned.
	require.NoError(t, s.prune(context.Background(), indexToHash(10)))
	assert.Equal(t, 90, len(s.Nodes), "Incorrect nodes count")
	assert.Equal(t, 90, len(s.NodeIndices), "Incorrect node indices count")

	// One more time.
	require.NoError(t, s.prune(context.Background(), indexToHash(20)))
	assert.Equal(t, 80, len(s.Nodes), "Incorrect nodes count")
	assert.Equal(t, 80, len(s.NodeIndices), "Incorrect node indices count")
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
		{&Node{FinalizedEpoch: 1, JustifiedEpoch: 1}, 1, 1, true},
		{&Node{FinalizedEpoch: 1, JustifiedEpoch: 1}, 2, 2, false},
		{&Node{FinalizedEpoch: 3, JustifiedEpoch: 4}, 4, 3, true},
	}
	for _, tc := range tests {
		s := &Store{
			JustifiedEpoch: tc.justifiedEpoch,
			FinalizedEpoch: tc.finalizedEpoch,
			Nodes:          []*Node{tc.n},
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
		{&Node{FinalizedEpoch: 1, JustifiedEpoch: 1}, 1, 1, true},
		{&Node{FinalizedEpoch: 1, JustifiedEpoch: 1}, 2, 2, false},
		{&Node{FinalizedEpoch: 3, JustifiedEpoch: 4}, 4, 3, true},
	}
	for _, tc := range tests {
		s := &Store{
			JustifiedEpoch: tc.justifiedEpoch,
			FinalizedEpoch: tc.finalizedEpoch,
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
			n: []*Node{{Parent: NonExistentNode}}, want: false},
		{m: map[[32]byte]uint64{{'a'}: 0},
			n: []*Node{{Parent: 0}}, r: [32]byte{'a'},
			want: true},
	}
	for _, tc := range tests {
		f := &ForkChoice{store: &Store{
			NodeIndices: tc.m,
			Nodes:       tc.n,
		}}
		assert.Equal(t, tc.want, f.HasParent(tc.r))
	}
}

func TestStore_AncestorRoot(t *testing.T) {
	ctx := context.Background()
	f := &ForkChoice{store: &Store{}}
	f.store.NodeIndices = map[[32]byte]uint64{}
	_, err := f.AncestorRoot(ctx, [32]byte{'a'}, 0)
	assert.ErrorContains(t, "node does not exist", err)
	f.store.NodeIndices[[32]byte{'a'}] = 0
	_, err = f.AncestorRoot(ctx, [32]byte{'a'}, 0)
	assert.ErrorContains(t, "node index out of range", err)
	f.store.NodeIndices[[32]byte{'b'}] = 1
	f.store.NodeIndices[[32]byte{'c'}] = 2
	f.store.Nodes = []*Node{
		{Slot: 1, Root: [32]byte{'a'}, Parent: NonExistentNode},
		{Slot: 2, Root: [32]byte{'b'}, Parent: 0},
		{Slot: 3, Root: [32]byte{'c'}, Parent: 1},
	}

	r, err := f.AncestorRoot(ctx, [32]byte{'c'}, 1)
	require.NoError(t, err)
	assert.Equal(t, bytesutil.ToBytes32(r), [32]byte{'a'})
	r, err = f.AncestorRoot(ctx, [32]byte{'c'}, 2)
	require.NoError(t, err)
	assert.Equal(t, bytesutil.ToBytes32(r), [32]byte{'b'})
}
