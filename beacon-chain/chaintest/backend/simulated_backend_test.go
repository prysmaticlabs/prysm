package backend

import "testing"

func TestRunChainTest(t *testing.T) {
	sb, err := NewSimulatedBackend()
	if err != nil {
		t.Fatal(err)
	}
	testCase := &ChainTestCase{
		Config: &ChainTestConfig{
			ShardCount:       3,
			CycleLength:      10,
			MinCommitteeSize: 3,
			ValidatorCount:   100,
		},
	}
	if err := sb.RunChainTest(testCase); err != nil {
		t.Errorf("Could not run chaintest: %v", err)
	}
}

func TestRunShuffleTest(t *testing.T) {
	sb, err := NewSimulatedBackend()
	if err != nil {
		t.Fatal(err)
	}
	testCase := &ShuffleTestCase{
		Input:  []uint32{1, 2, 3, 4, 5},
		Output: []uint32{2, 5, 3, 1, 4},
		Seed:   "abcde",
	}
	if err := sb.RunShuffleTest(testCase); err != nil {
		t.Errorf("Could not run chaintest: %v", err)
	}
}
