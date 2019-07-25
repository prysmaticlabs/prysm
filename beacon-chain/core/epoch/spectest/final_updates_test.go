package spectest

import (
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/epoch"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params/spectest"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"gopkg.in/d4l3k/messagediff.v1"
)

const finalUpdatesPrefix = "tests/epoch_processing/final_updates/"

func runFinalUpdatesTests(t *testing.T, filename string) {
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Fatalf("Could not load file %v", err)
	}

	s := &EpochProcessingTest{}
	if err := testutil.UnmarshalYaml(file, s); err != nil {
		t.Fatalf("Failed to Unmarshal: %v", err)
	}

	if err := spectest.SetConfig(s.Config); err != nil {
		t.Fatal(err)
	}

	if len(s.TestCases) == 0 {
		t.Fatal("No tests!")
	}

	for _, tt := range s.TestCases {
		t.Run(tt.Description, func(t *testing.T) {
			helpers.ClearAllCaches()

			var postState *pb.BeaconState
			postState, err = epoch.ProcessFinalUpdates(tt.Pre)
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(postState, tt.Post) {
				t.Error("Did not get expected state")
				diff, _ := messagediff.PrettyDiff(tt.Post, postState)
				t.Log(diff)
			}
		})
	}
}
