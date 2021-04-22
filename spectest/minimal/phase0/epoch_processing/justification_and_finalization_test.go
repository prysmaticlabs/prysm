package epoch_processing

import (
	"testing"

	"github.com/prysmaticlabs/prysm/shared/params/spectest"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/prysmaticlabs/prysm/spectest/shared/phase0/epoch_processing"
)

func TestMinimal_Phase0_EpochProcessing_JustificationAndFinalization(t *testing.T) {
	config := "minimal"
	require.NoError(t, spectest.SetConfig(t, config))

	testPath := "epoch_processing/justification_and_finalization/pyspec_tests"
	testFolders, testsFolderPath := testutil.TestFolders(t, config, "phase0", testPath)
	epoch_processing.RunJustificationAndFinalizationTests(t, testFolders, testsFolderPath)
}
