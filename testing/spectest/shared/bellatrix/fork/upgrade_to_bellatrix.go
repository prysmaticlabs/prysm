package fork

import (
	"path"
	"testing"

	"github.com/golang/snappy"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/execution"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/helpers"
	v2 "github.com/prysmaticlabs/prysm/v3/beacon-chain/state/v2"
	v3 "github.com/prysmaticlabs/prysm/v3/beacon-chain/state/v3"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/testing/spectest/utils"
	"github.com/prysmaticlabs/prysm/v3/testing/util"
	"google.golang.org/protobuf/proto"
)

// RunUpgradeToBellatrix is a helper function that runs bellatrix's fork spec tests.
// It unmarshals a pre- and post-state to check `UpgradeToBellatrix` comply with spec implementation.
func RunUpgradeToBellatrix(t *testing.T, config string) {
	require.NoError(t, utils.SetConfig(t, config))

	testFolders, testsFolderPath := utils.TestFolders(t, config, "bellatrix", "fork/fork/pyspec_tests")
	for _, folder := range testFolders {
		t.Run(folder.Name(), func(t *testing.T) {
			helpers.ClearCache()
			folderPath := path.Join(testsFolderPath, folder.Name())

			preStateFile, err := util.BazelFileBytes(path.Join(folderPath, "pre.ssz_snappy"))
			require.NoError(t, err)
			preStateSSZ, err := snappy.Decode(nil /* dst */, preStateFile)
			require.NoError(t, err, "Failed to decompress")
			preStateBase := &ethpb.BeaconStateAltair{}
			if err := preStateBase.UnmarshalSSZ(preStateSSZ); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}
			preState, err := v2.InitializeFromProto(preStateBase)
			require.NoError(t, err)
			postState, err := execution.UpgradeToBellatrix(preState)
			require.NoError(t, err)
			postStateFromFunction, err := v3.ProtobufBeaconState(postState.InnerStateUnsafe())
			require.NoError(t, err)

			postStateFile, err := util.BazelFileBytes(path.Join(folderPath, "post.ssz_snappy"))
			require.NoError(t, err)
			postStateSSZ, err := snappy.Decode(nil /* dst */, postStateFile)
			require.NoError(t, err, "Failed to decompress")
			postStateFromFile := &ethpb.BeaconStateBellatrix{}
			if err := postStateFromFile.UnmarshalSSZ(postStateSSZ); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			if !proto.Equal(postStateFromFile, postStateFromFunction) {
				t.Fatal("Post state does not match expected")
			}
		})
	}
}
