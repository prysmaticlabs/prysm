package main

import (
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/chaintest/backend"
)

// TestReadTestsFromYaml tests constructing test cases from yaml file.
func TestReadTestsFromYaml(t *testing.T) {
	tests, err := readTestsFromYaml(string("./tests"))
	if err != nil {
		t.Fatalf("Failed to read yaml files: %v", err)
	}
	sb, err := backend.NewSimulatedBackend()
	if err != nil {
		t.Fatalf("Could not create backend: %v", err)
	}
	if err := runTests(tests, sb); err != nil {
		t.Errorf("Failed to run yaml tests")
	}
}
