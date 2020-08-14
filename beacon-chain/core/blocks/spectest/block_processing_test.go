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
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"gopkg.in/d4l3k/messagediff.v1"
)

func init() {
	state.SkipSlotCache.Disable()
}

func runBlockProcessingTest(t *testing.T, config string) {
	require.NoError(t, spectest.SetConfig(t, config))

	testFolders, testsFolderPath := testutil.TestFolders(t, config, "sanity/blocks/pyspec_tests")
	for _, folder := range testFolders {
		t.Run(folder.Name(), func(t *testing.T) {
			helpers.ClearCache()
			preBeaconStateFile, err := testutil.BazelFileBytes(testsFolderPath, folder.Name(), "pre.ssz")
			require.NoError(t, err)
			beaconStateBase := &pb.BeaconState{}
			require.NoError(t, beaconStateBase.UnmarshalSSZ(preBeaconStateFile), "Failed to unmarshal")
			beaconState, err := stateTrie.InitializeFromProto(beaconStateBase)
			require.NoError(t, err)

			file, err := testutil.BazelFileBytes(testsFolderPath, folder.Name(), "meta.yaml")
			require.NoError(t, err)

			metaYaml := &SanityConfig{}
			require.NoError(t, testutil.UnmarshalYaml(file, metaYaml), "Failed to Unmarshal")

			var transitionError error
			for i := 0; i < metaYaml.BlocksCount; i++ {
				filename := fmt.Sprintf("blocks_%d.ssz", i)
				blockFile, err := testutil.BazelFileBytes(testsFolderPath, folder.Name(), filename)
				require.NoError(t, err)
				block := &ethpb.SignedBeaconBlock{}
				require.NoError(t, block.UnmarshalSSZ(blockFile), "Failed to unmarshal")
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
				require.NoError(t, err)

				postBeaconState := &pb.BeaconState{}
				require.NoError(t, postBeaconState.UnmarshalSSZ(postBeaconStateFile), "Failed to unmarshal")

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
