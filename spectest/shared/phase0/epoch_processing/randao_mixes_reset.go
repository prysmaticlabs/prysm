package epoch_processing

import (
	"os"
	"path"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/epoch"
	iface "github.com/prysmaticlabs/prysm/beacon-chain/state/interface"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func RunRandaoMixesResetTests(t *testing.T, testFolders []os.FileInfo, testsFolderPath string) {
	for _, folder := range testFolders {
		t.Run(folder.Name(), func(t *testing.T) {
			folderPath := path.Join(testsFolderPath, folder.Name())
			testutil.RunEpochOperationTest(t, folderPath, processRandaoMixesResetWrapper)
		})
	}
}

func processRandaoMixesResetWrapper(t *testing.T, state iface.BeaconState) (iface.BeaconState, error) {
	state, err := epoch.ProcessRandaoMixesReset(state)
	require.NoError(t, err, "Could not process final updates")
	return state, nil
}
