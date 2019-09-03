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
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params/spectest"
)

func runAttesterSlashingTest(t *testing.T, config string) {
	testsFolderPath := path.Join("tests", config, "phase0/operations/attester_slashing/pyspec_tests")
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
		attSlashingFilepath, err := bazel.Runfile(path.Join(testsFolderPath, folder.Name(), "attester_slashing.ssz"))
		if err != nil {
			t.Fatal(err)
		}
		attSlashingFile, err := ioutil.ReadFile(attSlashingFilepath)
		if err != nil {
			t.Fatal(err)
		}
		attSlashing := &ethpb.AttesterSlashing{}
		if err := ssz.Unmarshal(attSlashingFile, attSlashing); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		preSSZFilepath, err := bazel.Runfile(path.Join(testsFolderPath, folder.Name(), "pre.ssz"))
		if err != nil {
			t.Fatal(err)
		}
		preBeaconStateFile, err := ioutil.ReadFile(preSSZFilepath)
		if err != nil {
			t.Fatal(err)
		}
		preBeaconState := &pb.BeaconState{}
		if err := ssz.Unmarshal(preBeaconStateFile, preBeaconState); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		t.Run(folder.Name(), func(t *testing.T) {
			helpers.ClearAllCaches()
			body := &ethpb.BeaconBlockBody{
				AttesterSlashings: []*ethpb.AttesterSlashing{attSlashing},
			}

			postState, err := blocks.ProcessAttesterSlashings(preBeaconState, body)

			// If the post.ssz is not present, it means the test should fail on our end.
			postSSZFilepath, err := bazel.Runfile(path.Join(testsFolderPath, folder.Name(), "post.ssz"))
			postSSZExists := true
			if err != nil && strings.Contains(err.Error(), "could not locate file") {
				postSSZExists = false
			} else if err != nil {
				t.Fatal(err)
			}

			if postSSZExists {
				postBeaconStateFile, err := ioutil.ReadFile(postSSZFilepath)
				if err != nil {
					t.Fatal(err)
				}

				postBeaconState := &pb.BeaconState{}
				if err := ssz.Unmarshal(postBeaconStateFile, postBeaconState); err != nil {
					t.Fatalf("Failed to unmarshal: %v", err)
				}

				if !proto.Equal(postState, postBeaconState) {
					// diff, _ := messagediff.PrettyDiff(postState, postBeaconState)
					// t.Log(diff)
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
