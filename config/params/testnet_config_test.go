package params_test

import (
	"path"
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func testnetConfigFilePath(t *testing.T, network string) string {
	filepath, err := bazel.Runfile("external/eth2_networks")
	require.NoError(t, err)
	configFilePath := path.Join(filepath, "shared", network, "config.yaml")
	return configFilePath
}
