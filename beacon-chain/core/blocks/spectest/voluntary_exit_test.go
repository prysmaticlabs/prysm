package spectest

import (
	"path"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/shared/params/spectest"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func runVoluntaryExitTest(t *testing.T, config string) {
	require.NoError(t, spectest.SetConfig(t, config))

	testFolders, testsFolderPath := testutil.TestFolders(t, config, "operations/voluntary_exit/pyspec_tests")
	for _, folder := range testFolders {
		t.Run(folder.Name(), func(t *testing.T) {
			folderPath := path.Join(testsFolderPath, folder.Name())
			exitFile, err := testutil.BazelFileBytes(folderPath, "voluntary_exit.ssz")
			require.NoError(t, err)
			voluntaryExit := &ethpb.SignedVoluntaryExit{}
			require.NoError(t, voluntaryExit.UnmarshalSSZ(exitFile), "Failed to unmarshal")

			body := &ethpb.BeaconBlockBody{VoluntaryExits: []*ethpb.SignedVoluntaryExit{voluntaryExit}}
			testutil.RunBlockOperationTest(t, folderPath, body, blocks.ProcessVoluntaryExits)
		})
	}
}
