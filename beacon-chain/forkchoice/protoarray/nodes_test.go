package protoarray

import (
	"context"
	"testing"
)

func TestStore_Head_UnknownJustifiedRoot(t *testing.T) {
	s := &Store{NodeIndices: make(map[[32]byte]uint64)}

	if _, err := s.head(context.Background(), [32]byte{}); err.Error() != errUnknownJustifiedRoot.Error() {
		t.Fatal("Did not get wanted error")
	}
}

func TestStore_Head_UnknownJustifiedIndex(t *testing.T) {
	r := [32]byte{'A'}
	indices := make(map[[32]byte]uint64)
	indices[r] = 1
	s := &Store{NodeIndices: indices}

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
	s := &Store{NodeIndices: indices, Nodes: []*Node{{Root: r, BestDescendant: NonExistentNode}}}
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
	s := &Store{NodeIndices: indices, Nodes: []*Node{{Root: r, BestDescendant: 1}, {Root: best}}}
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
	s := &Store{NodeIndices: make(map[[32]byte]uint64)}
	if err := s.insert(context.Background(), 100, [32]byte{'A'}, [32]byte{'B'}, [32]byte{}, 1, 1); err != nil {
		t.Fatal(err)
	}

	if len(s.Nodes) != 1 {
		t.Error("Did not insert block")
	}
	if len(s.NodeIndices) != 1 {
		t.Error("Did not insert block")
	}
	if s.Nodes[0].Parent != NonExistentNode {
		t.Error("Incorrect parent")
	}
	if s.Nodes[0].JustifiedEpoch != 1 {
		t.Error("Incorrect justification")
	}
	if s.Nodes[0].FinalizedEpoch != 1 {
		t.Error("Incorrect finalization")
	}
	if s.Nodes[0].Root != [32]byte{'A'} {
		t.Error("Incorrect root")
	}
}

