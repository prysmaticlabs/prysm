package spectest

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/ghodss/yaml"
	"github.com/golang/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/params/spectest"
	"gopkg.in/d4l3k/messagediff.v1"
)

func TestBlockProcessingMinimalYaml(t *testing.T) {
	runBlockProcessingTest(t, "sanity_blocks_minimal.yaml")
}

func TestBlockProcessingMainnetYaml(t *testing.T) {
	runBlockProcessingTest(t, "sanity_blocks_mainnet.yaml")
}

func runBlockProcessingTest(t *testing.T, filename string) {
	filepath, err := bazel.Runfile("/eth2_spec_tests/tests/sanity/blocks/" + filename)
	if err != nil {
		t.Fatal(err)
	}
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
			helpers.ClearAllCaches()
			blocks.ClearEth1DataVoteCache()
			if tt.Description != "attester_slashing" {
				return
			}

			stateConfig := state.DefaultConfig()
			s := tt.Pre // Pre-state
			for _, b := range tt.Blocks {
				if tt.Pre, err = state.ExecuteStateTransition(ctx, tt.Pre, b, stateConfig); err != nil {
					t.Fatalf("Transition failed with block at slot %d: %v", b.Slot, err)
				}
			}

			if !proto.Equal(s, tt.Post) {
				diff, _ := messagediff.PrettyDiff(s, tt.Post)
				t.Log(diff)
				t.Fatal("Post state does not match expected")
			}
		})
	}
}
