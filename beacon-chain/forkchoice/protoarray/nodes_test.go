package protoarray

import (
	"context"
	"testing"
)

func TestStore_Head_UnknownJustifiedRoot(t *testing.T) {
	s := &Store{nodeIndices: make(map[[32]byte]uint64)}

	if _, err := s.head(context.Background(), [32]byte{}); err.Error() != errUnknownJustifiedRoot.Error() {
		t.Fatal("Did not get wanted error")
	}
}

func TestStore_Head_UnknownJustifiedIndex(t *testing.T) {
	r := [32]byte{'A'}
	indices := make(map[[32]byte]uint64)
	indices[r] = 1
	s := &Store{nodeIndices: indices}

	if _, err := s.head(context.Background(), r); err.Error() != errInvalidJustifiedIndex.Error() {
		t.Fatal("Did not get wanted error")
	}
}

func TestStore_Head_Itself(t *testing.T) {
	r := [32]byte{'A'}
	indices := make(map[[32]byte]uint64)
	indices[r] = 0

	// Since the justified node does not have a best descendant so the best node
	// is itself.
	s := &Store{nodeIndices: indices, nodes: []*Node{{root: r, BestDescendent: nonExistentNode}}}
	h, err := s.head(context.Background(), r)
	if err != nil {
		t.Fatal("Did not get wanted error")
	}

	if h != r {
		t.Error("Did not get wanted head")
	}
}

func TestStore_Head_BestDescendant(t *testing.T) {
	r := [32]byte{'A'}
	best := [32]byte{'B'}
	indices := make(map[[32]byte]uint64)
	indices[r] = 0

	// Since the justified node's best descendent is at index 1 and it's root is `best`,
	// the head should be `best`.
	s := &Store{nodeIndices: indices, nodes: []*Node{{root: r, BestDescendent: 1}, {root: best}}}
	h, err := s.head(context.Background(), r)
	if err != nil {
		t.Fatal("Did not get wanted error")
	}

	if h != best {
		t.Error("Did not get wanted head")
	}
}

func TestStore_Insert_UnknownParent(t *testing.T) {
	// The new node does not have a parent.
	s := &Store{nodeIndices: make(map[[32]byte]uint64)}
	if err := s.insert(context.Background(), 100, [32]byte{'A'}, [32]byte{'B'}, 1, 1); err != nil {
		t.Fatal(err)
	}

	if len(s.nodes) != 1 {
		t.Error("Did not insert block")
	}
	if len(s.nodeIndices) != 1 {
		t.Error("Did not insert block")
	}
	if s.nodes[0].Parent != nonExistentNode {
		t.Error("Incorrect parent")
	}
	if s.nodes[0].justifiedEpoch != 1 {
		t.Error("Incorrect justification")
	}
	if s.nodes[0].finalizedEpoch != 1 {
		t.Error("Incorrect finalization")
	}
	if s.nodes[0].root != [32]byte{'A'} {
		t.Error("Incorrect root")
	}
}

func TestStore_Insert_KnownParent(t *testing.T) {
	// Similar to UnknownParent test, but this time the new node has a valid parent already in store.
	// The new node builds on top of the parent.
	s := &Store{nodeIndices: make(map[[32]byte]uint64)}
	s.nodes = []*Node{{}}
	p := [32]byte{'B'}
	s.nodeIndices[p] = 0
	if err := s.insert(context.Background(), 100, [32]byte{'A'}, p, 1, 1); err != nil {
		t.Fatal(err)
	}

	if len(s.nodes) != 2 {
		t.Error("Did not insert block")
	}
	if len(s.nodeIndices) != 2 {
		t.Error("Did not insert block")
	}
	if s.nodes[1].Parent != 0 {
		t.Error("Incorrect parent")
	}
	if s.nodes[1].justifiedEpoch != 1 {
		t.Error("Incorrect justification")
	}
	if s.nodes[1].finalizedEpoch != 1 {
		t.Error("Incorrect finalization")
	}
	if s.nodes[1].root != [32]byte{'A'} {
		t.Error("Incorrect root")
	}
}

func TestStore_ApplyScoreChanges_InvalidDeltaLength(t *testing.T) {
	s := &Store{}

	// This will fail because node indices has length of 0, and delta list has a length of 1.
	if err := s.applyWeightChanges(context.Background(), 0, 0, []int{1}); err.Error() != errInvalidDeltaLength.Error() {
		t.Error("Did not get wanted error")
	}
}

