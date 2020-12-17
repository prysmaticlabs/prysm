package params

import (
	"io/ioutil"
	"path"
	"strings"
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestLoadConfigFileMainnet(t *testing.T) {
	assertVals := func(name string, c1, c2 *BeaconChainConfig) {
		//  Misc params.
		assert.Equal(t, c1.MaxCommitteesPerSlot, c2.MaxCommitteesPerSlot, "%s: MaxCommitteesPerSlot", name)
		assert.Equal(t, c1.TargetCommitteeSize, c2.TargetCommitteeSize, "%s: TargetCommitteeSize", name)
		assert.Equal(t, c1.MaxValidatorsPerCommittee, c2.MaxValidatorsPerCommittee, "%s: MaxValidatorsPerCommittee", name)
		assert.Equal(t, c1.MinPerEpochChurnLimit, c2.MinPerEpochChurnLimit, "%s: MinPerEpochChurnLimit", name)
		assert.Equal(t, c1.ChurnLimitQuotient, c2.ChurnLimitQuotient, "%s: ChurnLimitQuotient", name)
		assert.Equal(t, c1.ShuffleRoundCount, c2.ShuffleRoundCount, "%s: ShuffleRoundCount", name)
		assert.Equal(t, c1.MinGenesisActiveValidatorCount, c2.MinGenesisActiveValidatorCount, "%s: MinGenesisActiveValidatorCount", name)
		assert.Equal(t, c1.MinGenesisTime, c2.MinGenesisTime, "%s: MinGenesisTime", name)
		assert.Equal(t, c1.HysteresisQuotient, c2.HysteresisQuotient, "%s: HysteresisQuotient", name)
		assert.Equal(t, c1.HysteresisDownwardMultiplier, c2.HysteresisDownwardMultiplier, "%s: HysteresisDownwardMultiplier", name)
		assert.Equal(t, c1.HysteresisUpwardMultiplier, c2.HysteresisUpwardMultiplier, "%s: HysteresisUpwardMultiplier", name)

		// Fork Choice params.
		assert.Equal(t, c1.SafeSlotsToUpdateJustified, c2.SafeSlotsToUpdateJustified, "%s: SafeSlotsToUpdateJustified", name)

		// Validator params.
		assert.Equal(t, c1.Eth1FollowDistance, c2.Eth1FollowDistance, "%s: Eth1FollowDistance", name)
		assert.Equal(t, c1.TargetAggregatorsPerCommittee, c2.TargetAggregatorsPerCommittee, "%s: TargetAggregatorsPerCommittee", name)
		assert.Equal(t, c1.RandomSubnetsPerValidator, c2.RandomSubnetsPerValidator, "%s: RandomSubnetsPerValidator", name)
		assert.Equal(t, c1.EpochsPerRandomSubnetSubscription, c2.EpochsPerRandomSubnetSubscription, "%s: EpochsPerRandomSubnetSubscription", name)
		assert.Equal(t, c1.SecondsPerETH1Block, c2.SecondsPerETH1Block, "%s: SecondsPerETH1Block", name)

		// Deposit contract.
		assert.Equal(t, c1.DepositChainID, c2.DepositChainID, "%s: DepositChainID", name)
		assert.Equal(t, c1.DepositNetworkID, c2.DepositNetworkID, "%s: DepositNetworkID", name)
		assert.Equal(t, c1.DepositContractAddress, c2.DepositContractAddress, "%s: DepositContractAddress", name)
	}

	t.Run("mainnet", func(t *testing.T) {
		mainnetConfigFile := ConfigFilePath(t, "mainnet")
		LoadChainConfigFile(mainnetConfigFile)
		assertVals("mainnet", MainnetConfig(), BeaconConfig())
	})

	t.Run("minimal", func(t *testing.T) {
		minimalConfigFile := ConfigFilePath(t, "minimal")
		LoadChainConfigFile(minimalConfigFile)
		assertVals("minimal", MinimalSpecConfig(), BeaconConfig())
	})
}

func TestLoadConfigFile_OverwriteCorrectly(t *testing.T) {
	file, err := ioutil.TempFile("", "")
	require.NoError(t, err)
	// Set current config to minimal config
	OverrideBeaconConfig(MinimalSpecConfig())

	// load empty config file, so that it defaults to mainnet values
	LoadChainConfigFile(file.Name())
	if BeaconConfig().MinGenesisTime != MainnetConfig().MinGenesisTime {
		t.Errorf("Expected MinGenesisTime to be set to mainnet value: %d found: %d",
			MainnetConfig().MinGenesisTime,
			BeaconConfig().MinGenesisTime)
	}
	if BeaconConfig().SlotsPerEpoch != MainnetConfig().SlotsPerEpoch {
		t.Errorf("Expected SlotsPerEpoch to be set to mainnet value: %d found: %d",
			MainnetConfig().SlotsPerEpoch,
			BeaconConfig().SlotsPerEpoch)
	}
}

func Test_replaceHexStringWithYAMLFormat(t *testing.T) {

	testLines := []struct {
		line   string
		wanted string
	}{
		{
			line:   "ONE_BYTE: 0x41",
			wanted: "ONE_BYTE: 65\n",
		},
		{
			line:   "FOUR_BYTES: 0x41414141",
			wanted: "FOUR_BYTES: \n- 65\n- 65\n- 65\n- 65\n",
		},
		{
			line:   "THREE_BYTES: 0x414141",
			wanted: "THREE_BYTES: \n- 65\n- 65\n- 65\n- 0\n",
		},
		{
			line:   "EIGHT_BYTES: 0x4141414141414141",
			wanted: "EIGHT_BYTES: \n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n",
		},
		{
			line: "SIXTEEN_BYTES: 0x41414141414141414141414141414141",
			wanted: "SIXTEEN_BYTES: \n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n" +
				"- 65\n- 65\n- 65\n- 65\n",
		},
		{
			line: "TWENTY_BYTES: 0x4141414141414141414141414141414141414141",
			wanted: "TWENTY_BYTES: \n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n" +
				"- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n",
		},
		{
			line: "THIRTY_TWO_BYTES: 0x4141414141414141414141414141414141414141414141414141414141414141",
			wanted: "THIRTY_TWO_BYTES: \n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n" +
				"- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n" +
				"- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n",
		},
		{
			line: "FORTY_EIGHT_BYTES: 0x41414141414141414141414141414141414141414141414141414141414141414141" +
				"4141414141414141414141414141",
			wanted: "FORTY_EIGHT_BYTES: \n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n" +
				"- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n" +
				"- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n" +
				"- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n",
		},
		{
			line: "NINETY_SIX_BYTES: 0x414141414141414141414141414141414141414141414141414141414141414141414141" +
				"4141414141414141414141414141414141414141414141414141414141414141414141414141414141414141414141" +
				"41414141414141414141414141",
			wanted: "NINETY_SIX_BYTES: \n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n" +
				"- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n" +
				"- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n" +
				"- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n" +
				"- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n" +
				"- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n" +
				"- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n",
		},
	}
	for _, line := range testLines {
		parts := replaceHexStringWithYAMLFormat(line.line)
		res := strings.Join(parts, "\n")

		if res != line.wanted {
			t.Errorf("expected conversion to be: %v got: %v", line.wanted, res)
		}
	}
}

// ConfigFilePath sets the proper config and returns the relevant
// config file path from eth2-spec-tests directory.
func ConfigFilePath(t *testing.T, config string) string {
	configFolderPath := path.Join("tests", config)
	filepath, err := bazel.Runfile(configFolderPath)
	require.NoError(t, err)
	configFilePath := path.Join(filepath, "config", "phase0.yaml")
	return configFilePath
}
