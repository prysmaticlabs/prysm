package main

import (
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/chaintest/backend"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
)

func init() {
	featureconfig.InitFeatureConfig(&featureconfig.FeatureFlagConfig{
		EnableCrosslinks: true,
	})
}

func TestFromYaml_Pass(t *testing.T) {
	tests, err := readTestsFromYaml("./tests")
	if err != nil {
		t.Fatalf("Failed to read yaml files: %v", err)
	}

	sb, err := backend.NewSimulatedBackend()
	if err != nil {
		t.Fatalf("Could not create backend: %v", err)
	}

	if err := runTests(tests, sb); err != nil {
		t.Errorf("Failed to run yaml tests %v", err)
	}
}

func BenchmarkStateTestFromYaml(b *testing.B) {
	tests, err := readTestsFromYaml("./tests")
	if err != nil {
		b.Fatalf("Failed to read yaml files: %v", err)
	}

	sb, err := backend.NewSimulatedBackend()
	if err != nil {
		b.Fatalf("Could not create backend: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := runTests(tests, sb); err != nil {
			b.Errorf("Failed to run yaml tests %v", err)
		}
	}
}
