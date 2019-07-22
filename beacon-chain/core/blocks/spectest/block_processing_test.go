package spectest

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/shared/params/spectest"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"gopkg.in/d4l3k/messagediff.v1"
)

func TestBlockProcessingMinimalYaml(t *testing.T) {
	t.Skip("Test will fail with mainnet protos")

	runBlockProcessingTest(t, "sanity_blocks_minimal.yaml")
}

func TestBlockProcessingMainnetYaml(t *testing.T) {
	runBlockProcessingTest(t, "sanity_blocks_mainnet.yaml")
}

func runBlockProcessingTest(t *testing.T, filename string) {
	filepath, err := bazel.Runfile("tests/sanity/blocks/" + filename)
	if err != nil {
		t.Fatal(err)
	}
	file, err := ioutil.ReadFile(filepath)
	if err != nil {
		t.Fatalf("Could not load file %v", err)
	}

	s := &BlocksMainnet{}
	if err := testutil.UnmarshalYaml(file, s); err != nil {
		t.Fatalf("Failed to Unmarshal: %v", err)
	}

	if err := spectest.SetConfig(s.Config); err != nil {
		t.Fatalf("Could not set config: %v", err)
	}

	if len(s.TestCases) == 0 {
		t.Fatal("No tests!")
	}

	for _, tt := range s.TestCases {
		t.Run(tt.Description, func(t *testing.T) {
			ctx := context.Background()
			helpers.ClearAllCaches()
			blocks.ClearEth1DataVoteCache()

			stateConfig := &state.TransitionConfig{
				VerifySignatures: true,
				VerifyStateRoot:  true,
			}

			s := tt.Pre
			for _, b := range tt.Blocks {
				tt.Pre, err = state.ExecuteStateTransition(ctx, tt.Pre, b, stateConfig)
				if tt.Post == nil {
					if err == nil {
						t.Fatal("Transition did not fail despite being invalid")
					}
					continue
				}
				if err != nil {
					t.Fatalf("Transition failed with block at slot %d: %v", b.Slot, err)
				}
			}
			if tt.Post != nil {
				if !proto.Equal(s, tt.Post) {
					diff, _ := messagediff.PrettyDiff(s, tt.Post)
					t.Log(diff)
					t.Fatal("Post state does not match expected")
				}
			}
		})
	}
}
