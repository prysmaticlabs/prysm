package operations

import (
	"testing"

	"github.com/prysmaticlabs/prysm/shared/params/spectest"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/prysmaticlabs/prysm/spectest/shared/phase0/operations"
)

func TestMainnet_Phase0_Operations_ProposerSlashing(t *testing.T) {
	config := "mainnet"
	require.NoError(t, spectest.SetConfig(t, config))
	testFolders, testsFolderPath := testutil.TestFolders(t, config, "phase0", "operations/proposer_slashing/pyspec_tests")
	operations.RunProposerSlashingTest(t, testFolders, testsFolderPath)
}
