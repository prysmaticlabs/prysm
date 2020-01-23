package protoarray

import (
	"context"
	"testing"
)

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
		{root: [32]byte{'A'}, weight: 100},
		{root: [32]byte{'A'}, weight: 100},
		{parent: 1, root: [32]byte{'A'}, weight: 100}}}

	// Each node gets one unique vote. The weight should look like 103 <- 102 <- 101 because
	// they get propagated back.
	if err := s.applyWeightChanges(context.Background(), 0, 0, []int{1, 1, 1}); err != nil {
		t.Fatal(err)
	}

	if s.nodes[0].weight != 103 {
		t.Error("Did not get correct weight")
	}
	if s.nodes[1].weight != 102 {
		t.Error("Did not get correct weight")
	}
	if s.nodes[2].weight != 101 {
		t.Error("Did not get correct weight")
	}
}

func TestStore_ApplyScoreChanges_UpdateWeightsNegativeDelta(t *testing.T) {
	// Construct 3 nodes with weight 100 on each node. The 3 nodes linked to each other.
	s := &Store{nodes: []*Node{
		{root: [32]byte{'A'}, weight: 100},
		{root: [32]byte{'A'}, weight: 100},
		{parent: 1, root: [32]byte{'A'}, weight: 100}}}

	// Each node gets one unique vote which contributes to negative delta.
	// The weight should look like 97 <- 98 <- 99 because they get propagated back.
	if err := s.applyWeightChanges(context.Background(), 0, 0, []int{-1, -1, -1}); err != nil {
		t.Fatal(err)
	}

	if s.nodes[0].weight != 97 {
		t.Error("Did not get correct weight")
	}
	if s.nodes[1].weight != 98 {
		t.Error("Did not get correct weight")
	}
	if s.nodes[2].weight != 99 {
		t.Error("Did not get correct weight")
	}
}

func TestStore_ApplyScoreChanges_UpdateWeightsMixedDelta(t *testing.T) {
	// Construct 3 nodes with weight 100 on each node. The 3 nodes linked to each other.
	s := &Store{nodes: []*Node{
		{root: [32]byte{'A'}, weight: 100},
		{root: [32]byte{'A'}, weight: 100},
		{parent: 1, root: [32]byte{'A'}, weight: 100}}}

	// Each node gets one mixed vote. The weight should look like 100 <- 200 <- 250.
	if err := s.applyWeightChanges(context.Background(), 0, 0, []int{-100, -50, 150}); err != nil {
		t.Fatal(err)
	}

	if s.nodes[0].weight != 100 {
		t.Error("Did not get correct weight")
	}
	if s.nodes[1].weight != 200 {
		t.Error("Did not get correct weight")
	}
	if s.nodes[2].weight != 250 {
		t.Error("Did not get correct weight")
	}
}

func TestStore_UpdateBestChildAndDescendant_RemoveChild(t *testing.T) {
	// Make parent's best child equal's to input child index and child is not viable.
	s := &Store{nodes: []*Node{{bestChild: 1}, {}}, justifiedEpoch: 1, finalizedEpoch: 1}

	if err := s.updateBestChildAndDescendant(context.Background(), 0, 1); err != nil {
		t.Fatal(err)
	}

	// Verify parent's best child and best descendant are `none`.
	if s.nodes[0].bestChild != nonExistentNode {
		t.Error("Did not get correct best child index")
	}
	if s.nodes[0].bestDescendant != nonExistentNode {
		t.Error("Did not get correct best descendant index")
	}
}

