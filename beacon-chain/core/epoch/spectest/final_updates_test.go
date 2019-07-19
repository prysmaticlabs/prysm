package spectest

import (
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/ghodss/yaml"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/epoch"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params/spectest"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"gopkg.in/d4l3k/messagediff.v1"
)

func runFinalUpdatesTests(t *testing.T, filename string) {
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Fatalf("Could not load file %v", err)
	}

	s := &EpochProcessingTest{}
	if err := yaml.Unmarshal(file, s); err != nil {
		t.Fatalf("Failed to Unmarshal: %v", err)
	}

	if err := spectest.SetConfig(s.Config); err != nil {
		t.Fatal(err)
	}

	for _, tt := range s.TestCases[0:1] {
		t.Run(tt.Description, func(t *testing.T) {
			helpers.ClearAllCaches()
			preState := &pb.BeaconState{}
			if err := testutil.ConvertToPb(tt.Pre, preState); err != nil {
				t.Fatal(err)
			}

			var postState *pb.BeaconState
			postState, err = epoch.ProcessFinalUpdates(preState)
			if err != nil {
				t.Fatal(err)
			}

			expectedPostState := &pb.BeaconState{}
			if err := testutil.ConvertToPb(tt.Post, expectedPostState); err != nil {
				t.Fatal(err)
			}

			if expectedPostState.CurrentEpochAttestations == nil {
				expectedPostState.CurrentEpochAttestations = []*pb.PendingAttestation{}
			}

			if !reflect.DeepEqual(postState, expectedPostState) {
				t.Error("Did not get expected state")
				diff, _ := messagediff.PrettyDiff(expectedPostState, postState)
				t.Log(diff)
			}
		})
	}
}

const finalUpdatesPrefix = "tests/epoch_processing/final_updates/"

func TestFinalUpdatesMinimal(t *testing.T) {
	filepath, err := bazel.Runfile(finalUpdatesPrefix + "final_updates_minimal.yaml")
	if err != nil {
		t.Fatal(err)
	}
	runFinalUpdatesTests(t, filepath)
}

func TestFinalUpdatesMainnet(t *testing.T) {
	filepath, err := bazel.Runfile(finalUpdatesPrefix + "final_updates_mainnet.yaml")
	if err != nil {
		t.Fatal(err)
	}
	runFinalUpdatesTests(t, filepath)
}