func TestStore_ApplyScoreChanges_UpdateEpochs(t *testing.T) {
	s := &Store{}

	// The justified and finalized epochs in Store should be updated to 1 and 1 given the following input.
	if err := s.applyWeightChanges(context.Background(), 1, 1, []int{}); err != nil {
		t.Error("Did not get wanted error")
	}

	if s.justifiedEpoch != 1 {
		t.Error("Did not update justified epoch")
	}
	if s.finalizedEpoch != 1 {
		t.Error("Did not update justified epoch")
	}
}

func TestStore_ApplyScoreChanges_UpdateWeightsPositiveDelta(t *testing.T) {
	// Construct 3 nodes with weight 100 on each node. The 3 nodes linked to each other.
	s := &Store{nodes: []*Node{
		{root: [32]byte{'A'}, Weight: 100},
		{root: [32]byte{'A'}, Weight: 100},
		{Parent: 1, root: [32]byte{'A'}, Weight: 100}}}

	// Each node gets one unique vote. The weight should look like 103 <- 102 <- 101 because
	// they get propagated back.
	if err := s.applyWeightChanges(context.Background(), 0, 0, []int{1, 1, 1}); err != nil {
		t.Fatal(err)
	}

	if s.nodes[0].Weight != 103 {
		t.Error("Did not get correct weight")
	}
	if s.nodes[1].Weight != 102 {
		t.Error("Did not get correct weight")
	}
	if s.nodes[2].Weight != 101 {
		t.Error("Did not get correct weight")
	}
}

func TestStore_ApplyScoreChanges_UpdateWeightsNegativeDelta(t *testing.T) {
	// Construct 3 nodes with weight 100 on each node. The 3 nodes linked to each other.
	s := &Store{nodes: []*Node{
		{root: [32]byte{'A'}, Weight: 100},
		{root: [32]byte{'A'}, Weight: 100},
		{Parent: 1, root: [32]byte{'A'}, Weight: 100}}}

	// Each node gets one unique vote which contributes to negative delta.
	// The weight should look like 97 <- 98 <- 99 because they get propagated back.
	if err := s.applyWeightChanges(context.Background(), 0, 0, []int{-1, -1, -1}); err != nil {
		t.Fatal(err)
	}

	if s.nodes[0].Weight != 97 {
		t.Error("Did not get correct weight")
	}
	if s.nodes[1].Weight != 98 {
		t.Error("Did not get correct weight")
	}
	if s.nodes[2].Weight != 99 {
		t.Error("Did not get correct weight")
	}
}

func TestStore_ApplyScoreChanges_UpdateWeightsMixedDelta(t *testing.T) {
	// Construct 3 nodes with weight 100 on each node. The 3 nodes linked to each other.
	s := &Store{nodes: []*Node{
		{root: [32]byte{'A'}, Weight: 100},
		{root: [32]byte{'A'}, Weight: 100},
		{Parent: 1, root: [32]byte{'A'}, Weight: 100}}}

	// Each node gets one mixed vote. The weight should look like 100 <- 200 <- 250.
	if err := s.applyWeightChanges(context.Background(), 0, 0, []int{-100, -50, 150}); err != nil {
		t.Fatal(err)
	}

	if s.nodes[0].Weight != 100 {
		t.Error("Did not get correct weight")
	}
	if s.nodes[1].Weight != 200 {
		t.Error("Did not get correct weight")
	}
	if s.nodes[2].Weight != 250 {
		t.Error("Did not get correct weight")
	}
}

func TestStore_UpdateBestChildAndDescendant_RemoveChild(t *testing.T) {
	// Make parent's best child equal's to input child index and child is not viable.
	s := &Store{nodes: []*Node{{bestChild: 1}, {}}, justifiedEpoch: 1, finalizedEpoch: 1}

	if err := s.updateBestChildAndDescendant(0, 1); err != nil {
		t.Fatal(err)
	}

	// Verify parent's best child and best descendant are `none`.
	if s.nodes[0].bestChild != nonExistentNode {
		t.Error("Did not get correct best child index")
	}
	if s.nodes[0].BestDescendent != nonExistentNode {
		t.Error("Did not get correct best descendant index")
	}
}

func TestStore_UpdateBestChildAndDescendant_UpdateDescendant(t *testing.T) {
	// Make parent's best child equal to child index and child is viable.
	s := &Store{nodes: []*Node{{bestChild: 1}, {BestDescendent: nonExistentNode}}}

	if err := s.updateBestChildAndDescendant(0, 1); err != nil {
		t.Fatal(err)
	}

	// Verify parent's best child is the same and best descendant is not set to child index.
	if s.nodes[0].bestChild != 1 {
		t.Error("Did not get correct best child index")
	}
	if s.nodes[0].BestDescendent != 1 {
		t.Error("Did not get correct best descendant index")
	}
}

