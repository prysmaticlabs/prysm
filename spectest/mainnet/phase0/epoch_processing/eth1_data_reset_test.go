package epoch_processing

import (
	"testing"

	"github.com/prysmaticlabs/prysm/shared/params/spectest"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/prysmaticlabs/prysm/spectest/shared/phase0/epoch_processing"
)

func TestMainnet_Phase0_EpochProcessing_Eth1DataReset(t *testing.T) {
	config := "mainnet"
	require.NoError(t, spectest.SetConfig(t, config))

	testFolders, testsFolderPath := testutil.TestFolders(t, config, "phase0", "epoch_processing/eth1_data_reset/pyspec_tests")
	epoch_processing.RunEth1DataResetTests(t, testFolders, testsFolderPath)
}
