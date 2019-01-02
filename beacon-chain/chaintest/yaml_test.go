package main

import (
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/chaintest/backend"
)

// TestReadTestsFromYaml tests constructing test cases from yaml file.
func TestReadTestsFromYaml(t *testing.T) {
	tests, err := readTestsFromYaml("./tests")
	if err != nil {
		t.Fatalf("Failed to read yaml files: %v", err)
	}

	sb, err := backend.NewSimulatedBackend()
	if err != nil {
		t.Fatalf("Failed to setup simulated backend: %v", err)
	}

	if err := runTests(tests, sb); err != nil {
		t.Fatal(err)
	}

}

// TestReadTestsFromYaml tests the running of provided tests structs.
func TestRunTests(t *testing.T) {
	sb, err := backend.NewSimulatedBackend()
	if err != nil {
		t.Fatalf("Could not create backend: %v", err)
	}

	chainTestCase := &backend.ChainTestCase{
		Config: &backend.ChainTestConfig{
			ShardCount:       3,
			CycleLength:      10,
			MinCommitteeSize: 3,
			ValidatorCount:   100,
		},
	}
	shuffleTestCase := &backend.ShuffleTestCase{}

	chainTest := &backend.ChainTest{TestCases: []*backend.ChainTestCase{chainTestCase}}
	shuffleTest := &backend.ShuffleTest{TestCases: []*backend.ShuffleTestCase{shuffleTestCase}}
	if err = runTests([]interface{}{chainTest, shuffleTest}, sb); err != nil {
		t.Fatalf("Failed to run test cases: %v", err)
	}
}
