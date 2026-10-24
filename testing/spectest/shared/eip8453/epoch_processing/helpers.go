package epoch_processing

import (
	"os"
	"path"
	"strings"
	"testing"

	"github.com/OffchainLabs/prysm/v7/beacon-chain/core/helpers"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/state"
	state_native "github.com/OffchainLabs/prysm/v7/beacon-chain/state/state-native"
	"github.com/OffchainLabs/prysm/v7/build/bazel"
	ethpb "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
	"github.com/OffchainLabs/prysm/v7/testing/require"
	"github.com/OffchainLabs/prysm/v7/testing/spectest/utils"
	"github.com/OffchainLabs/prysm/v7/testing/util"
	"github.com/golang/snappy"
)

// runStateTest runs a state-only handler of the epoch_processing or round_processing runner.
func runStateTest(t *testing.T, config, runner, handler string, op func(state.BeaconState) error) {
	require.NoError(t, utils.SetConfig(t, config))
	testFolders, testsFolderPath := utils.TestFolders(t, config, "eip8453", path.Join(runner, handler, "pyspec_tests"))
	for _, folder := range testFolders {
		t.Run(folder.Name(), func(t *testing.T) {
			helpers.ClearCache()
			folderPath := path.Join(testsFolderPath, folder.Name())
			preFile, err := util.BazelFileBytes(folderPath, "pre.ssz_snappy")
			require.NoError(t, err)
			preSSZ, err := snappy.Decode(nil /* dst */, preFile)
			require.NoError(t, err, "Failed to decompress")
			pre := &ethpb.BeaconStateEIP8453{}
			require.NoError(t, pre.UnmarshalSSZ(preSSZ), "Failed to unmarshal")
			st, err := state_native.InitializeFromProtoUnsafeEIP8453(pre)
			require.NoError(t, err)

			postPath, err := bazel.Runfile(path.Join(folderPath, "post.ssz_snappy"))
			postExists := true
			if err != nil && strings.Contains(err.Error(), "could not locate file") {
				postExists = false
			} else if err != nil {
				t.Fatal(err)
			}

			err = op(st)
			if !postExists {
				if err == nil {
					t.Fatal("Did not fail when expected")
				}
				t.Logf("Expected failure; failure reason = %v", err)
				return
			}
			require.NoError(t, err)
			postFile, err := os.ReadFile(postPath) // #nosec G304
			require.NoError(t, err)
			postSSZ, err := snappy.Decode(nil /* dst */, postFile)
			require.NoError(t, err, "Failed to decompress")
			post := &ethpb.BeaconStateEIP8453{}
			require.NoError(t, post.UnmarshalSSZ(postSSZ), "Failed to unmarshal")
			pbState, err := state_native.ProtobufBeaconStateEIP8453(st.ToProtoUnsafe())
			require.NoError(t, err)
			require.DeepSSZEqual(t, pbState, post, "Post state does not match expected")
		})
	}
}
