package spectest

import (
	"context"
	"path"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/epoch"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/epoch/precompute"
	beaconstate "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/params/spectest"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func runSlashingsTests(t *testing.T, config string) {
	if err := spectest.SetConfig(t, config); err != nil {
		t.Fatal(err)
	}

	testFolders, testsFolderPath := testutil.TestFolders(t, config, "epoch_processing/slashings/pyspec_tests")
	for _, folder := range testFolders {
		t.Run(folder.Name(), func(t *testing.T) {
			folderPath := path.Join(testsFolderPath, folder.Name())
			testutil.RunEpochOperationTest(t, folderPath, processSlashingsWrapper)
			testutil.RunEpochOperationTest(t, folderPath, processSlashingsPrecomputeWrapper)
		})
	}
}

func processSlashingsWrapper(t *testing.T, state *beaconstate.BeaconState) (*beaconstate.BeaconState, error) {
	state, err := epoch.ProcessSlashings(state)
	if err != nil {
		t.Fatalf("could not process slashings: %v", err)
	}
	return state, nil
}

func processSlashingsPrecomputeWrapper(t *testing.T, state *beaconstate.BeaconState) (*beaconstate.BeaconState, error) {
	ctx := context.Background()
	vp, bp, err := precompute.New(ctx, state)
	if err != nil {
		t.Fatal(err)
	}
	_, bp, err = precompute.ProcessAttestations(ctx, state, vp, bp)
	if err != nil {
		t.Fatal(err)
	}

	return state, precompute.ProcessSlashingsPrecompute(state, bp)
}
