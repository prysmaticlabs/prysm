package backend

import "testing"

func TestRunChainTests(t *testing.T) {
	if 1 != 1 {
		t.Errorf("Expected %v", 1)
	}
}