func TestStore_Insert_KnownParent(t *testing.T) {
	// Similar to UnknownParent test, but this time the new node has a valid parent already in store.
	// The new node builds on top of the parent.
	s := &Store{NodeIndices: make(map[[32]byte]uint64)}
	s.Nodes = []*Node{{}}
	p := [32]byte{'B'}
	s.NodeIndices[p] = 0
	if err := s.insert(context.Background(), 100, [32]byte{'A'}, p, [32]byte{}, 1, 1); err != nil {
		t.Fatal(err)
	}

	if len(s.Nodes) != 2 {
		t.Error("Did not insert block")
	}
	if len(s.NodeIndices) != 2 {
		t.Error("Did not insert block")
	}
	if s.Nodes[1].Parent != 0 {
		t.Error("Incorrect parent")
	}
	if s.Nodes[1].JustifiedEpoch != 1 {
		t.Error("Incorrect justification")
	}
	if s.Nodes[1].FinalizedEpoch != 1 {
		t.Error("Incorrect finalization")
	}
	if s.Nodes[1].Root != [32]byte{'A'} {
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

	if s.JustifiedEpoch != 1 {
		t.Error("Did not update justified epoch")
	}
	if s.FinalizedEpoch != 1 {
		t.Error("Did not update justified epoch")
	}
}

func TestStore_ApplyScoreChanges_UpdateWeightsPositiveDelta(t *testing.T) {
	// Construct 3 Nodes with weight 100 on each node. The 3 Nodes linked to each other.
	s := &Store{Nodes: []*Node{
		{Root: [32]byte{'A'}, Weight: 100},
		{Root: [32]byte{'A'}, Weight: 100},
		{Parent: 1, Root: [32]byte{'A'}, Weight: 100}}}

	// Each node gets one unique vote. The weight should look like 103 <- 102 <- 101 because
	// they get propagated back.
	if err := s.applyWeightChanges(context.Background(), 0, 0, []int{1, 1, 1}); err != nil {
		t.Fatal(err)
	}

	if s.Nodes[0].Weight != 103 {
		t.Error("Did not get correct weight")
	}
	if s.Nodes[1].Weight != 102 {
		t.Error("Did not get correct weight")
	}
	if s.Nodes[2].Weight != 101 {
		t.Error("Did not get correct weight")
	}
}

func TestStore_ApplyScoreChanges_UpdateWeightsNegativeDelta(t *testing.T) {
	// Construct 3 Nodes with weight 100 on each node. The 3 Nodes linked to each other.
	s := &Store{Nodes: []*Node{
		{Root: [32]byte{'A'}, Weight: 100},
		{Root: [32]byte{'A'}, Weight: 100},
		{Parent: 1, Root: [32]byte{'A'}, Weight: 100}}}

	// Each node gets one unique vote which contributes to negative delta.
	// The weight should look like 97 <- 98 <- 99 because they get propagated back.
	if err := s.applyWeightChanges(context.Background(), 0, 0, []int{-1, -1, -1}); err != nil {
		t.Fatal(err)
	}

	if s.Nodes[0].Weight != 97 {
		t.Error("Did not get correct weight")
	}
	if s.Nodes[1].Weight != 98 {
		t.Error("Did not get correct weight")
	}
	if s.Nodes[2].Weight != 99 {
		t.Error("Did not get correct weight")
	}
}

func TestStore_ApplyScoreChanges_UpdateWeightsMixedDelta(t *testing.T) {
	// Construct 3 Nodes with weight 100 on each node. The 3 Nodes linked to each other.
	s := &Store{Nodes: []*Node{
		{Root: [32]byte{'A'}, Weight: 100},
		{Root: [32]byte{'A'}, Weight: 100},
		{Parent: 1, Root: [32]byte{'A'}, Weight: 100}}}

	// Each node gets one mixed vote. The weight should look like 100 <- 200 <- 250.
	if err := s.applyWeightChanges(context.Background(), 0, 0, []int{-100, -50, 150}); err != nil {
		t.Fatal(err)
	}

	if s.Nodes[0].Weight != 100 {
		t.Error("Did not get correct weight")
	}
	if s.Nodes[1].Weight != 200 {
		t.Error("Did not get correct weight")
	}
	if s.Nodes[2].Weight != 250 {
		t.Error("Did not get correct weight")
	}
}

func TestStore_UpdateBestChildAndDescendant_RemoveChild(t *testing.T) {
	// Make parent's best child equal's to input child index and child is not viable.
	s := &Store{Nodes: []*Node{{BestChild: 1}, {}}, JustifiedEpoch: 1, FinalizedEpoch: 1}

	if err := s.updateBestChildAndDescendant(0, 1); err != nil {
		t.Fatal(err)
	}

	// Verify parent's best child and best descendant are `none`.
	if s.Nodes[0].BestChild != NonExistentNode {
		t.Error("Did not get correct best child index")
	}
	if s.Nodes[0].BestDescendant != NonExistentNode {
		t.Error("Did not get correct best descendant index")
	}
}

func TestStore_UpdateBestChildAndDescendant_UpdateDescendant(t *testing.T) {
	// Make parent's best child equal to child index and child is viable.
	s := &Store{Nodes: []*Node{{BestChild: 1}, {BestDescendant: NonExistentNode}}}

	if err := s.updateBestChildAndDescendant(0, 1); err != nil {
		t.Fatal(err)
	}

	// Verify parent's best child is the same and best descendant is not set to child index.
	if s.Nodes[0].BestChild != 1 {
		t.Error("Did not get correct best child index")
	}
	if s.Nodes[0].BestDescendant != 1 {
		t.Error("Did not get correct best descendant index")
	}
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

	if err := s.updateBestChildAndDescendant(0, 2); err != nil {
		t.Fatal(err)
	}

	// Verify parent's best child and best descendant are set to child index.
	if s.Nodes[0].BestChild != 2 {
		t.Error("Did not get correct best child index")
	}
	if s.Nodes[0].BestDescendant != 2 {
		t.Error("Did not get correct best descendant index")
	}
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

	if err := s.updateBestChildAndDescendant(0, 2); err != nil {
		t.Fatal(err)
	}

	// Verify parent's best child and best descendant are set to child index.
	if s.Nodes[0].BestChild != 2 {
		t.Error("Did not get correct best child index")
	}
	if s.Nodes[0].BestDescendant != 2 {
		t.Error("Did not get correct best descendant index")
	}
}

func TestStore_UpdateBestChildAndDescendant_ChangeChildAtLeaf(t *testing.T) {
	// Make parent's best child to none and input child leads to viable index.
	s := &Store{
		JustifiedEpoch: 1,
		FinalizedEpoch: 1,
		Nodes: []*Node{{BestChild: NonExistentNode, JustifiedEpoch: 1, FinalizedEpoch: 1},
			{BestDescendant: NonExistentNode, JustifiedEpoch: 1, FinalizedEpoch: 1},
			{BestDescendant: NonExistentNode, JustifiedEpoch: 1, FinalizedEpoch: 1}}}

	if err := s.updateBestChildAndDescendant(0, 2); err != nil {
		t.Fatal(err)
	}

	// Verify parent's best child and best descendant are set to child index.
	if s.Nodes[0].BestChild != 2 {
		t.Error("Did not get correct best child index")
	}
	if s.Nodes[0].BestDescendant != 2 {
		t.Error("Did not get correct best descendant index")
	}
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

	if err := s.updateBestChildAndDescendant(0, 2); err != nil {
		t.Fatal(err)
	}

	// Verify parent's best child and best descendant are not changed.
	if s.Nodes[0].BestChild != 1 {
		t.Error("Did not get correct best child index")
	}
	if s.Nodes[0].BestDescendant != 0 {
		t.Error("Did not get correct best descendant index")
	}
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

	if err := s.updateBestChildAndDescendant(0, 2); err != nil {
		t.Fatal(err)
	}

	// Verify parent's best child and best descendant are not changed.
	if s.Nodes[0].BestChild != 1 {
		t.Error("Did not get correct best child index")
	}
	if s.Nodes[0].BestDescendant != 0 {
		t.Error("Did not get correct best descendant index")
	}
}

func TestStore_UpdateBestChildAndDescendant_NoChangeAtLeaf(t *testing.T) {
	// Make parent's best child to none and input child does not lead to viable index.
	s := &Store{
		JustifiedEpoch: 1,
		FinalizedEpoch: 1,
		Nodes: []*Node{{BestChild: NonExistentNode, JustifiedEpoch: 1, FinalizedEpoch: 1},
			{BestDescendant: NonExistentNode, JustifiedEpoch: 1, FinalizedEpoch: 1},
			{BestDescendant: NonExistentNode}}}

	if err := s.updateBestChildAndDescendant(0, 2); err != nil {
		t.Fatal(err)
	}

	// Verify parent's best child and best descendant are not changed.
	if s.Nodes[0].BestChild != NonExistentNode {
		t.Error("Did not get correct best child index")
	}
	if s.Nodes[0].BestDescendant != 0 {
		t.Error("Did not get correct best descendant index")
	}
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
	if err := s.prune(context.Background(), indexToHash(99)); err != nil {
		t.Fatal(err)
	}

	if len(s.Nodes) != 100 {
		t.Fatal("Incorrect Nodes count")
	}
	if len(s.NodeIndices) != 100 {
		t.Fatal("Incorrect node indices count")
	}
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
	if err := s.prune(context.Background(), indexToHash(99)); err != nil {
		t.Fatal(err)
	}

	if len(s.Nodes) != 1 {
		t.Error("Incorrect Nodes count")
	}
	if len(s.NodeIndices) != 1 {
		t.Error("Incorrect node indices count")
	}
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
	if err := s.prune(context.Background(), indexToHash(10)); err != nil {
		t.Fatal(err)
	}

	if len(s.Nodes) != 90 {
		t.Error("Incorrect Nodes count")
	}
	if len(s.NodeIndices) != 90 {
		t.Error("Incorrect node indices count")
	}

	// One more time.
	if err := s.prune(context.Background(), indexToHash(20)); err != nil {
		t.Fatal(err)
	}

	if len(s.Nodes) != 80 {
		t.Log(len(s.Nodes))
		t.Error("Incorrect Nodes count")
	}
	if len(s.NodeIndices) != 80 {
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
		{&Node{FinalizedEpoch: 1, JustifiedEpoch: 1}, 1, 1, true},
		{&Node{FinalizedEpoch: 1, JustifiedEpoch: 1}, 2, 2, false},
		{&Node{FinalizedEpoch: 3, JustifiedEpoch: 4}, 4, 3, true},
	}
	for _, tc := range tests {
		s := &Store{
			JustifiedEpoch: tc.justifiedEpoch,
			FinalizedEpoch: tc.finalizedEpoch,
		}
		if got := s.viableForHead(tc.n); got != tc.want {
			t.Errorf("viableForHead() = %v, want %v", got, tc.want)
		}
	}
}
