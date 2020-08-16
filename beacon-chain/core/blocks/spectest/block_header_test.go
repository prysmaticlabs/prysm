package spectest

import (
	"io/ioutil"
	"path"
	"strings"
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params/spectest"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"gopkg.in/d4l3k/messagediff.v1"
)

func runBlockHeaderTest(t *testing.T, config string) {
	require.NoError(t, spectest.SetConfig(t, config))

	testFolders, testsFolderPath := testutil.TestFolders(t, config, "operations/block_header/pyspec_tests")
	for _, folder := range testFolders {
		t.Run(folder.Name(), func(t *testing.T) {
			blockFile, err := testutil.BazelFileBytes(testsFolderPath, folder.Name(), "block.ssz")
			require.NoError(t, err)
			block := &ethpb.BeaconBlock{}
			require.NoError(t, block.UnmarshalSSZ(blockFile), "Failed to unmarshal")

			preBeaconStateFile, err := testutil.BazelFileBytes(testsFolderPath, folder.Name(), "pre.ssz")
			require.NoError(t, err)
			preBeaconStateBase := &pb.BeaconState{}
			require.NoError(t, preBeaconStateBase.UnmarshalSSZ(preBeaconStateFile), "Failed to unmarshal")
			preBeaconState, err := stateTrie.InitializeFromProto(preBeaconStateBase)
			require.NoError(t, err)

			// If the post.ssz is not present, it means the test should fail on our end.
			postSSZFilepath, err := bazel.Runfile(path.Join(testsFolderPath, folder.Name(), "post.ssz"))
			postSSZExists := true
			if err != nil && strings.Contains(err.Error(), "could not locate file") {
				postSSZExists = false
			} else if err != nil {
				t.Fatal(err)
			}

			// Spectest blocks are not signed, so we'll call NoVerify to skip sig verification.
			beaconState, err := blocks.ProcessBlockHeaderNoVerify(preBeaconState, block)
			if postSSZExists {
				require.NoError(t, err)

				postBeaconStateFile, err := ioutil.ReadFile(postSSZFilepath)
				require.NoError(t, err)

				postBeaconState := &pb.BeaconState{}
				require.NoError(t, postBeaconState.UnmarshalSSZ(postBeaconStateFile), "Failed to unmarshal")
				if !proto.Equal(beaconState.CloneInnerState(), postBeaconState) {
					diff, _ := messagediff.PrettyDiff(beaconState.CloneInnerState(), postBeaconState)
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
