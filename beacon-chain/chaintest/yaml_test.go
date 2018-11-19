package main

import (
	"log"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/chaintest/backend"
)

// TestReadTestsFromYaml tests constructing test cases from yaml file
func TestReadTestsFromYaml(t *testing.T) {
	if _, err := readTestsFromYaml(string("./tests")); err != nil {
		t.Fatalf("Failed to read yaml files")
	}
}

// TestReadTestsFromYaml tests the running of provided tests structs
func TestRunTests(t *testing.T) {
	sb, err := backend.NewSimulatedBackend()
	if err != nil {
		log.Fatalf("Could not create backend: %v", err)
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
		log.Fatalf("Fail to run test cases: %v", err)
	}
}
