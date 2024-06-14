package genesis

import (
	"strconv"
	"strings"
	"testing"

	"github.com/golang/snappy"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/testing/assert"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
	"github.com/prysmaticlabs/prysm/v4/testing/spectest/utils"
	"github.com/prysmaticlabs/prysm/v4/testing/util"
)

func RunValidityTest(t *testing.T, config string) {
	require.NoError(t, utils.SetConfig(t, config))

	testFolders, testsFolderPath := utils.TestFolders(t, config, "phase0", "genesis/validity/pyspec_tests")
	if len(testFolders) == 0 {
		t.Fatalf("No test folders found for %s/%s/%s", config, "phase0", "genesis/validity/pyspec_tests")
	}
	for _, folder := range testFolders {
		t.Run(folder.Name(), func(t *testing.T) {
			genesisFile, err := util.BazelFileBytes(testsFolderPath, folder.Name(), "genesis.ssz_snappy")
			require.NoError(t, err)
			genesisSSZ, err := snappy.Decode(nil /* dst */, genesisFile)
			require.NoError(t, err)
			expectedResultFile, err := util.BazelFileBytes(testsFolderPath, folder.Name(), "is_valid.yaml")
			require.NoError(t, err)
			expected, err := strconv.ParseBool(strings.Split(string(expectedResultFile), "\n")[0])
			require.NoError(t, err)
			err = (&ethpb.BeaconState{}).UnmarshalSSZ(genesisSSZ)
			assert.Equal(t, expected, err == nil)
		})
	}
}
