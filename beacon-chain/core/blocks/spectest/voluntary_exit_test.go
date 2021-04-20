package spectest

import (
	"context"
	"path"
	"testing"

	"github.com/golang/snappy"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	iface "github.com/prysmaticlabs/prysm/beacon-chain/state/interface"
	"github.com/prysmaticlabs/prysm/shared/params/spectest"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func runVoluntaryExitTest(t *testing.T, config string) {
	require.NoError(t, spectest.SetConfig(t, config))

	testFolders, testsFolderPath := testutil.TestFolders(t, config, "phase0", "operations/voluntary_exit/pyspec_tests")
	for _, folder := range testFolders {
		t.Run(folder.Name(), func(t *testing.T) {
			folderPath := path.Join(testsFolderPath, folder.Name())
			exitFile, err := testutil.BazelFileBytes(folderPath, "voluntary_exit.ssz_snappy")
			require.NoError(t, err)
			exitSSZ, err := snappy.Decode(nil /* dst */, exitFile)
			require.NoError(t, err, "Failed to decompress")
			voluntaryExit := &ethpb.SignedVoluntaryExit{}
			require.NoError(t, voluntaryExit.UnmarshalSSZ(exitSSZ), "Failed to unmarshal")

			body := &ethpb.BeaconBlockBody{VoluntaryExits: []*ethpb.SignedVoluntaryExit{voluntaryExit}}
			testutil.RunBlockOperationTest(t, folderPath, body, func(ctx context.Context, s iface.BeaconState, b *ethpb.SignedBeaconBlock) (iface.BeaconState, error) {
				return blocks.ProcessVoluntaryExits(ctx, s, b.Block.Body.VoluntaryExits)
			})
		})
	}
}
