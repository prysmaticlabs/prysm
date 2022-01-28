package params_test

import (
	"path"
	"path/filepath"
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/io/file"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func testnetConfigFilePath(t *testing.T, network string) string {
	filepath, err := bazel.Runfile("external/eth2_networks")
	require.NoError(t, err)
	configFilePath := path.Join(filepath, "shared", network, "config.yaml")
	return configFilePath
}

func TestE2EConfigParity(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	testDir := bazel.TestTmpDir()
	yamlDir := filepath.Join(testDir, "config.yaml")

	testCfg := params.E2EMainnetTestConfig()
	yamlObj := params.E2EMainnetConfigYaml()
	assert.NoError(t, file.WriteFile(yamlDir, yamlObj))

	params.LoadChainConfigFile(yamlDir)
	assert.DeepEqual(t, params.BeaconConfig(), testCfg)
}
