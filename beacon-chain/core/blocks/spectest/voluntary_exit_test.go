package spectest

import (
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/ghodss/yaml"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params/spectest"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func runVoluntaryExitTest(t *testing.T, filename string) {
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
			pre := &pb.BeaconState{}
			if err := testutil.ConvertToPb(tt.Pre, pre); err != nil {
				t.Fatal(err)
			}

			expectedPost := &pb.BeaconState{}
			if err = testutil.ConvertToPb(tt.Post, expectedPost); err != nil {
				t.Fatal(err)
			}

			exit := &pb.VoluntaryExit{}
			if err = testutil.ConvertToPb(tt.VoluntaryExit, exit); err != nil {
				t.Fatal(err)
			}

			block := &pb.BeaconBlock{Body: &pb.BeaconBlockBody{VoluntaryExits: []*pb.VoluntaryExit{exit}}}

			var postState *pb.BeaconState
			postState, err = blocks.ProcessVoluntaryExits(pre, block, true)
			// Note: This doesn't test anything worthwhile. It essentially tests
			// that *any* error has occurred, not any specific error.
			if len(expectedPost.Validators) == 0 {
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

var exitPrefix = "eth2_spec_tests/tests/operations/voluntary_exit/"

func TestVoluntaryExitMinimal(t *testing.T) {
	filepath, err := bazel.Runfile(exitPrefix + "voluntary_exit_mainnet.yaml")
	if err != nil {
		t.Fatal(err)
	}
	runVoluntaryExitTest(t, filepath)
}

func TestVoluntaryExitMainnet(t *testing.T) {
	filepath, err := bazel.Runfile(exitPrefix + "voluntary_exit_mainnet.yaml")
	if err != nil {
		t.Fatal(err)
	}
	runVoluntaryExitTest(t, filepath)
}
