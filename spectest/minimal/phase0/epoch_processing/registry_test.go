package epoch_processing

import (
	"testing"

	"github.com/prysmaticlabs/prysm/shared/params/spectest"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/prysmaticlabs/prysm/spectest/shared/phase0/epoch_processing"
)

func TestMinimal_Phase0_EpochProcessing_ReseRegistryUpdates(t *testing.T) {
	config := "minimal"
	require.NoError(t, spectest.SetConfig(t, config))

	testFolders, testsFolderPath := testutil.TestFolders(t, config, "phase0", "epoch_processing/registry_updates/pyspec_tests")
	epoch_processing.RunRegistryUpdatesTests(t, testFolders, testsFolderPath)
}