func TestStore_UpdateBestChildAndDescendant_ChangeChildByViability(t *testing.T) {
	// Make parent's best child not equal to child index, child leads to viable index and
	// parents best child doesnt lead to viable index.
	s := &Store{
		justifiedEpoch: 1,
		finalizedEpoch: 1,
		nodes: []*Node{{bestChild: 1, justifiedEpoch: 1, finalizedEpoch: 1},
			{BestDescendent: nonExistentNode},
			{BestDescendent: nonExistentNode, justifiedEpoch: 1, finalizedEpoch: 1}}}

	if err := s.updateBestChildAndDescendant(0, 2); err != nil {
		t.Fatal(err)
	}

	// Verify parent's best child and best descendant are set to child index.
	if s.nodes[0].bestChild != 2 {
		t.Error("Did not get correct best child index")
	}
	if s.nodes[0].BestDescendent != 2 {
		t.Error("Did not get correct best descendant index")
	}
}

func TestStore_UpdateBestChildAndDescendant_ChangeChildByWeight(t *testing.T) {
	// Make parent's best child not equal to child index, child leads to viable index and
	// parents best child leads to viable index but child has more weight than parent's best child.
	s := &Store{
		justifiedEpoch: 1,
		finalizedEpoch: 1,
		nodes: []*Node{{bestChild: 1, justifiedEpoch: 1, finalizedEpoch: 1},
			{BestDescendent: nonExistentNode, justifiedEpoch: 1, finalizedEpoch: 1},
			{BestDescendent: nonExistentNode, justifiedEpoch: 1, finalizedEpoch: 1, Weight: 1}}}

	if err := s.updateBestChildAndDescendant(0, 2); err != nil {
		t.Fatal(err)
	}

	// Verify parent's best child and best descendant are set to child index.
	if s.nodes[0].bestChild != 2 {
		t.Error("Did not get correct best child index")
	}
	if s.nodes[0].BestDescendent != 2 {
		t.Error("Did not get correct best descendant index")
	}
}

func TestStore_UpdateBestChildAndDescendant_ChangeChildAtLeaf(t *testing.T) {
	// Make parent's best child to none and input child leads to viable index.
	s := &Store{
		justifiedEpoch: 1,
		finalizedEpoch: 1,
		nodes: []*Node{{bestChild: nonExistentNode, justifiedEpoch: 1, finalizedEpoch: 1},
			{BestDescendent: nonExistentNode, justifiedEpoch: 1, finalizedEpoch: 1},
			{BestDescendent: nonExistentNode, justifiedEpoch: 1, finalizedEpoch: 1}}}

	if err := s.updateBestChildAndDescendant(0, 2); err != nil {
		t.Fatal(err)
	}

	// Verify parent's best child and best descendant are set to child index.
	if s.nodes[0].bestChild != 2 {
		t.Error("Did not get correct best child index")
	}
	if s.nodes[0].BestDescendent != 2 {
		t.Error("Did not get correct best descendant index")
	}
}

func TestStore_UpdateBestChildAndDescendant_NoChangeByViability(t *testing.T) {
	// Make parent's best child not equal to child index, child leads to not viable index and
	// parents best child leads to viable index.
	s := &Store{
		justifiedEpoch: 1,
		finalizedEpoch: 1,
		nodes: []*Node{{bestChild: 1, justifiedEpoch: 1, finalizedEpoch: 1},
			{BestDescendent: nonExistentNode, justifiedEpoch: 1, finalizedEpoch: 1},
			{BestDescendent: nonExistentNode}}}

	if err := s.updateBestChildAndDescendant(0, 2); err != nil {
		t.Fatal(err)
	}

	// Verify parent's best child and best descendant are not changed.
	if s.nodes[0].bestChild != 1 {
		t.Error("Did not get correct best child index")
	}
	if s.nodes[0].BestDescendent != 0 {
		t.Error("Did not get correct best descendant index")
	}
}

func TestStore_UpdateBestChildAndDescendant_NoChangeByWeight(t *testing.T) {
	// Make parent's best child not equal to child index, child leads to viable index and
	// parents best child leads to viable index but parent's best child has more weight.
	s := &Store{
		justifiedEpoch: 1,
		finalizedEpoch: 1,
		nodes: []*Node{{bestChild: 1, justifiedEpoch: 1, finalizedEpoch: 1},
			{BestDescendent: nonExistentNode, justifiedEpoch: 1, finalizedEpoch: 1, Weight: 1},
			{BestDescendent: nonExistentNode, justifiedEpoch: 1, finalizedEpoch: 1}}}

	if err := s.updateBestChildAndDescendant(0, 2); err != nil {
		t.Fatal(err)
	}

	// Verify parent's best child and best descendant are not changed.
	if s.nodes[0].bestChild != 1 {
		t.Error("Did not get correct best child index")
	}
	if s.nodes[0].BestDescendent != 0 {
		t.Error("Did not get correct best descendant index")
	}
}

