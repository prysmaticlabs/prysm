package fork

import (
	"fmt"
	"testing"

	"github.com/golang/snappy"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateV0"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/prysmaticlabs/prysm/spectest/utils"
)

type ForkConfig struct {
	PostFork    string `json:"post_fork"`
	ForkEpoch   int    `json:"fork_epoch"`
	ForkBlock   int    `json:"fork_block"`
	BlocksCount int    `json:"blocks_count"`
}

// RunForkTransitionTest is a helper function that runs Altair's transition core tests.
func RunForkTransitionTest(t *testing.T, config string) {
	require.NoError(t, utils.SetConfig(t, config))

	testFolders, testsFolderPath := utils.TestFolders(t, config, "altair", "transition/core/pyspec_tests")
	for _, folder := range testFolders {
		t.Run(folder.Name(), func(t *testing.T) {
			helpers.ClearCache()

			file, err := testutil.BazelFileBytes(testsFolderPath, folder.Name(), "meta.yaml")
			require.NoError(t, err)

			metaYaml := &ForkConfig{}
			require.NoError(t, utils.UnmarshalYaml(file, metaYaml), "Failed to Unmarshal")

			preforkBlocks := make([]*ethpb.SignedBeaconBlock, 0)
			postforkBlocks := make([]*ethpb.SignedBeaconBlockAltair, 0)
			for i := 0; i <= metaYaml.ForkBlock; i++ {
				fileName := fmt.Sprint("blocks_", i, ".ssz_snappy")
				blockFile, err := testutil.BazelFileBytes(testsFolderPath, folder.Name(), fileName)
				require.NoError(t, err)
				blockSSZ, err := snappy.Decode(nil /* dst */, blockFile)
				require.NoError(t, err, "Failed to decompress")
				block := &ethpb.SignedBeaconBlock{}
				require.NoError(t, block.UnmarshalSSZ(blockSSZ), "Failed to unmarshal")
				preforkBlocks = append(preforkBlocks, block)
			}
			t.Error(preforkBlocks[0].Block.StateRoot)
			for i := metaYaml.ForkBlock + 1; i < metaYaml.BlocksCount; i++ {
				fileName := fmt.Sprint("blocks_", i, ".ssz_snappy")
				blockFile, err := testutil.BazelFileBytes(testsFolderPath, folder.Name(), fileName)
				require.NoError(t, err)
				blockSSZ, err := snappy.Decode(nil /* dst */, blockFile)
				require.NoError(t, err, "Failed to decompress")
				block := &ethpb.SignedBeaconBlockAltair{}
				require.NoError(t, block.UnmarshalSSZ(blockSSZ), "Failed to unmarshal")
				postforkBlocks = append(postforkBlocks, block)
			}

			helpers.ClearCache()
			preBeaconStateFile, err := testutil.BazelFileBytes(testsFolderPath, folder.Name(), "pre.ssz_snappy")
			require.NoError(t, err)
			preBeaconStateSSZ, err := snappy.Decode(nil /* dst */, preBeaconStateFile)
			require.NoError(t, err, "Failed to decompress")
			beaconStateBase := &pb.BeaconState{}
			require.NoError(t, beaconStateBase.UnmarshalSSZ(preBeaconStateSSZ), "Failed to unmarshal")
			beaconState, err := stateV0.InitializeFromProto(beaconStateBase)
			require.NoError(t, err)
			t.Error(beaconState.Slot())
		})
	}
}
