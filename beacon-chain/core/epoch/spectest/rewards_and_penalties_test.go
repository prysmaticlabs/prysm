package spectest

import (
	"context"
	"path"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/epoch/precompute"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/params/spectest"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func runRewardsAndPenaltiesTests(t *testing.T, config string) {
	if err := spectest.SetConfig(t, config); err != nil {
		t.Fatal(err)
	}

	testPath := "epoch_processing/rewards_and_penalties/pyspec_tests"
	testFolders, testsFolderPath := testutil.TestFolders(t, config, testPath)
	for _, folder := range testFolders {
		helpers.ClearCache()
		t.Run(folder.Name(), func(t *testing.T) {
			folderPath := path.Join(testsFolderPath, folder.Name())
			testutil.RunEpochOperationTest(t, folderPath, processRewardsAndPenaltiesPrecomputeWrapper)
		})
	}
}

func processRewardsAndPenaltiesPrecomputeWrapper(t *testing.T, state *state.BeaconState) (*state.BeaconState, error) {
	ctx := context.Background()
	vp, bp, err := precompute.New(ctx, state)
	if err != nil {
		t.Fatal(err)
	}
	vp, bp, err = precompute.ProcessAttestations(ctx, state, vp, bp)
	if err != nil {
		t.Fatal(err)
	}

	state, err = precompute.ProcessRewardsAndPenaltiesPrecompute(state, bp, vp)
	if err != nil {
		t.Fatalf("could not process reward: %v", err)
	}

	return state, nil
}
