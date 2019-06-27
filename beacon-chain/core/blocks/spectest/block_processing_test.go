package spectest

import (
	"context"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/ghodss/yaml"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/shared/params/spectest"
)

func TestBlockProcessingMinimalYaml(t *testing.T) {
	ctx := context.Background()
	filepath, err := bazel.Runfile("/eth2_spec_tests/tests/sanity/blocks/sanity_blocks_minimal.yaml")
	if err != nil {
		t.Fatal(err)
	}

	file, err := ioutil.ReadFile(filepath)
	if err != nil {
		t.Fatalf("Could not load file %v", err)
	}

	s := &BlocksMinimal{}
	if err := yaml.Unmarshal(file, s); err != nil {
		t.Fatalf("Failed to Unmarshal: %v", err)
	}

	t.Logf("Title: %v", s.Title)
	t.Logf("Summary: %v", s.Summary)
	t.Logf("Fork: %v", s.Forks)
	t.Logf("Config: %v", s.Config)

	if err := spectest.SetConfig(s.Config); err != nil {
		t.Fatalf("Could not set config: %v", err)
	}

	for _, testCase := range s.TestCases {
		helpers.ClearAllCaches()
		// if testCase.Description == "voluntary_exit" {
		// 	continue
		// }
		fmt.Printf("Description: %s\n", testCase.Description)
		stateConfig := state.DefaultConfig()

		for _, b := range testCase.Blocks {
			if _, err = state.ExecuteStateTransition(ctx, testCase.Pre, b, stateConfig); err != nil {
				t.Fatal(err)
			}
		}
	}
}

// func TestBlockProcessingMainnetYaml(t *testing.T) {
// 	ctx := context.Background()
// 	filepath, err := bazel.Runfile("/eth2_spec_tests/tests/sanity/blocks/sanity_blocks_mainnet.yaml")
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	file, err := ioutil.ReadFile(filepath)
// 	if err != nil {
// 		t.Fatalf("Could not load file %v", err)
// 	}

// 	s := &BlocksMainnet{}
// 	if err := yaml.Unmarshal(file, s); err != nil {
// 		t.Fatalf("Failed to Unmarshal: %v", err)
// 	}

// 	t.Logf("Title: %v", s.Title)
// 	t.Logf("Summary: %v", s.Summary)
// 	t.Logf("Fork: %v", s.Forks)
// 	t.Logf("Config: %v", s.Config)

// 	if err := spectest.SetConfig(s.Config); err != nil {
// 		t.Fatalf("Could not set config: %v", err)
// 	}

// 	for _, testCase := range s.TestCases {
// 		t.Logf("Description: %s", testCase.Description)
// 		if testCase.Description == "attestation" || testCase.Description == "voluntary_exit" {
// 			continue
// 		}

// 		stateConfig := state.DefaultConfig()
// 		for _, b := range testCase.Blocks {
// 			if _, err = state.ExecuteStateTransition(ctx, testCase.Pre, b, stateConfig); err != nil {
// 				t.Fatal(err)
// 			}
// 		}
// 	}
// }
