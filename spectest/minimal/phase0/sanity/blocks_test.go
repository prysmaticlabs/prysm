package sanity

import (
	"testing"

	"github.com/prysmaticlabs/prysm/shared/params/spectest"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	sharedRunner "github.com/prysmaticlabs/prysm/spectest/shared/phase0/sanity"
)

func TestMinimal_Phase0_Sanity_Blocks(t *testing.T) {
	config := "minimal"
	require.NoError(t, spectest.SetConfig(t, config))
	testFolders, testsFolderPath := testutil.TestFolders(t, config, "phase0", "sanity/blocks/pyspec_tests")
	sharedRunner.RunBlockProcessingTest(t, testFolders, testsFolderPath)
}
