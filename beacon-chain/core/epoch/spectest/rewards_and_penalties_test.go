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
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func runRewardsAndPenaltiesTests(t *testing.T, config string) {
	require.NoError(t, spectest.SetConfig(t, config))

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
	require.NoError(t, err)
	vp, bp, err = precompute.ProcessAttestations(ctx, state, vp, bp)
	require.NoError(t, err)

	state, err = precompute.ProcessRewardsAndPenaltiesPrecompute(state, bp, vp)
	require.NoError(t, err, "Could not process reward")

	return state, nil
}
