package params

import (
	"io/ioutil"
	"path"
	"reflect"
	"strings"
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

var presets = []string{
	"phase0",
	"altair",
	//"custody_game",
	//"merge",
	//"sharding",
}

func TestLoadConfigFile(t *testing.T) {
	assertVals := func(name string, c1, c2 *BeaconChainConfig) {
		// Compare all fields with struct tag spec:"true".
		tp := reflect.TypeOf(*c1)
		for i := 0; i < tp.NumField(); i++ {
			f := tp.Field(i)
			if v, ok := f.Tag.Lookup("spec"); ok && v == "true" {
				a, b := reflect.ValueOf(*c1).Field(i), reflect.ValueOf(*c2).Field(i)
				if a.CanInterface() {
					assert.DeepEqual(t, a.Interface(), b.Interface(), "%sConfig.%s", name, f.Name)
				} else {
					t.Fatal("bad")
				}
			}
		}
	}

	t.Run("mainnet", func(t *testing.T) {
		files := []string{}
		files = append(files, configFilePath(t, "mainnet"))
		for _, fork := range presets {
			files = append(files, presetsFilePath(t, "mainnet", fork))
		}
		LoadChainConfigFiles(files)
		assertVals("mainnet", MainnetConfig(), BeaconConfig())
	})

	t.Run("minimal", func(t *testing.T) {
		files := []string{}
		files = append(files, configFilePath(t, "minimal"))
		for _, fork := range presets {
			files = append(files, presetsFilePath(t, "minimal", fork))
		}
		LoadChainConfigFiles(files)
		assertVals("minimal", MinimalSpecConfig(), BeaconConfig())
	})
}

func TestLoadConfigFile_OverwriteCorrectly(t *testing.T) {
	file, err := ioutil.TempFile("", "")
	require.NoError(t, err)
	// Set current config to minimal config
	OverrideBeaconConfig(MinimalSpecConfig())

	// load empty config file, so that it defaults to mainnet values
	LoadChainConfigFiles([]string{file.Name()})
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

// configFilePath sets the proper config and returns the relevant
// config file path from eth2-spec-tests directory.
func configFilePath(t *testing.T, config string) string {
	filepath, err := bazel.Runfile("external/eth2_spec")
	require.NoError(t, err)
	configFilePath := path.Join(filepath, "configs", config+".yaml")
	return configFilePath
}

// presetsFilePath sets the proper preset and returns the relevant
// preset file path from eth2-spec-tests directory.
func presetsFilePath(t *testing.T, config, fork string) string {
	filepath, err := bazel.Runfile("external/eth2_spec")
	require.NoError(t, err)
	configFilePath := path.Join(filepath, "presets", config, fork+".yaml")
	return configFilePath
}
