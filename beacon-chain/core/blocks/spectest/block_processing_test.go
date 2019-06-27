package spectest

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/ghodss/yaml"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/shared/params/spectest"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/golang/protobuf/proto"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
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

			stateConfig := state.DefaultConfig()
			s := tt.Pre // Pre-state
			for _, b := range tt.Blocks {
				if s, err = state.ExecuteStateTransition(ctx, s, b, stateConfig); err != nil {
					t.Fatal(err)
				}
			}
			expectedPost := &pb.BeaconState{}
			if err := testutil.ConvertToPb(tt.Post, expectedPost); err != nil {
				t.Fatal(err)
			}
			if !proto.Equal(s, expectedPost) {
				diff, _ := messagediff.PrettyDiff(s, expectedPost)
				t.Log(diff)
				t.Fatal("Post state does not match expected")
			}
		})
	}
}
