package rewards

import (
	"fmt"
	"path"
	"testing"

	"github.com/golang/snappy"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/helpers"
	state_native "github.com/prysmaticlabs/prysm/v5/beacon-chain/state/state-native"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	"github.com/prysmaticlabs/prysm/v5/testing/spectest/shared/common/rewards"
	"github.com/prysmaticlabs/prysm/v5/testing/spectest/utils"
	"github.com/prysmaticlabs/prysm/v5/testing/util"
)

// RunPrecomputeRewardsAndPenaltiesTests executes "rewards/{basic, leak, random}" tests.
func RunPrecomputeRewardsAndPenaltiesTests(t *testing.T, config string) {
	require.NoError(t, utils.SetConfig(t, config))

	_, testsFolderPath := utils.TestFolders(t, config, "bellatrix", "rewards")
	testTypes, err := util.BazelListDirectories(testsFolderPath)
	require.NoError(t, err)

	if len(testTypes) == 0 {
		t.Fatalf("No test types found for %s", testsFolderPath)
	}

	for _, testType := range testTypes {
		testPath := fmt.Sprintf("rewards/%s/pyspec_tests", testType)
		testFolders, testsFolderPath := utils.TestFolders(t, config, "bellatrix", testPath)
		if len(testFolders) == 0 {
			t.Fatalf("No test folders found for %s/%s/%s", config, "bellatrix", testPath)
		}
		for _, folder := range testFolders {
			helpers.ClearCache()
			t.Run(fmt.Sprintf("%v/%v", testType, folder.Name()), func(t *testing.T) {
				folderPath := path.Join(testsFolderPath, folder.Name())
				runPrecomputeRewardsAndPenaltiesTest(t, folderPath)
			})
		}
	}
}

func runPrecomputeRewardsAndPenaltiesTest(t *testing.T, testFolderPath string) {
	preBeaconStateFile, err := util.BazelFileBytes(path.Join(testFolderPath, "pre.ssz_snappy"))
	require.NoError(t, err)
	preBeaconStateSSZ, err := snappy.Decode(nil /* dst */, preBeaconStateFile)
	require.NoError(t, err, "Failed to decompress")
	preBeaconStateBase := &ethpb.BeaconStateBellatrix{}
	require.NoError(t, preBeaconStateBase.UnmarshalSSZ(preBeaconStateSSZ), "Failed to unmarshal")
	preBeaconState, err := state_native.InitializeFromProtoBellatrix(preBeaconStateBase)
	require.NoError(t, err)
	rewards.PrecomputeRewardsAndPenalties(t, preBeaconState, testFolderPath)
}
