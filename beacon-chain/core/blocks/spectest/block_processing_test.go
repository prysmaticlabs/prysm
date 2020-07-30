// Package spectest contains all comformity specification tests
// for block processing according to the eth2 spec.
package spectest

import (
	"context"
	"fmt"
	"io/ioutil"
	"path"
	"strings"
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params/spectest"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"gopkg.in/d4l3k/messagediff.v1"
)

func init() {
	state.SkipSlotCache.Disable()
}

func runBlockProcessingTest(t *testing.T, config string) {
	if err := spectest.SetConfig(t, config); err != nil {
		t.Fatal(err)
	}

	testFolders, testsFolderPath := testutil.TestFolders(t, config, "sanity/blocks/pyspec_tests")

	for _, folder := range testFolders {
		t.Run(folder.Name(), func(t *testing.T) {
			helpers.ClearCache()
			preBeaconStateFile, err := testutil.BazelFileBytes(testsFolderPath, folder.Name(), "pre.ssz")
			if err != nil {
				t.Fatal(err)
			}
			beaconStateBase := &pb.BeaconState{}
			if err := beaconStateBase.UnmarshalSSZ(preBeaconStateFile); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}
			beaconState, err := stateTrie.InitializeFromProto(beaconStateBase)
			if err != nil {
				t.Fatal(err)
			}

			file, err := testutil.BazelFileBytes(testsFolderPath, folder.Name(), "meta.yaml")
			if err != nil {
				t.Fatal(err)
			}

			metaYaml := &SanityConfig{}
			if err := testutil.UnmarshalYaml(file, metaYaml); err != nil {
				t.Fatalf("Failed to Unmarshal: %v", err)
			}

			var transitionError error
			for i := 0; i < metaYaml.BlocksCount; i++ {
				filename := fmt.Sprintf("blocks_%d.ssz", i)
				blockFile, err := testutil.BazelFileBytes(testsFolderPath, folder.Name(), filename)
				if err != nil {
					t.Fatal(err)
				}
				block := &ethpb.SignedBeaconBlock{}
				if err := block.UnmarshalSSZ(blockFile); err != nil {
					t.Fatalf("Failed to unmarshal: %v", err)
				}
				beaconState, transitionError = state.ExecuteStateTransition(context.Background(), beaconState, block)
				if transitionError != nil {
					break
				}
			}

			// If the post.ssz is not present, it means the test should fail on our end.
			postSSZFilepath, readError := bazel.Runfile(path.Join(testsFolderPath, folder.Name(), "post.ssz"))
			postSSZExists := true
			if readError != nil && strings.Contains(readError.Error(), "could not locate file") {
				postSSZExists = false
			} else if readError != nil {
				t.Fatal(readError)
			}

			if postSSZExists {
				if transitionError != nil {
					t.Errorf("Unexpected error: %v", transitionError)
				}

				postBeaconStateFile, err := ioutil.ReadFile(postSSZFilepath)
				if err != nil {
					t.Fatal(err)
				}

				postBeaconState := &pb.BeaconState{}
				if err := postBeaconState.UnmarshalSSZ(postBeaconStateFile); err != nil {
					t.Fatalf("Failed to unmarshal: %v", err)
				}

				if !proto.Equal(beaconState.InnerStateUnsafe(), postBeaconState) {
					diff, _ := messagediff.PrettyDiff(beaconState.InnerStateUnsafe(), postBeaconState)
					t.Log(diff)
					t.Fatal("Post state does not match expected")
				}
			} else {
				// Note: This doesn't test anything worthwhile. It essentially tests
				// that *any* error has occurred, not any specific error.
				if transitionError == nil {
					t.Fatal("Did not fail when expected")
				}
				t.Logf("Expected failure; failure reason = %v", transitionError)
				return
			}
		})
	}
}
