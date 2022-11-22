package operations

import (
	"context"
	"path"
	"testing"

	"github.com/golang/snappy"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/testing/spectest/utils"
	"github.com/prysmaticlabs/prysm/v3/testing/util"
)

func RunBLSToExecutionChangeTest(t *testing.T, config string) {
	require.NoError(t, utils.SetConfig(t, config))
	testFolders, testsFolderPath := utils.TestFolders(t, config, "eip4844", "operations/bls_to_execution_change/pyspec_tests")
	for _, folder := range testFolders {
		t.Run(folder.Name(), func(t *testing.T) {
			folderPath := path.Join(testsFolderPath, folder.Name())
			changeFile, err := util.BazelFileBytes(folderPath, "address_change.ssz_snappy")
			require.NoError(t, err)
			changeSSZ, err := snappy.Decode(nil /* dst */, changeFile)
			require.NoError(t, err, "Failed to decompress")
			change := &ethpb.SignedBLSToExecutionChange{}
			require.NoError(t, change.UnmarshalSSZ(changeSSZ), "Failed to unmarshal")

			body := &ethpb.BeaconBlockBodyCapella{
				BlsToExecutionChanges: []*ethpb.SignedBLSToExecutionChange{change},
			}
			RunBlockOperationTest(t, folderPath, body, func(_ context.Context, s state.BeaconState, b interfaces.SignedBeaconBlock) (state.BeaconState, error) {
				return blocks.ProcessBLSToExecutionChanges(s, b)
			})
		})
	}
}
