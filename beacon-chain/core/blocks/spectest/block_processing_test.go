package spectest

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/ghodss/yaml"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/shared/params/spectest"
)

func TestBlockProcessingMinimalYaml(t *testing.T) {
	runBlockProcessingTest(t, "sanity_blocks_minimal.yaml")
}

func TestBlockProcessingMainnetYaml(t *testing.T) {
	runBlockProcessingTest(t, "sanity_blocks_mainnet.yaml")
}

func runBlockProcessingTest(t *testing.T, filename string) {
	filepath, err := bazel.Runfile("/eth2_spec_tests/tests/sanity/blocks/" + filename)
	file, err := ioutil.ReadFile(filepath)
	if err != nil {
		t.Fatalf("Could not load file %v", err)
	}

	s := &BlocksMainnet{}
	if err := yaml.Unmarshal(file, s); err != nil {
		t.Fatalf("Failed to Unmarshal: %v", err)
	}

	if err := spectest.SetConfig(s.Config); err != nil {
		t.Fatalf("Could not set config: %v", err)
	}

	for _, tt := range s.TestCases {
		t.Run(tt.Description, func(t *testing.T) {
			ctx := context.Background()

			// TODO: Unskip these tests.
			if tt.Description == "attestation" ||  tt.Description == "voluntary_exit" {
				t.Skip("Not passing yet...")
			}

			stateConfig := state.DefaultConfig()
			for _, b := range tt.Blocks {
				if _, err = state.ExecuteStateTransition(ctx, tt.Pre, b, stateConfig); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}
