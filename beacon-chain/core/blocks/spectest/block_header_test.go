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
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"gopkg.in/d4l3k/messagediff.v1"
)

func runBlockHeaderTest(t *testing.T, config string) {
	if err := spectest.SetConfig(config); err != nil {
		t.Fatal(err)
	}

	testFolders, testsFolderPath := testutil.TestFolders(t, config, "operations/block_header/pyspec_tests")
	for _, folder := range testFolders {
		t.Run(folder.Name(), func(t *testing.T) {
			helpers.ClearAllCaches()
			blockFile, err := testutil.BazelFileBytes(testsFolderPath, folder.Name(), "block.ssz")
			if err != nil {
				t.Fatal(err)
			}
			block := &ethpb.BeaconBlock{}
			if err := ssz.Unmarshal(blockFile, block); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			preBeaconStateFile, err := testutil.BazelFileBytes(testsFolderPath, folder.Name(), "pre.ssz")
			if err != nil {
				t.Fatal(err)
			}
			preBeaconState := &pb.BeaconState{}
			if err := ssz.Unmarshal(preBeaconStateFile, preBeaconState); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			// If the post.ssz is not present, it means the test should fail on our end.
			postSSZFilepath, err := bazel.Runfile(path.Join(testsFolderPath, folder.Name(), "post.ssz"))
			postSSZExists := true
			if err != nil && strings.Contains(err.Error(), "could not locate file") {
				postSSZExists = false
			} else if err != nil {
				t.Fatal(err)
			}

			beaconState, err := blocks.ProcessBlockHeader(preBeaconState, block)
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
