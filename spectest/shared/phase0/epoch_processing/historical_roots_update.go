package epoch_processing

import (
	"path"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/epoch"
	iface "github.com/prysmaticlabs/prysm/beacon-chain/state/interface"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/prysmaticlabs/prysm/spectest/utils"
)

func RunHistoricalRootsUpdateTests(t *testing.T, config string) {
	require.NoError(t, utils.SetConfig(t, config))

	testFolders, testsFolderPath := testutil.TestFolders(t, config, "phase0", "epoch_processing/historical_roots_update/pyspec_tests")
	for _, folder := range testFolders {
		t.Run(folder.Name(), func(t *testing.T) {
			folderPath := path.Join(testsFolderPath, folder.Name())
			testutil.RunEpochOperationTest(t, folderPath, processHistoricalRootsUpdateWrapper)
		})
	}
}

func processHistoricalRootsUpdateWrapper(t *testing.T, state iface.BeaconState) (iface.BeaconState, error) {
	state, err := epoch.ProcessHistoricalRootsUpdate(state)
	require.NoError(t, err, "Could not process final updates")
	return state, nil
}
