package trieutil

import "testing"

func TestMerkleTrie_BranchIndices(t *testing.T) {
	m := &MerkleTrie{depth: 3}
	indices := m.BranchIndices(1024)
	expected := []int{1024, 512, 256}
	for i := 0; i < len(indices); i++ {
		if expected[i] != indices[i] {
			t.Errorf("Expected %d, received %d", expected[i], indices[i])
		}
	}
}
