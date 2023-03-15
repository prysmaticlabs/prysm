package epoch_processing

import (
	"context"
	"path"
	"testing"

	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/epoch"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
	"github.com/prysmaticlabs/prysm/v4/testing/spectest/utils"
)

// RunRegistryUpdatesTests executes "epoch_processing/registry_updates" tests.
func RunRegistryUpdatesTests(t *testing.T, config string) {
	require.NoError(t, utils.SetConfig(t, config))

	testFolders, testsFolderPath := utils.TestFolders(t, config, "phase0", "epoch_processing/registry_updates/pyspec_tests")
	if len(testFolders) == 0 {
		t.Fatalf("No test folders found for %s/%s/%s", config, "phase0", "epoch_processing/registry_updates/pyspec_tests")
	}
	for _, folder := range testFolders {
		t.Run(folder.Name(), func(t *testing.T) {
			folderPath := path.Join(testsFolderPath, folder.Name())
			// Important to clear cache for every test or else the old value of active validator count gets reused.
			helpers.ClearCache()
			RunEpochOperationTest(t, folderPath, processRegistryUpdatesWrapper)
		})
	}
}

func processRegistryUpdatesWrapper(_ *testing.T, state state.BeaconState) (state.BeaconState, error) {
	return epoch.ProcessRegistryUpdates(context.Background(), state)
}
