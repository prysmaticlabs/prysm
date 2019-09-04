package spectest

import (
	"io/ioutil"
	"path"
	"strings"
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state/stateutils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params/spectest"
	"gopkg.in/d4l3k/messagediff.v1"
)

func runDepositTest(t *testing.T, config string) {
	testsFolderPath := path.Join("tests", config, "phase0/operations/deposit/pyspec_tests")
	filepath, err := bazel.Runfile(testsFolderPath)
	if err != nil {
		t.Fatal(err)
	}
	testFolders, err := ioutil.ReadDir(filepath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if err := spectest.SetConfig(config); err != nil {
		t.Fatal(err)
	}

	for _, folder := range testFolders {
		depositFile, err := SSZFileBytes(testsFolderPath, folder.Name(), "deposit.ssz")
		if err != nil {
			t.Fatal(err)
		}
		deposit := &ethpb.Deposit{}
		if err := ssz.Unmarshal(depositFile, deposit); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		preBeaconStateFile, err := SSZFileBytes(testsFolderPath, folder.Name(), "pre.ssz")
		if err != nil {
			t.Fatal(err)
		}
		beaconState := &pb.BeaconState{}
		if err := ssz.Unmarshal(preBeaconStateFile, beaconState); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		t.Run(folder.Name(), func(t *testing.T) {
			helpers.ClearAllCaches()

			// If the post.ssz is not present, it means the test should fail on our end.
			postSSZFilepath, err := bazel.Runfile(path.Join(testsFolderPath, folder.Name(), "post.ssz"))
			postSSZExists := true
			if err != nil && strings.Contains(err.Error(), "could not locate file") {
				postSSZExists = false
			} else if err != nil {
				t.Fatal(err)
			}

			valMap := stateutils.ValidatorIndexMap(beaconState)
			beaconState, err := blocks.ProcessDeposit(beaconState, deposit, valMap)
			if postSSZExists {
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}

				postBeaconStateFile, err := ioutil.ReadFile(postSSZFilepath)
				if err != nil {
					t.Fatal(err)
				}

				postBeaconState := &pb.BeaconState{}
				if err := ssz.Unmarshal(postBeaconStateFile, postBeaconState); err != nil {
					t.Fatalf("Failed to unmarshal: %v", err)
				}

				if !proto.Equal(beaconState, postBeaconState) {
					diff, _ := messagediff.PrettyDiff(beaconState, postBeaconState)
					t.Log(diff)
					t.Fatal("Post state does not match expected")
				}
			} else {
				// Note: This doesn't test anything worthwhile. It essentially tests
				// that *any* error has occurred, not any specific error.
				if err == nil {
					t.Fatal("Did not fail when expected")
				}
				t.Logf("Expected failure; failure reason = %v", err)
				return
			}
		})
	}
}
