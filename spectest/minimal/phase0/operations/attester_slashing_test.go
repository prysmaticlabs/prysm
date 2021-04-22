package operations

import (
	"testing"

	"github.com/prysmaticlabs/prysm/shared/params/spectest"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/prysmaticlabs/prysm/spectest/shared/phase0/operations"
)

func TestMinimal_Phase0_Operations_AttesterSlashing(t *testing.T) {
	config := "minimal"
	require.NoError(t, spectest.SetConfig(t, config))
	testFolders, testsFolderPath := testutil.TestFolders(t, config, "phase0", "operations/attester_slashing/pyspec_tests")
	operations.RunAttesterSlashingTest(t, testFolders, testsFolderPath)
}
