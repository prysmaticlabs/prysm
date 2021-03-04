package spectest

import (
	"context"
	"path"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/epoch/precompute"
	iface "github.com/prysmaticlabs/prysm/beacon-chain/state/interface"
	"github.com/prysmaticlabs/prysm/shared/params/spectest"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func runJustificationAndFinalizationTests(t *testing.T, config string) {
	require.NoError(t, spectest.SetConfig(t, config))

	testPath := "epoch_processing/justification_and_finalization/pyspec_tests"
	testFolders, testsFolderPath := testutil.TestFolders(t, config, testPath)
	for _, folder := range testFolders {
		t.Run(folder.Name(), func(t *testing.T) {
			folderPath := path.Join(testsFolderPath, folder.Name())
			testutil.RunEpochOperationTest(t, folderPath, processJustificationAndFinalizationPrecomputeWrapper)
		})
	}
}

func processJustificationAndFinalizationPrecomputeWrapper(t *testing.T, st iface.BeaconState) (iface.BeaconState, error) {
	ctx := context.Background()
	vp, bp, err := precompute.New(ctx, st)
	require.NoError(t, err)
	_, bp, err = precompute.ProcessAttestations(ctx, st, vp, bp)
	require.NoError(t, err)

	st, err = precompute.ProcessJustificationAndFinalizationPreCompute(st, bp)
	require.NoError(t, err, "Could not process justification")

	return st, nil
}
