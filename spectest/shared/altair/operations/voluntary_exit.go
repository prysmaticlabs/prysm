package operations

import (
	"context"
	"path"
	"testing"

	"github.com/golang/snappy"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	prysmv2 "github.com/prysmaticlabs/prysm/proto/prysm/v2"
	"github.com/prysmaticlabs/prysm/proto/prysm/v2/block"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/prysmaticlabs/prysm/spectest/utils"
)

func RunVoluntaryExitTest(t *testing.T, config string) {
	require.NoError(t, utils.SetConfig(t, config))
	testFolders, testsFolderPath := utils.TestFolders(t, config, "altair", "operations/voluntary_exit/pyspec_tests")
	for _, folder := range testFolders {
		t.Run(folder.Name(), func(t *testing.T) {
			folderPath := path.Join(testsFolderPath, folder.Name())
			exitFile, err := testutil.BazelFileBytes(folderPath, "voluntary_exit.ssz_snappy")
			require.NoError(t, err)
			exitSSZ, err := snappy.Decode(nil /* dst */, exitFile)
			require.NoError(t, err, "Failed to decompress")
			voluntaryExit := &ethpb.SignedVoluntaryExit{}
			require.NoError(t, voluntaryExit.UnmarshalSSZ(exitSSZ), "Failed to unmarshal")

			body := &prysmv2.BeaconBlockBody{VoluntaryExits: []*ethpb.SignedVoluntaryExit{voluntaryExit}}
			RunBlockOperationTest(t, folderPath, body, func(ctx context.Context, s state.BeaconState, b block.SignedBeaconBlock) (state.BeaconState, error) {
				return blocks.ProcessVoluntaryExits(ctx, s, b.Block().Body().VoluntaryExits())
			})
		})
	}
}
