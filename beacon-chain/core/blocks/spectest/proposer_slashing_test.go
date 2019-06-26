package spectest

import (
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/ghodss/yaml"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params/spectest"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func runProposerSlashingTest(t *testing.T, filename string) {
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Fatalf("Could not load file %v", err)
	}

	test := &BlockOperationTest{}
	if err := yaml.Unmarshal(file, test); err != nil {
		t.Fatalf("Failed to Unmarshal: %v", err)
	}

	if err := spectest.SetConfig(test.Config); err != nil {
		t.Fatal(err)
	}

	for _, tt := range test.TestCases {
		t.Run(tt.Description, func(t *testing.T) {
			helpers.ClearAllCaches()
			pre := &pb.BeaconState{}
			if err := testutil.ConvertToPb(tt.Pre, pre); err != nil {
				t.Fatal(err)
			}

			expectedPost := &pb.BeaconState{}
			if err = testutil.ConvertToPb(tt.Post, expectedPost); err != nil {
				t.Fatal(err)
			}

			proposerSlashing := &pb.ProposerSlashing{}
			if err = testutil.ConvertToPb(tt.ProposerSlashing, proposerSlashing); err != nil {
				t.Fatal(err)
			}

			block := &pb.BeaconBlock{Body: &pb.BeaconBlockBody{ProposerSlashings: []*pb.ProposerSlashing{proposerSlashing}}}

			var postState *pb.BeaconState
			postState, err = blocks.ProcessProposerSlashings(pre, block, true)
			// Note: This doesn't test anything worthwhile. It essentially tests
			// that *any* error has occurred, not any specific error.
			if len(expectedPost.ValidatorRegistry) == 0 {
				if err == nil {
					t.Fatal("Did not fail when expected")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(postState, expectedPost) {
				t.Error("Post state does not match expected")
			}
		})
	}
}

var proposerSlashingPrefix = "eth2_spec_tests/tests/operations/proposer_slashing/"

func TestProposerSlashingMinimal(t *testing.T) {
	filepath, err := bazel.Runfile(proposerSlashingPrefix + "proposer_slashing_minimal.yaml")
	if err != nil {
		t.Fatal(err)
	}
	runProposerSlashingTest(t, filepath)
}

func TestProposerSlashingMainnet(t *testing.T) {
	filepath, err := bazel.Runfile(proposerSlashingPrefix + "proposer_slashing_mainnet.yaml")
	if err != nil {
		t.Fatal(err)
	}
	runProposerSlashingTest(t, filepath)
}
