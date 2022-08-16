package fork

import (
	"context"
	"path"
	"testing"

	"github.com/golang/snappy"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/altair"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/helpers"
	v1 "github.com/prysmaticlabs/prysm/v3/beacon-chain/state/v1"
	statealtair "github.com/prysmaticlabs/prysm/v3/beacon-chain/state/v2"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/testing/spectest/utils"
	"github.com/prysmaticlabs/prysm/v3/testing/util"
	"google.golang.org/protobuf/proto"
)

// RunUpgradeToAltair is a helper function that runs Altair's fork spec tests.
// It unmarshals a pre and post state to check `UpgradeToAltair` comply with spec implementation.
func RunUpgradeToAltair(t *testing.T, config string) {
	require.NoError(t, utils.SetConfig(t, config))

	testFolders, testsFolderPath := utils.TestFolders(t, config, "altair", "fork/fork/pyspec_tests")
	for _, folder := range testFolders {
		t.Run(folder.Name(), func(t *testing.T) {
			helpers.ClearCache()
			folderPath := path.Join(testsFolderPath, folder.Name())

			preStateFile, err := util.BazelFileBytes(path.Join(folderPath, "pre.ssz_snappy"))
			require.NoError(t, err)
			preStateSSZ, err := snappy.Decode(nil /* dst */, preStateFile)
			require.NoError(t, err, "Failed to decompress")
			preStateBase := &ethpb.BeaconState{}
			if err := preStateBase.UnmarshalSSZ(preStateSSZ); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}
			preState, err := v1.InitializeFromProto(preStateBase)
			require.NoError(t, err)
			postState, err := altair.UpgradeToAltair(context.Background(), preState)
			require.NoError(t, err)
			postStateFromFunction, err := statealtair.ProtobufBeaconState(postState.InnerStateUnsafe())
			require.NoError(t, err)

			postStateFile, err := util.BazelFileBytes(path.Join(folderPath, "post.ssz_snappy"))
			require.NoError(t, err)
			postStateSSZ, err := snappy.Decode(nil /* dst */, postStateFile)
			require.NoError(t, err, "Failed to decompress")
			postStateFromFile := &ethpb.BeaconStateAltair{}
			if err := postStateFromFile.UnmarshalSSZ(postStateSSZ); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			if !proto.Equal(postStateFromFile, postStateFromFunction) {
				t.Fatal("Post state does not match expected")
			}
		})
	}
}