func TestStore_UpdateBestChildAndDescendant_UpdateDescendant(t *testing.T) {
	// Make parent's best child equal to child index and child is viable.
	s := &Store{nodes: []*Node{{bestChild: 1}, {bestDescendant: nonExistentNode}}}

	if err := s.updateBestChildAndDescendant(context.Background(), 0, 1); err != nil {
		t.Fatal(err)
	}

	// Verify parent's best child is the same and best descendant is not set to child index.
	if s.nodes[0].bestChild != 1 {
		t.Error("Did not get correct best child index")
	}
	if s.nodes[0].bestDescendant != 1 {
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
			{bestDescendant: nonExistentNode},
			{bestDescendant: nonExistentNode, justifiedEpoch: 1, finalizedEpoch: 1}}}

	if err := s.updateBestChildAndDescendant(context.Background(), 0, 2); err != nil {
		t.Fatal(err)
	}

	// Verify parent's best child and best descendant are set to child index.
	if s.nodes[0].bestChild != 2 {
		t.Error("Did not get correct best child index")
	}
	if s.nodes[0].bestDescendant != 2 {
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
			{bestDescendant: nonExistentNode, justifiedEpoch: 1, finalizedEpoch: 1},
			{bestDescendant: nonExistentNode, justifiedEpoch: 1, finalizedEpoch: 1, weight: 1}}}

	if err := s.updateBestChildAndDescendant(context.Background(), 0, 2); err != nil {
		t.Fatal(err)
	}

	// Verify parent's best child and best descendant are set to child index.
	if s.nodes[0].bestChild != 2 {
		t.Error("Did not get correct best child index")
	}
	if s.nodes[0].bestDescendant != 2 {
		t.Error("Did not get correct best descendant index")
	}
}

func TestStore_UpdateBestChildAndDescendant_ChangeChildAtLeaf(t *testing.T) {
	// Make parent's best child to none and input child leads to viable index.
	s := &Store{
		justifiedEpoch: 1,
		finalizedEpoch: 1,
		nodes: []*Node{{bestChild: nonExistentNode, justifiedEpoch: 1, finalizedEpoch: 1},
			{bestDescendant: nonExistentNode, justifiedEpoch: 1, finalizedEpoch: 1},
			{bestDescendant: nonExistentNode, justifiedEpoch: 1, finalizedEpoch: 1}}}

	if err := s.updateBestChildAndDescendant(context.Background(), 0, 2); err != nil {
		t.Fatal(err)
	}

	// Verify parent's best child and best descendant are set to child index.
	if s.nodes[0].bestChild != 2 {
		t.Error("Did not get correct best child index")
	}
	if s.nodes[0].bestDescendant != 2 {
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
			{bestDescendant: nonExistentNode, justifiedEpoch: 1, finalizedEpoch: 1},
			{bestDescendant: nonExistentNode}}}

	if err := s.updateBestChildAndDescendant(context.Background(), 0, 2); err != nil {
		t.Fatal(err)
	}

	// Verify parent's best child and best descendant are not changed.
	if s.nodes[0].bestChild != 1 {
		t.Error("Did not get correct best child index")
	}
	if s.nodes[0].bestDescendant != 0 {
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
			{bestDescendant: nonExistentNode, justifiedEpoch: 1, finalizedEpoch: 1, weight: 1},
			{bestDescendant: nonExistentNode, justifiedEpoch: 1, finalizedEpoch: 1}}}

	if err := s.updateBestChildAndDescendant(context.Background(), 0, 2); err != nil {
		t.Fatal(err)
	}

	// Verify parent's best child and best descendant are not changed.
	if s.nodes[0].bestChild != 1 {
		t.Error("Did not get correct best child index")
	}
	if s.nodes[0].bestDescendant != 0 {
		t.Error("Did not get correct best descendant index")
	}
}

func TestStore_UpdateBestChildAndDescendant_NoChangeAtLeaf(t *testing.T) {
	// Make parent's best child to none and input child does not lead to viable index.
	s := &Store{
		justifiedEpoch: 1,
		finalizedEpoch: 1,
		nodes: []*Node{{bestChild: nonExistentNode, justifiedEpoch: 1, finalizedEpoch: 1},
			{bestDescendant: nonExistentNode, justifiedEpoch: 1, finalizedEpoch: 1},
			{bestDescendant: nonExistentNode}}}

	if err := s.updateBestChildAndDescendant(context.Background(), 0, 2); err != nil {
		t.Fatal(err)
	}

	// Verify parent's best child and best descendant are not changed.
	if s.nodes[0].bestChild != nonExistentNode {
		t.Error("Did not get correct best child index")
	}
	if s.nodes[0].bestDescendant != 0 {
		t.Error("Did not get correct best descendant index")
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
		got, err := s.leadsToViableHead(context.Background(), tc.n)
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
		if got := s.viableForHead(context.Background(), tc.n); got != tc.want {
			t.Errorf("viableForHead() = %v, want %v", got, tc.want)
		}
	}
}
