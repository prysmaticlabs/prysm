package spectest

import (
	"path"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/epoch"
	beaconstate "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/params/spectest"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func runFinalUpdatesTests(t *testing.T, config string) {
	if err := spectest.SetConfig(config); err != nil {
		t.Fatal(err)
	}

	testFolders, testsFolderPath := testutil.TestFolders(t, config, "epoch_processing/final_updates/pyspec_tests")
	for _, folder := range testFolders {
		t.Run(folder.Name(), func(t *testing.T) {
			folderPath := path.Join(testsFolderPath, folder.Name())
			testutil.RunEpochOperationTest(t, folderPath, processFinalUpdatesWrapper)
		})
	}
}

func processFinalUpdatesWrapper(t *testing.T, state *beaconstate.BeaconState) (*beaconstate.BeaconState, error) {
	state, err := epoch.ProcessFinalUpdates(state)
	if err != nil {
		t.Fatalf("could not process final updates: %v", err)
	}
	return state, nil
}