func TestStore_UpdateBestChildAndDescendant_NoChangeAtLeaf(t *testing.T) {
	// Make parent's best child to none and input child does not lead to viable index.
	s := &Store{
		justifiedEpoch: 1,
		finalizedEpoch: 1,
		nodes: []*Node{{bestChild: nonExistentNode, justifiedEpoch: 1, finalizedEpoch: 1},
			{BestDescendent: nonExistentNode, justifiedEpoch: 1, finalizedEpoch: 1},
			{BestDescendent: nonExistentNode}}}

	if err := s.updateBestChildAndDescendant(0, 2); err != nil {
		t.Fatal(err)
	}

	// Verify parent's best child and best descendant are not changed.
	if s.nodes[0].bestChild != nonExistentNode {
		t.Error("Did not get correct best child index")
	}
	if s.nodes[0].BestDescendent != 0 {
		t.Error("Did not get correct best descendant index")
	}
}

func TestStore_Prune_LessThanThreshold(t *testing.T) {
	// Define 100 nodes in store.
	numOfNodes := 100
	indices := make(map[[32]byte]uint64)
	nodes := make([]*Node, 0)
	for i := 0; i < numOfNodes; i++ {
		indices[indexToHash(uint64(i))] = uint64(i)
		nodes = append(nodes, &Node{Slot: uint64(i)})
	}

	s := &Store{nodes: nodes, nodeIndices: indices, pruneThreshold: 100}

	// Finalized root is at index 99 so everything before 99 should be pruned,
	// but pruneThreshold is at 100 so nothing will be pruned.
	if err := s.prune(context.Background(), indexToHash(99)); err != nil {
		t.Fatal(err)
	}

	if len(s.nodes) != 100 {
		t.Fatal("Incorrect nodes count")
	}
	if len(s.nodeIndices) != 100 {
		t.Fatal("Incorrect node indices count")
	}
}

func TestStore_Prune_MoreThanThreshold(t *testing.T) {
	// Define 100 nodes in store.
	numOfNodes := 100
	indices := make(map[[32]byte]uint64)
	nodes := make([]*Node, 0)
	for i := 0; i < numOfNodes; i++ {
		indices[indexToHash(uint64(i))] = uint64(i)
		nodes = append(nodes, &Node{Slot: uint64(i), root: indexToHash(uint64(i)),
			BestDescendent: nonExistentNode, bestChild: nonExistentNode})
	}

	s := &Store{nodes: nodes, nodeIndices: indices}

	// Finalized root is at index 99 so everything before 99 should be pruned.
	if err := s.prune(context.Background(), indexToHash(99)); err != nil {
		t.Fatal(err)
	}

	if len(s.nodes) != 1 {
		t.Error("Incorrect nodes count")
	}
	if len(s.nodeIndices) != 1 {
		t.Error("Incorrect node indices count")
	}
}

func TestStore_Prune_MoreThanOnce(t *testing.T) {
	// Define 100 nodes in store.
	numOfNodes := 100
	indices := make(map[[32]byte]uint64)
	nodes := make([]*Node, 0)
	for i := 0; i < numOfNodes; i++ {
		indices[indexToHash(uint64(i))] = uint64(i)
		nodes = append(nodes, &Node{Slot: uint64(i), root: indexToHash(uint64(i)),
			BestDescendent: nonExistentNode, bestChild: nonExistentNode})
	}

	s := &Store{nodes: nodes, nodeIndices: indices}

	// Finalized root is at index 11 so everything before 11 should be pruned.
	if err := s.prune(context.Background(), indexToHash(10)); err != nil {
		t.Fatal(err)
	}

	if len(s.nodes) != 90 {
		t.Error("Incorrect nodes count")
	}
	if len(s.nodeIndices) != 90 {
		t.Error("Incorrect node indices count")
	}

	// One more time.
	if err := s.prune(context.Background(), indexToHash(20)); err != nil {
		t.Fatal(err)
	}

	if len(s.nodes) != 80 {
		t.Log(len(s.nodes))
		t.Error("Incorrect nodes count")
	}
	if len(s.nodeIndices) != 80 {
		t.Error("Incorrect node indices count")
	}
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
		if err != nil {
			t.Fatal(err)
		}
		if got != tc.want {
			t.Errorf("viableForHead() = %v, want %v", got, tc.want)
		}
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
		if got := s.viableForHead(tc.n); got != tc.want {
			t.Errorf("viableForHead() = %v, want %v", got, tc.want)
		}
	}
}
