package epoch_processing

import (
	"path"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/altair"
	"github.com/prysmaticlabs/prysm/beacon-chain/state-native"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/spectest/utils"
)

// RunParticipationFlagUpdatesTests executes "epoch_processing/participation_flag_updates" tests.
func RunParticipationFlagUpdatesTests(t *testing.T, config string) {
	require.NoError(t, utils.SetConfig(t, config))

	testFolders, testsFolderPath := utils.TestFolders(t, config, "bellatrix", "epoch_processing/participation_flag_updates/pyspec_tests")
	for _, folder := range testFolders {
		t.Run(folder.Name(), func(t *testing.T) {
			folderPath := path.Join(testsFolderPath, folder.Name())
			RunEpochOperationTest(t, folderPath, processParticipationFlagUpdatesWrapper)
		})
	}
}

func processParticipationFlagUpdatesWrapper(t *testing.T, state state.BeaconState) (state.BeaconState, error) {
	state, err := altair.ProcessParticipationFlagUpdates(state)
	require.NoError(t, err, "Could not process participation flag update")
	return state, nil
}
