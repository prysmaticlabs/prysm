package main

import (
	"io/ioutil"
	"testing"

	"github.com/go-yaml/yaml"
	"github.com/prysmaticlabs/prysm/beacon-chain/chaintest/backend"
)

func TestFromYaml(t *testing.T) {
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
	file, err := ioutil.ReadFile("./tests/state-tests/no-blocks.yaml")
	if err != nil {
		b.Fatal(err)
	}

	test := &backend.StateTest{}
	if err := yaml.Unmarshal(file, test); err != nil {
		b.Fatal(err)
	}

	sb, err := backend.NewSimulatedBackend()
	if err != nil {
		b.Fatalf("Failed to setup simulated backend: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, testCase := range test.TestCases {
			if err := sb.RunStateTransitionTest(testCase); err != nil {
				b.Error(err)
			}
		}
	}

}
