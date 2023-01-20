package fork

import (
	"path"
	"testing"

	"github.com/golang/snappy"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/capella"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/helpers"
	state_native "github.com/prysmaticlabs/prysm/v3/beacon-chain/state/state-native"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/testing/spectest/utils"
	"github.com/prysmaticlabs/prysm/v3/testing/util"
	"google.golang.org/protobuf/proto"
)

// RunUpgradeToDeneb4 is a helper function that runs Deneb4's fork spec tests.
// It unmarshals a pre- and post-state to check `UpgradeToDeneb4` comply with spec implementation.
func RunUpgradeToDeneb4(t *testing.T, config string) {
	require.NoError(t, utils.SetConfig(t, config))

	testFolders, testsFolderPath := utils.TestFolders(t, config, "deneb", "fork/fork/pyspec_tests")
	for _, folder := range testFolders {
		t.Run(folder.Name(), func(t *testing.T) {
			if folder.Name() != "fork_next_epoch_with_block" {
				t.Skip("Skipping non-upgrade_to_Deneb4 test")
			}
			helpers.ClearCache()
			folderPath := path.Join(testsFolderPath, folder.Name())

			preStateFile, err := util.BazelFileBytes(path.Join(folderPath, "pre.ssz_snappy"))
			require.NoError(t, err)
			preStateSSZ, err := snappy.Decode(nil /* dst */, preStateFile)
			require.NoError(t, err, "Failed to decompress")
			preStateBase := &ethpb.BeaconStateCapella{}
			if err := preStateBase.UnmarshalSSZ(preStateSSZ); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}
			preState, err := state_native.InitializeFromProtoCapella(preStateBase)
			require.NoError(t, err)
			postState, err := capella.UpgradeToDeneb(preState)
			require.NoError(t, err)
			postStateFromFunction, err := state_native.ProtobufBeaconStateDeneb(postState.ToProtoUnsafe())
			require.NoError(t, err)

			postStateFile, err := util.BazelFileBytes(path.Join(folderPath, "post.ssz_snappy"))
			require.NoError(t, err)
			postStateSSZ, err := snappy.Decode(nil /* dst */, postStateFile)
			require.NoError(t, err, "Failed to decompress")
			postStateFromFile := &ethpb.BeaconStateDeneb{}
			if err := postStateFromFile.UnmarshalSSZ(postStateSSZ); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			if !proto.Equal(postStateFromFile, postStateFromFunction) {
				t.Log(postStateFromFile.LatestExecutionPayloadHeader)
				t.Log(postStateFromFunction.LatestExecutionPayloadHeader)
				t.Fatal("Post state does not match expected")
			}
		})
	}
}
